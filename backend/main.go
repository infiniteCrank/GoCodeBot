package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Global variables
var upgrader = websocket.Upgrader{}
var discoveredIntents map[string][]string        // Holds potential new intents and their associated phrases
var programmingTerms = make(map[string][]string) // Dynamically defined programming terms

// Intent struct for intent classification
type Intent struct {
	Name            string
	TrainingPhrases []string
}

const exampleThreshold = 3

// Predefined intents for the chatbot
var intents = []Intent{
	{
		Name:            "greeting",
		TrainingPhrases: []string{"hello", "hi", "how are you", "good morning", "hey"},
	},
	{
		Name:            "farewell",
		TrainingPhrases: []string{"bye", "goodbye", "see you later", "take care"},
	},
	{
		Name:            "help",
		TrainingPhrases: []string{"help me", "I need assistance", "can you help me"},
	},
	// Add more intents as needed...
}

var dataset []DataPoint // Collection of data points for KNN

// Structure for new training data
type TrainingData struct {
	Query  string `json:"query"`
	Answer string `json:"answer"`
}

// Structure for user feedback
type Feedback struct {
	Query    string `json:"query"`
	Response string `json:"response"`
	Rating   int    `json:"rating"`
}

// Function to initialize programming terms dynamically from the corpus
func initializeProgrammingTerms(corpus []string) {
	for _, line := range corpus {
		// Extract potential programming terms using regex heuristic
		terms := extractProgrammingTerms(line)
		for _, term := range terms {
			// Add the term to the dictionary with a placeholder description
			programmingTerms[term] = []string{"No description available yet."} // Placeholder
		}
	}
}

// Extract potential programming terms from a line of text
func extractProgrammingTerms(text string) []string {
	var terms []string

	// Use regex to find capitalized words or common programming patterns
	re := regexp.MustCompile(`\b([A-Z][a-zA-Z]*)\b`) // Match capitalized words (likely class names, etc.)
	matches := re.FindAllString(text, -1)

	for _, match := range matches {
		terms = append(terms, match)
	}

	return terms
}

// Main entry point for the application
func main() {
	// Initialize discovered intents
	discoveredIntents = make(map[string][]string)

	connectDatabase() // Connect to the database

	// Load existing discovered intents from the database
	loadDiscoveredIntents()

	// Load the existing corpus of training phrases
	corpus, err := LoadCorpus("go_corpus.md")
	if err != nil {
		log.Fatal("Error loading corpus:", err)
	}

	// Dynamically initialize programming terms from the corpus
	initializeProgrammingTerms(corpus)

	// Start a goroutine to validate new intents periodically
	go func() {
		for range time.Tick(time.Minute) { // Check every minute
			validateNewIntents()
		}
	}()

	// Start a goroutine to retrain the model based on user feedback periodically
	go func() {
		for range time.Tick(time.Hour) { // Adjust to your preferred interval
			retrainModelBasedOnFeedback()
		}
	}()

	router := mux.NewRouter()
	router.HandleFunc("/ws", handleWebSocket)                   // WebSocket connection handler
	router.HandleFunc("/train", handleTraining).Methods("POST") // Training endpoint
	http.Handle("/", http.FileServer(http.Dir("./frontend")))   // Serve static files

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Validate new intents periodically
func validateNewIntents() {
	for intentKey, queries := range discoveredIntents {
		if len(queries) >= exampleThreshold { // Set a threshold for how many examples define a new intent
			log.Println("New Intent Discovered:", intentKey, "with queries:", queries)

			// Automatically create the new intent with existing phrases
			newIntent := Intent{Name: intentKey, TrainingPhrases: queries}
			intents = append(intents, newIntent) // Add the new intent to the intent list
			// Persist the new intent to the database
			persistDiscoveredIntent(intentKey, strings.Join(queries, ";"))

			// Clear the discovered intent after saving
			delete(discoveredIntents, intentKey)
		}
	}
}

// Persist discovered intents into the database
func persistDiscoveredIntent(intentName string, phrase string) {
	_, err := db.Exec("INSERT INTO discovered_intents (intent_name, training_phrases) VALUES (?, ?) ON DUPLICATE KEY UPDATE training_phrases = CONCAT(training_phrases, ';', ?)", intentName, phrase, phrase)
	if err != nil {
		log.Println("Error saving discovered intent to database:", err)
	}
}

// Handle new training data and retraining of the TF-IDF model
func handleTraining(w http.ResponseWriter, r *http.Request) {
	var data TrainingData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Append the new training data to the dataset
	dataset = append(dataset, DataPoint{Vector: nil, Answer: data.Answer})
	retrainTFIDFModel()        // Refresh the model
	saveTrainingDataToDB(data) // Persist the training data to the database
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// Load programming concepts from the corpus
func loadCorpusConcepts(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var currentSection string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "##") {
			if currentSection != "" {
				programmingTerms[currentSection] = []string{"Definition or description not explicitly detailed."}
			}
			currentSection = strings.TrimSpace(line[2:])
		}
	}
	if currentSection != "" {
		programmingTerms[currentSection] = []string{"Definition or description not explicitly detailed."}
	}

	return scanner.Err()
}

// Extract entities based on dictionary lookup
func extractEntitiesAdvanced(query string) []string {
	entities := make([]string, 0)
	words := strings.Fields(strings.ToLower(query))

	// Check each word in the query against the dynamically defined programming terms
	for _, word := range words {
		if relatedEntities, exists := programmingTerms[word]; exists {
			entities = append(entities, word)               // Add the key term
			entities = append(entities, relatedEntities...) // Add related terms
		}
	}
	return entities
}

// Use regex to find programming constructs
func extractEntitiesWithRegex(query string) []string {
	var entities []string

	// Example regex patterns for capturing common programming constructs
	patterns := []string{
		`(?i)\b(func|package|import|type|var)\b`, // Match Go keywords
		`(?i)\b([A-Z][a-zA-Z]*)\b`,               // Match capitalized terms (e.g., struct names)
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(query, -1)
		entities = append(entities, matches...)
	}

	return entities
}

// Retrain the model based on collected feedback and interactions
func retrainModelBasedOnFeedback() {
	log.Println("Retraining model based on collected feedback and interactions.")

	// Fetch interaction logs from the database
	rows, err := db.Query("SELECT query, response FROM interactions")
	if err != nil {
		log.Println("Error fetching interaction logs from database:", err)
		return
	}
	defer rows.Close()

	var feedbackCorpus []string
	for rows.Next() {
		var query, response string
		if err := rows.Scan(&query, &response); err != nil {
			log.Println("Error scanning interaction log:", err)
			continue
		}
		// Prepare the feedback corpus
		feedbackCorpus = append(feedbackCorpus, query, response)
	}

	// Load the existing corpus of training phrases
	corpus, err := LoadCorpus("go_corpus.md")
	if err != nil {
		log.Fatal("Error loading corpus:", err)
	}

	// Combine the feedback corpus with the existing corpus
	corpus = append(corpus, feedbackCorpus...)

	// Create a new TF-IDF model based on the updated corpus
	tfidf := NewTFIDF(corpus)

	// Recalculate TF-IDF vectors for the dataset
	for i := range dataset {
		dataset[i].Vector = tfidf.CalculateVector(dataset[i].Answer) // Recalculate TF-IDF vectors
	}

	log.Println("Model retraining completed successfully.")
}

// Load existing discovered intents from the database
func loadDiscoveredIntents() {
	rows, err := db.Query("SELECT intent_name, training_phrases FROM discovered_intents")
	if err != nil {
		log.Println("Error loading discovered intents from database:", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var intentName string
		var trainingPhrases string

		if err := rows.Scan(&intentName, &trainingPhrases); err != nil {
			log.Println("Error scanning discovered intent:", err)
			continue
		}

		// Split training phrases and assign to the discovered intents map
		phrases := strings.Split(trainingPhrases, ";") // Assuming semicolon separation
		discoveredIntents[intentName] = phrases
	}
}

// Handle user input and respond based on KNN and dynamic entities
func handleUserInput(query string) string {
	// Load the expanded corpus
	corpus, err := LoadCorpus("go_corpus.md")
	if err != nil {
		log.Fatal("Error loading corpus:", err)
	}
	// Create the TF-IDF model and calculate the query vector
	tfidf := NewTFIDF(corpus)                // Create a new TF-IDF model
	queryVec := tfidf.CalculateVector(query) // Calculate TF-IDF for the user query

	// Get response using KNN
	return KNN(queryVec, dataset, 3) // Adjust k as needed
}

// LoadCorpus loads the corpus from a text file and returns a slice of strings.
func LoadCorpus(filename string) ([]string, error) {
	var corpus []string
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		corpus = append(corpus, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return corpus, nil
}

// Handle the websocket interaction for user queries
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error during connection upgrade:", err)
		return
	}
	defer conn.Close()

	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Error on read:", err)
			break
		}

		switch msg["type"] {
		case "query":
			query := msg["query"].(string)

			// Extract noun phrases and advanced entities from the query
			nounPhrases := extractNounPhrases(query)
			entities := extractEntitiesAdvanced(query)

			// Convert the query into a TF-IDF vector
			corpus, err := LoadCorpus("go_corpus.md")
			if err != nil {
				log.Fatal("Error loading corpus:", err)
			}
			tfidf := NewTFIDF(corpus)                // Create a new TF-IDF model
			queryVec := tfidf.CalculateVector(query) // Calculate TF-IDF for the user query

			// Use KNN to get relevant responses
			knnResponse := KNN(queryVec, dataset, 3) // Get KNN responses

			// Prepare the final response
			var finalResponse string

			// Combine KNN response with recognized entities
			if knnResponse != "" {
				finalResponse = knnResponse
				if len(entities) > 0 {
					finalResponse += "\n\nRelated Topics: " + strings.Join(entities, ", ")
				}
			} else {
				// If no KNN responses, generate responses based on noun phrases
				finalResponse = generateResponseFromNounPhrases(nounPhrases)
			}

			// Send the final response back to the client
			err = conn.WriteJSON(map[string]string{"type": "response", "response": finalResponse})
			if err != nil {
				log.Println("Error on write:", err)
			}

			// Log the interaction
			saveInteraction(query, finalResponse) // Log the interaction with query and response.
		}
	}
}

// Function to extract noun phrases
func extractNounPhrases(query string) []string {
	words := strings.Fields(strings.ToLower(query))
	nounPhrases := make([]string, 0)

	for _, word := range words {
		if isProgrammingTerm(word) {
			nounPhrases = append(nounPhrases, word)
		}
	}

	return nounPhrases
}

// Function to check if a word is a recognized programming term
func isProgrammingTerm(word string) bool {
	// Check against dynamically loaded programmingTerms map
	if _, exists := programmingTerms[word]; exists {
		return true
	}
	return false
}

// Function to generate responses based on extracted noun phrases
func generateResponseFromNounPhrases(nounPhrases []string) string {
	var responses []string
	for _, nounPhrase := range nounPhrases {
		if len(programmingTerms[nounPhrase]) > 0 {
			responses = append(responses, programmingTerms[nounPhrase][0]) // Access the first description for that term
		} else {
			responses = append(responses, "I'm sorry, but I do not have information on: "+nounPhrase)
		}
	}
	if len(responses) > 0 {
		return strings.Join(responses, "\n") // Join the responses for clarity
	}
	return "Sorry, I couldn't find relevant information."
}
