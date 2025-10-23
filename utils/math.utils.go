package utils

import "errors"

func Mean(values []float64) (float64, error) {
	if len(values) == 0 {
		return 0, errors.New("não é possível calcular a média de um slice vazio")
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values)), nil
}

func WeightedMean(values, weights []float64) (float64, error) {
	if len(values) == 0 || len(weights) == 0 {
		return 0, errors.New("valores e pesos não podem estar vazios")
	}
	if len(values) != len(weights) {
		return 0, errors.New("valores e pesos devem ter o mesmo tamanho")
	}

	var sumVal, sumWeights float64
	for i := range values {
		sumVal += values[i] * weights[i]
		sumWeights += weights[i]
	}

	if sumWeights == 0 {
		return 0, errors.New("a soma dos pesos não pode ser zero")
	}

	return sumVal / sumWeights, nil
}

func Sum(values []float64) float64 {
	var total float64
	for _, v := range values {
		total += v
	}
	return total
}
