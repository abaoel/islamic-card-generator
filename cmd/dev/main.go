// Command dev runs a local HTTP server that mirrors the Vercel routing:
//
//	POST /api/suggest  -> JSON verses+du'a from AI
//	POST /api/card     -> PNG du'a card
//	GET  /             -> ./public/index.html
//
// Usage:
//
//	# copy .env.example -> .env, paste your GROQ_API_KEY
//	go run ./cmd/dev
//	# open http://localhost:3000
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/abaoel/islamic-card-generator/dua"
)

func main() {
	_ = godotenv.Load()

	mux := http.NewServeMux()

	mux.HandleFunc("/api/suggest", func(w http.ResponseWriter, r *http.Request) {
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
	})

	mux.HandleFunc("/api/card", func(w http.ResponseWriter, r *http.Request) {
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
	})

	mux.Handle("/", http.FileServer(http.Dir("public")))

	addr := ":" + envDefault("PORT", "3000")
	log.Printf("islamic-card-generator dev server → http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func envDefault(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
