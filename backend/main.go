package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// GET /health
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	resp := HealthResponse{
		Status:  "ok",
		Message: "Lifebot backend is running",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// POST /upload-report
// form-data:
//
//	file: <PDF file>
func uploadReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit + parse multipart form
	if err := r.ParseMultipartForm(20 << 20); err != nil { // 20 MB
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read file field 'file'", http.StatusBadRequest)
		return
	}
	defer file.Close()

	log.Println("Uploaded file:", header.Filename)

	// Read PDF into memory
	pdfBytes, err := io.ReadAll(file)
	if err != nil {
		log.Println("Error reading PDF bytes:", err)
		http.Error(w, "Failed to read PDF", http.StatusInternalServerError)
		return
	}

	// Call Gemini
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	rec, err := GetGeminiRecommendationsFromPDF(ctx, pdfBytes)
	if err != nil {
		log.Println("Gemini error:", err)
		http.Error(w, "Failed to get recommendations from Gemini", http.StatusInternalServerError)
		return
	}

	// Success → return GeminiRecommendation as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rec)
}

// CORS middleware for Next.js on http://localhost:3000
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/upload-report", uploadReportHandler)

	handler := withCORS(mux)

	port := ":8080"
	log.Printf("Backend running on http://localhost%v\n", port)
	if err := http.ListenAndServe(port, handler); err != nil {
		log.Fatal("Server failed:", err)
	}
}
