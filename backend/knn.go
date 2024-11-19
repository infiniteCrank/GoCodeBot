package main

import (
	"sort"
)

type DataPoint struct {
	Vector map[string]float64
	Answer string
}

type Distance struct {
	Index int
	Value float64
}

// Compares distances for sorting
type ByDistance []Distance

func (a ByDistance) Len() int           { return len(a) }
func (a ByDistance) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByDistance) Less(i, j int) bool { return a[i].Value < a[j].Value }

func KNN(queryVec map[string]float64, dataset []DataPoint, k int) string {
	distances := make([]Distance, len(dataset))

	for i, point := range dataset {
		dist := EuclideanDistance(queryVec, point.Vector)
		distances[i] = Distance{Index: i, Value: dist}
	}

	// Sort distances
	sort.Sort(ByDistance(distances))

	// Count the answers of the k nearest neighbors
	answerCount := make(map[string]int)
	for i := 0; i < k; i++ {
		answerCount[dataset[distances[i].Index].Answer]++
	}

	// Find the answer with the highest count
	var bestAnswer string
	maxCount := 0
	for answer, count := range answerCount {
		if count > maxCount {
			maxCount = count
			bestAnswer = answer
		}
	}

	return bestAnswer
}
