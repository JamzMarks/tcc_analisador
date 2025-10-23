package services

import (
	"context"
	"fmt"
	"log"
	"math"
	"tcc_analisador/types"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type AnalisadorService struct {
	uri      string
	username string
	password string
	driver   neo4j.DriverWithContext
}

func NewGraphService(uri, username, password string) *AnalisadorService {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))

	if err != nil {
		log.Fatalf("Erro ao criar driver Neo4j: %v", err)
	}
	return &AnalisadorService{
		uri:      uri,
		username: username,
		password: password,
		driver:   driver,
	}
}

func (s *AnalisadorService) Close() {
	if s.driver != nil {
		s.driver.Close(context.Background())
	}
}

func (s *AnalisadorService) TestConnection() {
	ctx := context.Background()
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	result, err := session.Run(ctx, "RETURN 'Conexão OK!' AS message", nil)
	if err != nil {
		log.Fatalf("Erro ao executar query: %v", err)
	}

	if result.Next(ctx) {
		fmt.Println(result.Record().Values[0])
	} else if err = result.Err(); err != nil {
		log.Fatalf("Erro ao ler resultado: %v", err)
	}
}

func (s *AnalisadorService) AnalyzeWays() ([]types.WayFlowResult, error) {
	ctx := context.Background()
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	query := `
		MATCH (d:Device)-[:FEED_DATA_ON]->(w:OSMWay)
		RETURN 
			elementId(w) AS wayId,
			w.highway AS roadType,
			round(avg(d.flow * d.confiability), 5) AS avgFlowByReliability
	`

	result, err := session.Run(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao executar análise: %w", err)
	}

	var ways []types.WayFlowResult
	for result.Next(ctx) {
		record := result.Record()

		wayID := record.Values[0].(string)
		roadType := record.Values[1].(string)
		avgFlow := record.Values[2].(float64)

		priority := types.GetRoadPriority(roadType)
		normalizedFlow := avgFlow * priority
		log.Printf("Valor normalizado: %.5f", normalizedFlow)
		ways = append(ways, types.WayFlowResult{
			WayID:                wayID,
			RoadType:             roadType,
			AvgFlowByReliability: normalizedFlow,
		})
	}

	if err = result.Err(); err != nil {
		return nil, fmt.Errorf("erro ao ler resultados: %w", err)
	}

	waysData := make([]map[string]any, len(ways))
	for i, way := range ways {
		rawPriority := types.GetRoadPriority(way.RoadType) * way.AvgFlowByReliability * 100
		priority := math.Round(rawPriority*100) / 100
		waysData[i] = map[string]any{
			"wayId":    way.WayID,
			"priority": priority,
		}
	}

	_, err = session.Run(ctx, `
		UNWIND $ways AS w
		MATCH (way:OSMWay)
		WHERE elementId(way) = w.wayId
		SET way.priority = w.priority
	`, map[string]any{
		"ways": waysData,
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao atualizar prioridades: %w", err)
	}

	return ways, nil
}
