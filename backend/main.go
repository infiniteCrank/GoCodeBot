package main

import (
	"bufio"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

var discoveredIntents map[string][]string // Holds potential new intents and their associated phrases

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

func main() {

	// Initialize discovered intents
	discoveredIntents = make(map[string][]string)

	connectDatabase()

	// Load any existing discovered intents from the database
	loadDiscoveredIntents()

	// Start the validation loop for new intents
	go func() {
		for range time.Tick(time.Minute) { // Check every minute
			validateNewIntents()
		}
	}()

	router := mux.NewRouter()
	router.HandleFunc("/ws", handleWebSocket)
	router.HandleFunc("/train", handleTraining).Methods("POST")
	http.Handle("/", http.FileServer(http.Dir("./frontend"))) // Serve static files

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
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
	// Load the expanded corpus
	corpus, err := LoadCorpus("go_corpus.md")
	if err != nil {
		log.Fatal("Error loading corpus:", err)
	}
	// Assuming dataset is loaded/predefined
	// Create the TF-IDF model and calculate the query vector
	tfidf := NewTFIDF(corpus) // Implement loadCorpus to retrieve your documents
	queryVec := tfidf.CalculateVector(query)

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
				response = handleUserInput(query) // Get response from KNN
			default:
				response = handleUserInput(query) // General response fallback
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

func validateNewIntents() {
	for intentKey, queries := range discoveredIntents {
		if len(queries) > exampleThreshold { // Set a threshold for how many examples define a new intent
			// Here you can automatically create a new intent with existing phrases
			log.Println("New Intent Discovered:", intentKey, "with queries:", queries)
			// Optionally, you may want to ask users about this new intent
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

	// Calculate TF-IDF for input query
	tf := NewTFIDF(corpus)
	queryVec := tf.CalculateVector(preprocessedQuery)

	bestIntent := ""
	highestSimilarity := -1.0

	// Classify query against intents
	for _, intent := range intents {
		for _, phrase := range intent.TrainingPhrases {
			phraseVec := tf.CalculateVector(phrase)             // Calculate vector for the training phrase
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

//set up a Go routine that runs at defined intervals to analyze the gathered data, adjust the corpus, and re-calculate the TF-IDF model
// go func() {
//     for range time.Tick(time.Hour) { // Adjust to your preferred interval
//         retrainModelBasedOnFeedback()
//     }
// }()

// func retrainModelBasedOnFeedback() {
//     // Logic to fetch interaction_logs from the database
//     // Implement your retraining logic here, possibly reloading the corpus

//     // Example:
//     // Analyze frequent queries and adjust the corpus
//     log.Println("Retraining model based on collected feedback and interactions.")
// }
