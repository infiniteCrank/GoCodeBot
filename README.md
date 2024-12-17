# GoCodeBot

this is an AI chat bot that uses NLP and machine learning to teach people how to write code in GoLang

### DB Setup
```
CREATE USER 'gobot'@'localhost' IDENTIFIED BY 'somepassword!';
CREATE DATABASE gobotdb;
USE gobotdb;
CREATE TABLE interactions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    query VARCHAR(255) NOT NULL,
    response TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE feedback (
    id INT AUTO_INCREMENT PRIMARY KEY,
    query VARCHAR(255) NOT NULL,
    response TEXT NOT NULL,
    rating INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE interaction_logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    query VARCHAR(255) NOT NULL,
    response TEXT NOT NULL,
    feedback_rating INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE discovered_intents (
    id INT AUTO_INCREMENT PRIMARY KEY,
    intent_name VARCHAR(255) NOT NULL UNIQUE,
    training_phrases TEXT NOT NULL
);
```


### Run the Go Server:
Navigate to the /backend folder, and start the server with:

```
go run main.go database.go knn.go tfidf.go
```

# GoLang Bot

## Overview

This repository contains an AI chatbot designed to teach Go programming using NLP techniques, including TF-IDF and KNN algorithms.

## Key Features

- **Continuous Learning Loop**: Dynamically updates based on user feedback and query interactions.
- **User Interaction Logging**: Keeps structured logs of user interactions to inform model improvements.
- **Custom NLP Processing**: Involves stemming and lemmatization tailored for programming context.

## Code Structure

- `main.go`: Main application logic and entry point for the server.
- `database.go`: Database connection and CRUD operations for data logging.
- `knn.go`: Implements the K-Nearest Neighbors algorithm for query processing based on user input.
- `tfidf.go`: Contains the TF-IDF algorithm for vectorization of user queries and responses.
- `feedback.go`: Manages storing and processing user feedback.
- `user_interaction.go`: Responsible for logging user interactions and tracking feedback for continuous improvement.

## Key Components

### Continuous Learning Loop

- **Purpose**: Allows the bot to dynamically learn from user interactions over time, improving its capacity to respond to frequently asked questions and adapt to user feedback.
- **Implementation**: Upon receiving feedback (e.g. ratings), the bot logs interactions and re-evaluates its model corpus periodically (e.g. every hour) to adapt its response logic based on popular queries or patterns.

### User Interaction Logging

- **Purpose**: Maintain a running log of all user interactions, including queries, responses, and feedback ratings.
- **Implementation**:
  - Interaction data is stored in the `interaction_logs` table in the MariaDB database.
  - Queries are monitored for frequency tracking, which aids in identifying common user inquiries for potential expansion of the knowledge base.

### NLP Processing

- **Purpose**: To accurately interpret user queries and provide relevant programming information.
- **Stemming and Lemmatization**: Custom algorithms are developed to handle programming terms. Specific rules to retain programming keywords and adjust common English verbs used in coding context are outlined and implemented.
- **User Input Processing**: Each query undergoes processing to remove stop words, apply stemming, and perform lemmatization before matching against the stored corpus.

## Setup Instructions

1. **Prerequisites**:

   - GoLang installed (version 1.15+ recommended).
   - MariaDB (or MySQL) installed and configured.
   - Create a database to hold the interactions and a user for your Go server.

2. **Database Setup**:

   - Create tables for `interactions`, `interaction_logs`, and `feedback` using the provided SQL commands in earlier sections.

3. **Configuration**:

   - Update database connection parameters in `database.go` with your credentials.

4. **Running the Application**:

   - Navigate to the project directory in your terminal.
   - Run `go run main.go database.go knn.go tfidf.go feedback.go user_interaction.go`.
   - Access the chatbot via your web browser at `http://localhost:8080`.

5. **Interacting with the Bot**:
   - Type programming-related questions to get responses from the bot.
   - Provide feedback on responses using the rating system to further enhance the bot's learning capability.

## Future Enhancements

- **Expanded Corpus**: Incorporate more detailed programming examples and knowledge into the bot's responses for greater coverage of topics.
- **User Personalization**: Develop functionality for user profiles to offer personalized learning experiences based on previous interactions.
- **Advanced NLP Techniques**: Consider implementing additional NLP features, such as deeper semantic understanding, context awareness, or even machine learning-based approaches to improve query handling.

## Contributions

Contributions are welcome! If you have suggestions for improvements or new features, please open an issue or submit a pull request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
