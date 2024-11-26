package main

import (
	"bufio"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

var discoveredIntents map[string][]string // Holds potential new intents and their associated phrases

var programmingTerms = make(map[string][]string) // Dynamically defined programming terms

// Intent struct for intent classification
type Intent struct {
	Name            string
	TrainingPhrases []string
}

const exampleThreshold = 3

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

var dataset []DataPoint

// Handle new training data
type TrainingData struct {
	Query  string `json:"query"`
	Answer string `json:"answer"`
}

type Feedback struct {
	Query    string `json:"query"`
	Response string `json:"response"`
	Rating   int    `json:"rating"`
}

type KeywordEntity struct {
	Name        string
	Description string
	Category    string
}

var (
	corpus              []string
	corpusKeywords      map[string]float64
	tfidf               *TFIDF
	programmingKeywords map[string]KeywordEntity
)

func initialize() {
	// Initialize discovered intents
	discoveredIntents = make(map[string][]string)

	connectDatabase()

	// Load programming keywords
	err := loadProgrammingKeywords("Go_keyword_entities.txt")
	if err != nil {
		log.Fatal("Error loading programming keywords:", err)
	}

	// Load any existing discovered intents from the database
	loadDiscoveredIntents()

	// Load programming concepts from the corpus
	err = loadCorpusConcepts("go_corpus.md")
	if err != nil {
		log.Fatal("Error loading corpus concepts:", err)
	}

	// Load the existing corpus of training phrases
	corpus, err = LoadCorpus("go_corpus.md")
	if err != nil {
		log.Fatal("Error loading corpus:", err)
	}

	// Create the TF-IDF model
	tfidf = NewTFIDF(corpus)

	// Extract keywords from the corpus
	corpusKeywords = tfidf.ExtractKeywords(corpus, 20) // Adjust top N as necessary

	// Dynamically initialize programming terms from the corpus
	initializeProgrammingTerms(corpus)
}

func loadProgrammingKeywords(filename string) error {
	programmingKeywords = make(map[string]KeywordEntity)
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	currentCategory := ""

	for scanner.Scan() {
		line := scanner.Text()

		// Ignore empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for keyword categories
		if strings.HasPrefix(line, "Control Flow Keywords") ||
			strings.HasPrefix(line, "Function and Variable Keywords") ||
			strings.HasPrefix(line, "Data Structure Keywords") {
			currentCategory = line
			continue
		}

		// Each keyword line format: keyword - description
		if strings.Contains(line, "-") {
			parts := strings.SplitN(line, "-", 2)
			if len(parts) == 2 {
				keyword := strings.TrimSpace(parts[0])
				description := strings.TrimSpace(parts[1])
				programmingKeywords[keyword] = KeywordEntity{
					Name:        keyword,
					Description: description,
					Category:    currentCategory,
				}
			}
		}
	}

	return scanner.Err()
}

func main() {

	// init the corpus and supporting data
	initialize()

	// Start the validation loop for new intents
	go func() {
		for range time.Tick(time.Minute) { // Check every minute
			validateNewIntents()
		}
	}()

	// Set up a Go routine that runs at defined intervals to retrain based on feedback
	go func() {
		for range time.Tick(time.Hour) { // Adjust to your preferred interval
			retrainModelBasedOnFeedback()
		}
	}()

	router := mux.NewRouter()
	router.HandleFunc("/ws", handleWebSocket)
	router.HandleFunc("/train", handleTraining).Methods("POST")
	http.Handle("/", http.FileServer(http.Dir("./frontend"))) // Serve static files

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
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
	re := regexp.MustCompile(`\b([A-Z][a-zA-Z0-9]*)\b`) // Match capitalized words (likely class names, etc.)
	matches := re.FindAllString(text, -1)

	// Add regex matches to terms
	terms = append(terms, matches...)

	// Add programming keywords from the map
	for keyword := range programmingKeywords {
		if strings.Contains(text, keyword) {
			terms = append(terms, keyword) // Add keyword if found in text
		}
	}

	return terms
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
		dataset[i].Vector = tfidf.CalculateVector(dataset[i].Answer) // Recalculate vectors for each existing dataset entry
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

func handleUserInput(query string) string {

	// Initialize on the first user input if not already done
	if corpus == nil {
		initialize()
	}

	queryVec := tfidf.CalculateVector(query)

	// Get response using KNN
	response := KNN(queryVec, dataset, 3) // Adjust k as needed
	// Check if the query contains any extracted keywords
	var relatedKeywords []string
	for term := range corpusKeywords {
		if strings.Contains(strings.ToLower(query), term) {
			relatedKeywords = append(relatedKeywords, term)
		}
	}

	// Enhance the response with related topics
	if len(relatedKeywords) > 0 {
		response += "\n\nRelated Keywords: " + strings.Join(relatedKeywords, ", ")
	}

	return response // Return the final response
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

// Function to handle new training data
func handleTraining(w http.ResponseWriter, r *http.Request) {
	var data TrainingData
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Add it to dataset and retrain
	dataset = append(dataset, DataPoint{Vector: nil, Answer: data.Answer})
	retrainTFIDFModel()        // Function implementation needed
	saveTrainingDataToDB(data) // Persist to DB if needed
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
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

			// Use KNN to get relevant responses
			knnResponse := handleUserInput(query) // Get response from KNN

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

			// Classify user intent
			intent := classifyIntent(query)

			if intent == "" { // Intent is not recognized
				// Add the new query to discovered intents
				aggregateDiscoveredIntents(query)
			}

			var response string

			// Respond based on classified intent
			switch intent {
			case "greeting":
				response = "Bot: Hello! How can I assist you today?"
			case "farewell":
				response = "Bot: Goodbye! Have a great day!"
			case "help":
				// Get response using KNN for help queries
				response = finalResponse // Get response from KNN
			default:
				response = finalResponse // General response fallback
			}

			// Send the response back to the client
			err = conn.WriteJSON(map[string]string{"type": "response", "response": response})
			if err != nil {
				log.Println("Error on write:", err)
			}

			// Log the interaction after generating a response.
			saveInteraction(query, response) // Log the interaction with query and response.

		case "feedback":
			feedback := Feedback{
				Query:    msg["query"].(string),
				Response: msg["response"].(string),
				Rating:   int(msg["rating"].(float64)),
			}
			saveFeedbackToDB(feedback)                                         // Persist to feedback table
			logInteraction(feedback.Query, feedback.Response, feedback.Rating) // Log interaction
		}
	}
}

// Function to aggregate new intents discovered from user queries
func aggregateDiscoveredIntents(query string) {
	processedQuery := preprocessInput(query) // Preprocess input

	// Find a suitable cluster key for the query.
	clusterKey := findClusterKey(processedQuery)

	// Add the query to the corresponding discovered intent cluster.
	if _, exists := discoveredIntents[clusterKey]; !exists {
		discoveredIntents[clusterKey] = []string{} // Initialize if doesn't exists
	}
	discoveredIntents[clusterKey] = append(discoveredIntents[clusterKey], processedQuery) // Add the new query

	// Persisting the new intent to the database
	persistDiscoveredIntent(clusterKey, processedQuery)
}

// Function to find a cluster key based on a processed query
func findClusterKey(query string) string {
	// For simplicity, use the first few words as a basic key
	words := strings.Fields(query)
	if len(words) > 0 {
		return strings.Join(words[:min(3, len(words))], "_") // Join the first three words as a key
	}
	return query // If no words, return the query itself
}

// Function to persist discovered intents into the database
func persistDiscoveredIntent(intentName string, phrase string) {
	_, err := db.Exec("INSERT INTO discovered_intents (intent_name, training_phrases) VALUES (?, ?) ON DUPLICATE KEY UPDATE training_phrases = CONCAT(training_phrases, ';', ?)", intentName, phrase, phrase)
	if err != nil {
		log.Println("Error saving discovered intent to database:", err)
	}
}

// Helper function to return the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Validate new intents periodically
func validateNewIntents() {
	for intentKey, queries := range discoveredIntents {
		if len(queries) >= exampleThreshold { // Set a threshold for how many examples define a new intent
			log.Println("New Intent Discovered:", intentKey, "with queries:", queries)

			// Automatically create the new intent with existing phrases
			newIntent := Intent{Name: intentKey, TrainingPhrases: queries}
			intents = append(intents, newIntent) // Add the new intent to the intent
			// Persist the new intent to the database
			persistDiscoveredIntent(intentKey, strings.Join(queries, ";"))

			// Clear the discovered intent after saving
			delete(discoveredIntents, intentKey)
		}
	}
}

func saveFeedbackToDB(feedback Feedback) {
	_, err := db.Exec("INSERT INTO feedback(query, response, rating) VALUES(?, ?, ?)", feedback.Query, feedback.Response, feedback.Rating)
	if err != nil {
		log.Println("Error saving feedback:", err)
	}
}

// Function for intent classification
func classifyIntent(query string) string {
	preprocessedQuery := preprocessInput(query)

	// Create a corpus from the intents' training phrases
	corpus := []string{}
	for _, intent := range intents {
		corpus = append(corpus, intent.TrainingPhrases...)
	}

	queryVec := tfidf.CalculateVector(preprocessedQuery)

	bestIntent := ""
	highestSimilarity := -1.0

	// Classify query against intents
	for _, intent := range intents {
		for _, phrase := range intent.TrainingPhrases {
			phraseVec := tfidf.CalculateVector(phrase)          // Calculate vector for the training phrase
			similarity := cosineSimilarity(queryVec, phraseVec) // Compute cosine similarity

			// Check for the best intent based on similarity
			if similarity > highestSimilarity {
				highestSimilarity = similarity
				bestIntent = intent.Name
			}
		}
	}

	return bestIntent
}

// Preprocess the user input (lowercase, etc.)
func preprocessInput(input string) string {
	return strings.ToLower(input)
}

// Cosine similarity function (reuse or implement below if not defined elsewhere)
func cosineSimilarity(vec1, vec2 map[string]float64) float64 {
	dotProduct := 0.0
	normA := 0.0
	normB := 0.0

	for key, val1 := range vec1 {
		if val2, found := vec2[key]; found {
			dotProduct += val1 * val2
		}
		normA += val1 * val1
	}

	for _, val2 := range vec2 {
		normB += val2 * val2
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)) // Return cosine similarity
}
