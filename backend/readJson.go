package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func ReadJsonConfigToEnvVars(fileName string) {
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Decode the JSON data
	decoder := json.NewDecoder(file)
	var configData map[string]string
	err = decoder.Decode(&configData)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	// Set environment variables
	for key, value := range configData {
		os.Setenv(key, value)
	}

}
