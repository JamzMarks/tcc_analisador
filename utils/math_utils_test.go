package utils

import (
	"math"
	"testing"
)

// Verifica o cálculo da média simples
func TestMean(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		expected float64
		wantErr  bool
	}{
		{"média normal", []float64{10, 20, 30}, 20, false},
		{"único valor", []float64{42}, 42, false},
		{"vazio", []float64{}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Mean(tt.values)
			if (err != nil) != tt.wantErr {
				t.Errorf("Mean() erro esperado=%v, obtido=%v", tt.wantErr, err)
				return
			}
			if !tt.wantErr && math.Abs(got-tt.expected) > 1e-9 {
				t.Errorf("Mean() = %v, esperado %v", got, tt.expected)
			}
		})
	}
}

// Verifica o cálculo da média ponderada
func TestWeightedMean(t *testing.T) {
	tests := []struct {
		name     string
		values   []float64
		weights  []float64
		expected float64
		wantErr  bool
	}{
		{"ponderada normal", []float64{10, 20, 30}, []float64{1, 1, 2}, 22.5, false},
		{"pesos iguais", []float64{5, 15, 25}, []float64{1, 1, 1}, 15, false},
		{"pesos zero", []float64{1, 2, 3}, []float64{0, 0, 0}, 0, true},
		{"tamanhos diferentes", []float64{1, 2}, []float64{1}, 0, true},
		{"vazio", []float64{}, []float64{}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := WeightedMean(tt.values, tt.weights)
			if (err != nil) != tt.wantErr {
				t.Errorf("WeightedMean() erro esperado=%v, obtido=%v", tt.wantErr, err)
				return
			}
			if !tt.wantErr && math.Abs(got-tt.expected) > 1e-9 {
				t.Errorf("WeightedMean() = %v, esperado %v", got, tt.expected)
			}
		})
	}
}
