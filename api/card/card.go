// Package handler exports the Vercel serverless entry point for POST
// /api/card which renders a PNG du'a card from the given JSON body.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/abaoel/islamic-card-generator/dua"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var p dua.RenderParams
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "bad JSON body: "+err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	if err := dua.Render(w, p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
