package main

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)


// DeviceEvent representa uma nova leitura de fluxo vinda de um device
type DeviceEvent struct {
	EdgeID    string    // id da aresta CONNECT_TO
	DeviceID  string    // id do device que enviou
	Flow      float64   // [0..1]
	Timestamp time.Time // momento da medição
}

// Edge representa uma aresta CONNECT_TO entre 2 OSMNodes e referência para a OSMWay
type Edge struct {
	ID       string
	WayID    string  // qual OSMWay essa aresta pertence
	Flow     float64 // último valor
	DeviceID string  // qual device atualizou por último (opcional)
	Updated  time.Time
	mu       sync.RWMutex
}

// WayMetrics são as métricas agregadas mantidas por OSMWay
type WayMetrics struct {
	WayID        string
	Highway      string  // e.g. "primary", "secondary"
	AvgFlow      float64
	MinFlow      float64
	MaxFlow      float64
	DevicesCount int
	LastUpdated  time.Time
	Priority     float64 // 0..100
	mu           sync.RWMutex
}

// -------------------------------
// Interface para persistência / graph DB
// -------------------------------

// GraphStore permite integração com banco de grafos/DB;
// implementamos uma versão in-memory abaixo e você pode trocar por Neo4j/Arango/etc.
type GraphStore interface {
	GetEdge(ctx context.Context, edgeID string) (*Edge, error)
	UpsertEdge(ctx context.Context, e *Edge) error
	GetWayMetrics(ctx context.Context, wayID string) (*WayMetrics, error)
	UpsertWayMetrics(ctx context.Context, wm *WayMetrics) error
	ListEdgesByWay(ctx context.Context, wayID string) ([]*Edge, error)
	GetWayHighway(ctx context.Context, wayID string) (string, error)
}

// -------------------------------
// Implementação in-memory simples
// -------------------------------

type MemStore struct {
	edges map[string]*Edge
	ways  map[string]*WayMetrics
	mu    sync.RWMutex
}

func NewMemStore() *MemStore {
	return &MemStore{
		edges: make(map[string]*Edge),
		ways:  make(map[string]*WayMetrics),
	}
}

func (m *MemStore) GetEdge(ctx context.Context, edgeID string) (*Edge, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.edges[edgeID]
	if !ok {
		return nil, fmt.Errorf("edge not found: %s", edgeID)
	}
	return e, nil
}

func (m *MemStore) UpsertEdge(ctx context.Context, e *Edge) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.edges[e.ID] = e
	return nil
}

func (m *MemStore) GetWayMetrics(ctx context.Context, wayID string) (*WayMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	w, ok := m.ways[wayID]
	if !ok {
		return nil, fmt.Errorf("way not found: %s", wayID)
	}
	return w, nil
}

func (m *MemStore) UpsertWayMetrics(ctx context.Context, wm *WayMetrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ways[wm.WayID] = wm
	return nil
}

func (m *MemStore) ListEdgesByWay(ctx context.Context, wayID string) ([]*Edge, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := []*Edge{}
	for _, e := range m.edges {
		if e.WayID == wayID {
			res = append(res, e)
		}
	}
	return res, nil
}

// GetWayHighway retorna o tipo highway com base em WayMetrics salvo (simplificação)
func (m *MemStore) GetWayHighway(ctx context.Context, wayID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	w, ok := m.ways[wayID]
	if !ok {
		return "", fmt.Errorf("way not found: %s", wayID)
	}
	return w.Highway, nil
}

// -------------------------------
// Analyzer
// -------------------------------

// Config contém parâmetros para cálculo de prioridade
type Config struct {
	Alpha            float64       // peso do fluxo vs highway (0..1). maior = fluxo pesa mais
	MinSamplesToCalc int           // quantos edges precisamos para confiar no avg
	DecayWindow      time.Duration // janela para decaimento temporal (opcional)
	HighwayWeights   map[string]float64
}

func DefaultConfig() *Config {
	return &Config{
		Alpha:            0.7,
		MinSamplesToCalc: 1,
		DecayWindow:      5 * time.Minute,
		HighwayWeights: map[string]float64{
			"motorway":       	100,
			"trunk":          	90,
			"primary":        	80,
			"secondary":      	60,
			"tertiary":       	40,
			"primary_link": 	70,
			"secondary_link": 	60,
			"tertiary_link": 	50,
			"residential":    	20,
		},
	}
}

// Analyzer processa eventos e atualiza métricas
type Analyzer struct {
	store  GraphStore
	cfg    *Config
	evCh   chan DeviceEvent
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewAnalyzer(store GraphStore, cfg *Config) *Analyzer {
	cctx, cancel := context.WithCancel(context.Background())
	return &Analyzer{
		store:  store,
		cfg:    cfg,
		evCh:   make(chan DeviceEvent, 1024),
		ctx:    cctx,
		cancel: cancel,
	}
}

func (a *Analyzer) Start(workers int) {
	for i := 0; i < workers; i++ {
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			for {
				select {
				case ev := <-a.evCh:
					a.handleEvent(ev)
				case <-a.ctx.Done():
					return
				}
			}
		}()
	}
}

func (a *Analyzer) Stop() {
	a.cancel()
	a.wg.Wait()
}

func (a *Analyzer) PushEvent(ev DeviceEvent) {
	select {
	case a.evCh <- ev:
	default:
		// canal cheio — estratégia: drop, log ou blocking. Aqui simples drop com aviso.
		fmt.Println("warning: event channel full, dropping event")
	}
}

// handleEvent atualiza aresta e recalcula métricas da way correspondente
func (a *Analyzer) handleEvent(ev DeviceEvent) {
	ctx := a.ctx
	// 1) obter aresta (ou criar)
	e, err := a.store.GetEdge(ctx, ev.EdgeID)
	if err != nil {
		// criar aresta mínima — na prática, seu DB já teria esse mapeamento
		e = &Edge{
			ID:      ev.EdgeID,
			WayID:   inferWayID(ev.EdgeID), // placeholder: sua implementação
			Flow:    ev.Flow,
			Updated: ev.Timestamp,
		}
	}
	// atualizar aresta
	e.mu.Lock()
	e.Flow = ev.Flow
	e.DeviceID = ev.DeviceID
	e.Updated = ev.Timestamp
	e.mu.Unlock()
	_ = a.store.UpsertEdge(ctx, e)

	// 2) recalcular métricas da way — leitura de todas as arestas da way
	wayID := e.WayID
	edges, err := a.store.ListEdgesByWay(ctx, wayID)
	if err != nil {
		fmt.Println("error listing edges: ", err)
		return
	}

	// agregação simples: avg, min, max, unique devices count
	var sum float64
	var min = math.Inf(1)
	var max = math.Inf(-1)
	devices := make(map[string]struct{})
	count := 0
	now := time.Now()

	for _, ed := range edges {
		ed.mu.RLock()
		f := ed.Flow
		id := ed.DeviceID
		updated := ed.Updated
		ed.mu.RUnlock()

		// aplicar decaimento simples: se leitura muito antiga, ignorar ou pesar menos
		age := now.Sub(updated)
		if age > a.cfg.DecayWindow {
			// ignorar leitura muito antiga
			continue
		}

		sum += f
		if f < min {
			min = f
		}
		if f > max {
			max = f
		}
		if id != "" {
			devices[id] = struct{}{}
		}
		count++
	}

	if count == 0 {
		// sem dados úteis — não alteramos a way
		return
	}

	avg := sum / float64(count)
	devicesCount := len(devices)

	// 3) obter ou criar WayMetrics
	wm, err := a.store.GetWayMetrics(ctx, wayID)
	if err != nil {
		// criar conjunto mínimo
		hw, _ := a.store.GetWayHighway(ctx, wayID)
		if hw == "" {
			hw = "residential"
		}
		wm = &WayMetrics{
			WayID:        wayID,
			Highway:      hw,
			AvgFlow:      avg,
			MinFlow:      min,
			MaxFlow:      max,
			DevicesCount: devicesCount,
			LastUpdated:  now,
		}
	} else {
		wm.mu.Lock()
		wm.AvgFlow = avg
		wm.MinFlow = min
		wm.MaxFlow = max
		wm.DevicesCount = devicesCount
		wm.LastUpdated = now
		wm.mu.Unlock()
	}

	// 4) calcular prioridade
	priority := a.computePriority(wm)
	wm.mu.Lock()
	wm.Priority = priority
	wm.mu.Unlock()

	_ = a.store.UpsertWayMetrics(ctx, wm)

	// opcional: emitir evento para downstream (e.g., alertas, recalculo de rotas)
	fmt.Printf("way %s updated: avg=%.5f min=%.5f max=%.5f devices=%d priority=%.2f\n",
		wm.WayID, avg, min, max, devicesCount, priority)
}

// computePriority combina avg/min flows e peso do highway em 0..100
func (a *Analyzer) computePriority(wm *WayMetrics) float64 {
	wm.mu.RLock()
	avg := wm.AvgFlow
	min := wm.MinFlow
	hw := wm.Highway
	devices := wm.DevicesCount
	wm.mu.RUnlock()

	// P_fluxo baseado em min e avg — min tem mais peso para punir gargalos
	pMin := (1 - min) * 100
	pAvg := (1 - avg) * 100

	// highway weight normalizado 0..100 (já na config)
	hwWeight, ok := a.cfg.HighwayWeights[hw]
	if !ok {
		hwWeight = 20 // default
	}

	// combinar: min tem peso maior, avg complementa, highway entra como ajuste
	pFlux := 0.6*pMin + 0.4*pAvg // 0..100

	alpha := a.cfg.Alpha
	// prioridade final: combinação linear
	priority := alpha*pFlux + (1-alpha)*hwWeight

	// ajuste por devices_count (se muitos devices, pode aumentar confiança)
	if devices >= 3 {
		// boost leve se houver múltiplas leituras concordantes — saturado a +5
		priority = math.Min(100, priority+5)
	}

	// garantir limites
t	if priority < 0 {
		priority = 0
	}
	if priority > 100 {
		priority = 100
	}
	return priority
}

// inferWayID é placeholder — no seu sistema você tem esse mapeamento já
func inferWayID(edgeID string) string {
	// ex: edgeID = "way_123_edge_5" -> way_123
	// implementação simplificada:
	var wayID string
	for i := 0; i < len(edgeID); i++ {
		if edgeID[i] == '_' {
			wayID = edgeID[:i]
			break
		}
	}
	if wayID == "" {
		return "way_unknown"
	}
	return wayID
}

// -------------------------------
// Exemplo main: simula eventos
// -------------------------------

func main() {
	store := NewMemStore()

	// criar algumas edges
	store.UpsertEdge(context.Background(), &Edge{ID: "wayA_e1", WayID: "wayA", Flow: 0.9, Updated: time.Now()})
	store.UpsertEdge(context.Background(), &Edge{ID: "wayA_e2", WayID: "wayA", Flow: 0.8, Updated: time.Now()})
	store.UpsertEdge(context.Background(), &Edge{ID: "wayB_e1", WayID: "wayB", Flow: 0.2, Updated: time.Now()})

	cfg := DefaultConfig()
	an := NewAnalyzer(store, cfg)
	an.Start(4)

	// simular eventos
	an.PushEvent(DeviceEvent{EdgeID: "wayA_e1", DeviceID: "dev1", Flow: 0.85, Timestamp: time.Now()})
	an.PushEvent(DeviceEvent{EdgeID: "wayB_e1", DeviceID: "dev2", Flow: 0.12, Timestamp: time.Now()})
	an.PushEvent(DeviceEvent{EdgeID: "wayA_e2", DeviceID: "dev3", Flow: 0.05, Timestamp: time.Now()})

	// deixar processar
	time.Sleep(500 * time.Millisecond)
	an.Stop()
}
