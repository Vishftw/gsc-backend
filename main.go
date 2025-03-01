package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretspb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

type heartbeatResponse struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func heartbeatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := heartbeatResponse{
		Name:   "gsc-backend",
		Status: "running",
	}
	json.NewEncoder(w).Encode(response)
}

var db *pgx.Conn

func initDB() {
	var databaseURL string
	var err error
	if os.Getenv("ENV") == "LOCAL" {
		err = godotenv.Load()
		if err != nil {
			log.Fatalf("Error loading .env file: %v", err)
		}
		databaseURL = os.Getenv("DATABASE_URL")
		fmt.Println("Using local database URL:", databaseURL)
	} else {
		// Running in Cloud Run → Fetch from Secret Manager
		databaseURL, err = getSecret("DATABASE_URL")
		if err != nil {
			log.Fatalf("Failed to access secret: %v", err)
		}
		fmt.Println("Using secret-managed database URL")
	}

	db, err = pgx.Connect(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	fmt.Println("Connected to database!")
}

func getCarsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(context.Background(), "SELECT id, brand, model, price, description FROM cars")
	if err != nil {
		http.Error(w, "Failed to fetch cars", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var cars []map[string]interface{}
	for rows.Next() {
		var id int
		var brand, model, description string
		var price float64

		err := rows.Scan(&id, &brand, &model, &price, &description)
		if err != nil {
			http.Error(w, "Error scanning row", http.StatusInternalServerError)
			return
		}

		cars = append(cars, map[string]interface{}{
			"id":          id,
			"brand":       brand,
			"model":       model,
			"price":       price,
			"description": description,
		})
	}

	// Convert to JSON and send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cars)
}

func main() {
	initDB()
	http.HandleFunc("/heartbeat", heartbeatHandler)
	http.HandleFunc("/cars", getCarsHandler)
	fmt.Println("Server listening on port 8080")
	http.ListenAndServe(":8080", nil)
}

func getSecret(secretName string) (string, error) {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	accessRequest := &secretspb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretName),
	}
	result, err := client.AccessSecretVersion(ctx, accessRequest)
	if err != nil {
		return "", err
	}
	return string(result.Payload.Data), nil
}
