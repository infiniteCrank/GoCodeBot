package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/ws", handleWebSocket)
	http.Handle("/", http.FileServer(http.Dir("./frontend"))) // Serve static files
	http.Handle("/ws", router)
	setupRoutes(router)

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

var dataset []DataPoint

func handleUserInput(query string) string {
	// Load the expanded corpus
	corpus, err := LoadCorpus("go_corpus.txt")
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

type TrainingData struct {
	Query  string `json:"query"`
	Answer string `json:"answer"`
}

func setupRoutes(r *mux.Router) {
	r.HandleFunc("/train", handleTraining).Methods("POST")
	r.HandleFunc("/ws", handleWebSocket)
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

type Feedback struct {
	Query    string `json:"query"`
	Response string `json:"response"`
	Rating   int    `json:"rating"`
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
			response := handleUserInput(query)
			err = conn.WriteJSON(map[string]string{"type": "response", "response": response})
			if err != nil {
				log.Println("Error on write:", err)
			}

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

func saveFeedbackToDB(feedback Feedback) {
	_, err := db.Exec("INSERT INTO feedback(query, response, rating) VALUES(?, ?, ?)", feedback.Query, feedback.Response, feedback.Rating)
	if err != nil {
		log.Println("Error saving feedback:", err)
	}
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
