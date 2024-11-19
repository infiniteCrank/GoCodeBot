package main

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

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

func logInteraction(query, response string, rating int) {
	_, err := db.Exec("INSERT INTO interaction_logs(query, response, feedback_rating) VALUES(?, ?, ?)", query, response, rating)
	if err != nil {
		log.Println("Error logging interaction:", err)
	}
}
