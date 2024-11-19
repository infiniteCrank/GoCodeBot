package main

import (
	"math"
	"sort"
)

// DataPoint represents a single entry in the dataset for KNN.
// It contains a vector (representing its TF-IDF values), the response associated with that entry, and the associated intent.
type DataPoint struct {
	Vector map[string]float64 // TF-IDF vector for the data point
	Answer string             // The response associated with this data point
	Intent string             // The identified intent of the data point (optional)
}

// EuclideanDistance calculates the Euclidean distance between two vectors.
func EuclideanDistance(vec1, vec2 map[string]float64) float64 {
	var sum float64
	// Iterate over all keys in vec1 to compute the distance
	for key := range vec1 {
		// If key exists in vec2, compute the squared difference, otherwise treat vec2[key] as 0
		diff := vec1[key] - vec2[key]
		sum += diff * diff
	}
	// Include terms that are in vec2 but not in vec1
	for key := range vec2 {
		if _, exists := vec1[key]; !exists {
			sum += vec2[key] * vec2[key]
		}
	}
	return math.Sqrt(sum) // Return the square root of the sum of squared differences
}

// Distance represents the distance between a data point and a query along with its index in the dataset.
type Distance struct {
	Index int     // Index of the original DataPoint in the dataset
	Value float64 // Calculated distance to the query point
}

// ByDistance is a type that implements sorting of Distance slices based on their Value.
type ByDistance []Distance

// Len returns the number of elements in the collection.
func (a ByDistance) Len() int {
	return len(a)
}

// Swap exchanges the elements with indexes i and j.
func (a ByDistance) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// Less reports whether the element with index i should sort before the element with index j.
func (a ByDistance) Less(i, j int) bool {
	return a[i].Value < a[j].Value // Sort by increasing distance
}

// KNN function finds the k nearest neighbors to a given query vector.
// It returns the most common answer among the nearest neighbors.
func KNN(queryVec map[string]float64, dataset []DataPoint, k int) string {
	distances := make([]Distance, len(dataset)) // Initialize distances slice

	// Calculate the Euclidean distance for each data point in the dataset
	for i, point := range dataset {
		dist := EuclideanDistance(queryVec, point.Vector) // Calculate distance
		distances[i] = Distance{Index: i, Value: dist}    // Store index and distance
	}

	// Sort distances to find the nearest neighbors
	sort.Sort(ByDistance(distances))

	// Count the frequency of answers among the k nearest neighbors
	answerCount := make(map[string]int)
	for i := 0; i < k && i < len(distances); i++ {
		answerCount[dataset[distances[i].Index].Answer]++
	}

	// Determine the answer with the highest count (most common answer)
	var bestAnswer string
	maxCount := 0
	for answer, count := range answerCount {
		if count > maxCount {
			maxCount = count
			bestAnswer = answer
		}
	}

	return bestAnswer // Return the most common answer among the nearest neighbors
}
