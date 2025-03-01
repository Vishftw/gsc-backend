package main

import (
	"encoding/json"
	"fmt"
	"net/http"
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

func main() {
	http.HandleFunc("/heartbeat", heartbeatHandler)
	fmt.Println("Server listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
