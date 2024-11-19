package main

import (
	"log"
	"math"
	"strings"
)

// Define a set of common stop words.
// These are words that are typically filtered out before processing text.
var stopWords = map[string]struct{}{
	"a": {}, "and": {}, "the": {}, "is": {}, "to": {},
	"of": {}, "in": {}, "it": {}, "that": {}, "you": {},
	"this": {}, "for": {}, "on": {}, "are": {}, "with": {},
	"as": {}, "be": {}, "by": {}, "at": {}, "from": {},
	"or": {}, "an": {}, "but": {}, "not": {}, "we": {},
	// Add more common stop words as necessary
}

// TFIDF struct holds the term frequency and inverse document frequency.
type TFIDF struct {
	TermFrequency  map[string]float64 // Frequencies of terms in the corpus
	InverseDocFreq map[string]float64 // Inverse document frequencies for terms
}

// NewTFIDF creates a new TFIDF instance based on the provided corpus of documents.
func NewTFIDF(corpus []string) *TFIDF {
	tf := make(map[string]float64)  // Initialize map to store term frequencies
	idf := make(map[string]float64) // Initialize map to store inverse document frequencies

	// Calculate Term Frequency (TF)
	for _, doc := range corpus {
		words := strings.Fields(doc) // Split document into words
		for _, word := range words {
			tf[word]++ // Count occurrences of each word
		}
	}

	// Calculate Inverse Document Frequency (IDF)
	for term := range tf {
		idf[term] = math.Log(float64(len(corpus)) / (1 + float64(countDocumentsContainingTerm(corpus, term))))
	}

	// Return a new instance of TFIDF with calculated TF and IDF
	return &TFIDF{TermFrequency: tf, InverseDocFreq: idf}
}

// countDocumentsContainingTerm counts how many documents contain a specific term.
func countDocumentsContainingTerm(corpus []string, term string) int {
	count := 0 // Initialize count to zero
	for _, doc := range corpus {
		if strings.Contains(doc, term) { // Check if the term exists in the document
			count++ // Increment count if term is found
		}
	}
	return count // Return the total count
}

// CalculateVector computes the TF-IDF vector for a given document.
func (tfidf *TFIDF) CalculateVector(doc string) map[string]float64 {
	words := strings.Fields(doc)          // Split the document into individual words
	processedWords := processWords(words) // Apply enhanced NLP processing

	vector := make(map[string]float64)         // Initialize map to hold the TF-IDF vector
	totalWords := float64(len(processedWords)) // Get total number of processed words

	// Calculate the TF-IDF vector for each processed word
	for _, word := range processedWords {
		if _, exists := tfidf.TermFrequency[word]; exists { // Check if the word exists in Term Frequency map
			// Calculate the TF-IDF score for the word and add it to the vector
			vector[word] = (tfidf.TermFrequency[word] / totalWords) * tfidf.InverseDocFreq[word]
		}
	}

	return vector // Return the computed TF-IDF vector
}

// removeStopWords filters out stop words from the given list of words.
func removeStopWords(words []string) []string {
	filtered := []string{} // Initialize slice to store filtered words
	for _, word := range words {
		_, found := stopWords[word] // Check if the word is a stop word
		if !found {
			filtered = append(filtered, word) // Append non-stop words to the filtered slice
		}
	}
	return filtered // Return the filtered list of words
}

// stem applies basic stemming to a given word using rule-based techniques.
func stem(word string) string {
	// Basic stemming rules (not exhaustive)
	if strings.HasSuffix(word, "ing") {
		return strings.TrimSuffix(word, "ing") // Remove "ing"
	} else if strings.HasSuffix(word, "ed") {
		return strings.TrimSuffix(word, "ed") // Remove "ed"
	} else if strings.HasSuffix(word, "s") {
		return strings.TrimSuffix(word, "s") // Remove "s"
	}
	return word // Return the original word if no rules apply
}

// removeStopWordsAndStem processes a list of words, removing stop words and applying stemming.
func removeStopWordsAndStem(words []string) []string {
	filtered := []string{} // Initialize slice to hold filtered and stemmed words
	for _, word := range words {
		_, found := stopWords[word] // Check if it's a stop word
		if !found {
			stemmedWord := stem(word)                // Stem the word
			filtered = append(filtered, stemmedWord) // Add to filtered list
		}
	}
	return filtered // Return the list of filtered and stemmed words
}

// saveTrainingDataToDB stores training data in the database.
func saveTrainingDataToDB(data TrainingData) {
	// Insert the user query and corresponding answer into the interactions table
	_, err := db.Exec("INSERT INTO interactions(query, response) VALUES(?, ?)", data.Query, data.Answer)
	if err != nil {
		log.Println("Error saving training data:", err) // Log any errors encountered during the database operation
	}
}

// retrainTFIDFModel recalculates the TF-IDF values for the entire dataset, refreshing the model based on new data.
func retrainTFIDFModel() {
	// Reload the latest dataset to recalculate TF-IDF values
	// Load the expanded corpus
	corpus, err := LoadCorpus("go_corpus.md")
	if err != nil {
		log.Fatal("Error loading corpus:", err)
	}
	// Assuming dataset is loaded/predefined
	// Create the TF-IDF model and calculate the query vector
	tfidf := NewTFIDF(corpus) // Implement loadCorpus to retrieve your documents
	for i := range dataset {
		dataset[i].Vector = tfidf.CalculateVector(dataset[i].Answer) // Recalculate vectors for each existing dataset entry
	}
}

// advancedStem applies a more sophisticated stemming process with programming-specific rules.
func advancedStem(word string) string {
	// Define common suffixes to be removed
	suffixes := []string{"es", "ed", "ing", "s", "ly", "ment", "ness", "ity", "ism", "er"}

	// Specific keywords related to Go that should not be altered
	programmingKeywords := map[string]struct{}{
		"func": {}, "package": {}, "import": {}, "interface": {}, "go": {},
		"goroutine": {}, "channel": {}, "select": {}, "struct": {},
		"map": {}, "slice": {}, "var": {}, "const": {}, "type": {},
		"defer": {}, "fallthrough": {},
	}

	// Check if the word is a Go keyword and return it unchanged
	if _, isKeyword := programmingKeywords[word]; isKeyword {
		return word
	}

	// Process the word to remove any common suffix
	for _, suffix := range suffixes {
		if strings.HasSuffix(word, suffix) {
			// Handle the case for "ies" specifically
			if suffix == "es" && strings.HasSuffix(word[:len(word)-2], "i") {
				return word[:len(word)-2] // Return the stemmed word
			}
			if suffix == "ed" || suffix == "ing" {
				return word[:len(word)-len(suffix)] // Return base form, e.g., "running" -> "run"
			}
			return word[:len(word)-len(suffix)] // Remove the suffix for other cases
		}
	}
	return word // Return the original word if no modifications were made
}

// removeStopWordsAndAdvancedStem filters out stop words and applies advanced stemming.
func removeStopWordsAndAdvancedStem(words []string) []string {
	filtered := []string{} // Initialize a slice to hold filtered and stemmed words
	for _, word := range words {
		_, found := stopWords[word] // Check for stop words
		if !found {
			// Apply advanced stemming to the word
			stemmedWord := advancedStem(word)
			filtered = append(filtered, stemmedWord) // Append to the filtered list
		}
	}
	return filtered // Return the list of filtered and stemmed words
}

// lemmatize targets programming-specific terms to convert them to their base form.
func lemmatize(word string) string {
	// Define lemmatization rules for common programming-related vocabulary
	lemmatizationRules := map[string]string{
		"execute":    "execute", // No change, but potentially useful when parsing
		"running":    "run",
		"returns":    "return",
		"defined":    "define",
		"compiles":   "compile",
		"calls":      "call",
		"creating":   "create",
		"invoke":     "invoke",
		"declares":   "declare",
		"references": "reference",
		"implements": "implement",
		"utilizes":   "utilize",
		"tests":      "test",
		"loops":      "loop",
		"deletes":    "delete",
	}

	// Check if the word has a mapping in the rules
	if baseForm, found := lemmatizationRules[word]; found {
		return baseForm // Return the lemmatized base form
	}

	// Fallback: Remove common verb endings for simple transformations
	if strings.HasSuffix(word, "ing") || strings.HasSuffix(word, "ed") {
		return word[:len(word)-len("ing")] // Return the base form
	}

	return word // Return the original word if no rules apply
}

// processWords applies the above NLP processing to a list of words.
func processWords(words []string) []string {
	// Remove stop words and perform advanced stemming on the words
	filtered := removeStopWordsAndAdvancedStem(words)

	// Lemmatize the processed words
	for i, word := range filtered {
		filtered[i] = lemmatize(word) // Apply lemmatization
	}

	return filtered // Return the final list of processed words
}
