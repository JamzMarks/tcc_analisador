package services

import (
	"context"
	"fmt"
	"injector/types"
	"log"

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

func (s *AnalisadorService) InjectBuffer(data []types.EdgeData) {
	ctx := context.Background()
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: "neo4j",
	})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			UNWIND $batch AS row
			MATCH (d:Device {deviceId: row.deviceId})
			SET d.flow = row.flow,
				d.confiability = row.confiability,
				d.lastUpdate = row.ts
			RETURN d.deviceId
		`

		batch := make([]map[string]any, len(data))
		for i, e := range data {
			batch[i] = map[string]any{
				"deviceId":     e.DeviceId,
				"flow":         e.Data.Flow,
				"confiability": e.Data.Confiability,
				"ts":           e.TS,
			}
		}

		params := map[string]any{"batch": batch}

		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		for result.Next(ctx) {
			log.Printf("Flow atualizado para device: %s", result.Record().Values[0])
		}

		return nil, result.Err()
	})

	if err != nil {
		log.Printf("Erro ao injetar buffer: %v", err)
	}
}
