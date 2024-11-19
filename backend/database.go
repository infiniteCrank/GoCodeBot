package main

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

// Database connection variable.
var db *sql.DB

func connectDatabase() {
	var err error
	db, err = sql.Open("mysql", "user:password@tcp(127.0.0.1:3306)/yourdatabase")
	if err != nil {
		log.Fatal(err)
	}
}

func saveInteraction(query, response string) {
	_, err := db.Exec("INSERT INTO interactions(query, response) VALUES(?, ?)", query, response)
	if err != nil {
		log.Println("Error saving interaction:", err)
	}
}

// logInteraction logs user interactions including their queries, responses, and ratings into the database.
func logInteraction(query, response string, rating int) {
	// Store user query, response, and rating in the interaction_logs table
	_, err := db.Exec("INSERT INTO interaction_logs(query, response, feedback_rating) VALUES(?, ?, ?)", query, response, rating)
	if err != nil {
		log.Println("Error logging interaction:", err) // Log any error encountered while logging interaction
	}
}
