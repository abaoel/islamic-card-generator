// Package handler exports the Vercel serverless entry point for POST
// /api/suggest which asks the AI for verses + a du'a given a situation.
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/abaoel/islamic-card-generator/dua"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Situation string `json:"situation"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad JSON body: "+err.Error(), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	s, err := dua.Suggest(ctx, body.Situation)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(s)
}
