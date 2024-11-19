package main

import (
	"log"
	"math"
	"strings"
)

type TFIDF struct {
	TermFrequency  map[string]float64
	InverseDocFreq map[string]float64
}

func NewTFIDF(corpus []string) *TFIDF {
	tf := make(map[string]float64)
	idf := make(map[string]float64)

	// Calculate Term Frequency
	for _, doc := range corpus {
		words := strings.Fields(doc)
		for _, word := range words {
			tf[word]++
		}
	}

	// Calculate Inverse Document Frequency
	for term := range tf {
		idf[term] = math.Log(float64(len(corpus)) / (1 + float64(countDocumentsContainingTerm(corpus, term))))
	}

	return &TFIDF{TermFrequency: tf, InverseDocFreq: idf}
}

func countDocumentsContainingTerm(corpus []string, term string) int {
	count := 0
	for _, doc := range corpus {
		if strings.Contains(doc, term) {
			count++
		}
	}
	return count
}

func (tfidf *TFIDF) CalculateVector(doc string) map[string]float64 {
	words := strings.Fields(doc)          // Split document into words
	processedWords := processWords(words) // Apply enhanced NLP processing

	vector := make(map[string]float64)
	totalWords := float64(len(processedWords)) // Get the total words after processing

	// Create the TF-IDF vector for the processed words
	for _, word := range processedWords {
		if _, exists := tfidf.TermFrequency[word]; exists {
			vector[word] = (tfidf.TermFrequency[word] / totalWords) * tfidf.InverseDocFreq[word]
		}
	}

	return vector
}

func removeStopWords(words []string) []string {
	filtered := []string{}
	for _, word := range words {
		_, found := stopWords[word]
		if !found {
			filtered = append(filtered, word)
		}
	}
	return filtered
}

func stem(word string) string {
	// Very basic rule-based stemmer (not complete)
	if strings.HasSuffix(word, "ing") {
		return strings.TrimSuffix(word, "ing")
	} else if strings.HasSuffix(word, "ed") {
		return strings.TrimSuffix(word, "ed")
	} else if strings.HasSuffix(word, "s") {
		return strings.TrimSuffix(word, "s")
	}
	return word
}

func removeStopWordsAndStem(words []string) []string {
	filtered := []string{}
	for _, word := range words {
		_, found := stopWords[word]
		if !found {
			stemmedWord := stem(word)
			filtered = append(filtered, stemmedWord)
		}
	}
	return filtered
}

func saveTrainingDataToDB(data TrainingData) {
	_, err := db.Exec("INSERT INTO interactions(query, response) VALUES(?, ?)", data.Query, data.Answer)
	if err != nil {
		log.Println("Error saving training data:", err)
	}
}

func retrainTFIDFModel() {
	// This will recalculate TF-IDF values for the entire dataset
	tfidf := NewTFIDF(loadCorpus()) // Reload the latest dataset
	for i := range dataset {
		dataset[i].Vector = tfidf.CalculateVector(dataset[i].Answer) // Recalculate vectors for existing dataset
	}
}

// Function to enhance stemming process with common rules
// Enhanced stemmer specific to GoLang terminology
func advancedStem(word string) string {
	// Common suffixes for stemming including programming-related words
	suffixes := []string{"es", "ed", "ing", "s", "ly", "ment", "ness", "ity", "ism", "er"}

	// Specific programming keywords that should not be altered
	programmingKeywords := map[string]struct{}{
		"func": {}, "package": {}, "import": {}, "interface": {}, "go": {}, "goroutine": {},
		"channel": {}, "select": {}, "struct": {}, "map": {}, "slice": {},
		"var": {}, "const": {}, "type": {}, "defer": {}, "fallthrough": {},
	}

	// Check if the word is a keyword and return unchanged
	if _, isKeyword := programmingKeywords[word]; isKeyword {
		return word
	}

	// Process general suffixes
	for _, suffix := range suffixes {
		if strings.HasSuffix(word, suffix) {
			// Special handling for common scenarios
			if suffix == "es" && strings.HasSuffix(word[:len(word)-2], "i") {
				return word[:len(word)-2] // handle "ies"
			}
			if suffix == "ed" || suffix == "ing" {
				return word[:len(word)-len(suffix)] // Return to root form
			}
			return word[:len(word)-len(suffix)] // Remove other suffixes
		}
	}
	return word // Return word as is if no suffix matches
}

// Function to process a list of words, applying stemming
func removeStopWordsAndAdvancedStem(words []string) []string {
	filtered := []string{}
	for _, word := range words {
		_, found := stopWords[word]
		if !found {
			stemmedWord := advancedStem(word)
			filtered = append(filtered, stemmedWord)
		}
	}
	return filtered
}

// Simple lemmatization targeting programming-related terms
func lemmatize(word string) string {
	// Rules specific for programming-related vocabulary
	lemmatizationRules := map[string]string{
		"execute":  "execute", // No change, but you could consider variations
		"running":  "run",
		"returns":  "return",
		"defined":  "define",
		"compiles": "compile",
		"calls":    "call",
		"creating": "create",
		// Add more rules specifically for programming...
		"invoke":     "invoke",
		"declares":   "declare",
		"references": "reference",
		"implements": "implement",
		"utilizes":   "utilize",
		"tests":      "test",
		"loops":      "loop",
		"deletes":    "delete",
	}

	// Check the mapping first
	if baseForm, found := lemmatizationRules[word]; found {
		return baseForm
	}

	// Basic fallback: Remove verb endings for common forms
	if strings.HasSuffix(word, "ing") || strings.HasSuffix(word, "ed") {
		return word[:len(word)-len("ing")] // take the base form
	}

	return word // Return the word untouched if no rule applies
}

// Process words specifically for programming
func processWords(words []string) []string {
	filtered := removeStopWordsAndAdvancedStem(words) // Existing filtering/stemming
	for i, word := range filtered {
		filtered[i] = lemmatize(word) // Apply lemmatization
	}
	return filtered
}
