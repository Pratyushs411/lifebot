package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// One live sample from ESP-32
type LiveSample struct {
	Timestamp time.Time `json:"timestamp"`
	Spo2      float64   `json:"spo2"`
	Temp      float64   `json:"temp"`
	Ecg       float64   `json:"ecg"`
	Gsr       float64   `json:"gsr"`
}

var (
	liveSamples []LiveSample
	liveMu      sync.Mutex
)

// POST /esp-sample
// Body JSON: { "spo2": number, "temp": number, "ecg": number, "gsr": number }
func espSampleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Spo2 float64 `json:"spo2"`
		Temp float64 `json:"temp"`
		Ecg  float64 `json:"ecg"`
		Gsr  float64 `json:"gsr"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	sample := LiveSample{
		Timestamp: time.Now(),
		Spo2:      payload.Spo2,
		Temp:      payload.Temp,
		Ecg:       payload.Ecg,
		Gsr:       payload.Gsr,
	}

	liveMu.Lock()
	defer liveMu.Unlock()

	// append sample
	liveSamples = append(liveSamples, sample)

	// keep only last 30 seconds in memory
	cutoff := time.Now().Add(-30 * time.Second)
	i := 0
	for _, s := range liveSamples {
		if s.Timestamp.After(cutoff) {
			liveSamples[i] = s
			i++
		}
	}
	liveSamples = liveSamples[:i]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ok": true,
	})
}

// Compute average over a given time window (e.g. last 10s).
func computeAverage(window time.Duration) (*LiveSample, error) {
	liveMu.Lock()
	defer liveMu.Unlock()

	cutoff := time.Now().Add(-window)

	var cnt int
	var sumSpo2, sumTemp, sumEcg, sumGsr float64

	for _, s := range liveSamples {
		if s.Timestamp.After(cutoff) {
			cnt++
			sumSpo2 += s.Spo2
			sumTemp += s.Temp
			sumEcg += s.Ecg
			sumGsr += s.Gsr
		}
	}

	if cnt == 0 {
		return nil, fmt.Errorf("no recent samples in last %v", window)
	}

	return &LiveSample{
		Timestamp: time.Now(),
		Spo2:      sumSpo2 / float64(cnt),
		Temp:      sumTemp / float64(cnt),
		Ecg:       sumEcg / float64(cnt),
		Gsr:       sumGsr / float64(cnt),
	}, nil
}

// Request body for /live-read
type liveReadRequest struct {
	PatientName string `json:"patientName"`
}

// POST /live-read
// Frontend sends: { "patientName": "Pratyush" }
// Backend:
//  1. clears buffer
//  2. waits 10 seconds while ESP pushes data
//  3. averages last 10s
//  4. calls Gemini for recommendations based on averaged vitals
//  5. returns GeminiRecommendation with PatientName + parameters
func liveReadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req liveReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	patientName := req.PatientName

	// 1) Clear existing samples so we only use fresh 10s window
	liveMu.Lock()
	liveSamples = nil
	liveMu.Unlock()

	// 2) Wait 10 seconds while ESP is sending /esp-sample
	time.Sleep(10 * time.Second)

	// 3) Compute average over last 10s
	avg, err := computeAverage(10 * time.Second)
	if err != nil {
		http.Error(w, "Not enough live data from ESP (need ~10s of readings)", http.StatusBadRequest)
		return
	}

	// 4) Ask Gemini for recommendations based on averaged vitals
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	rec, err := GetGeminiRecommendationsForVitals(ctx, avg)
	if err != nil {
		http.Error(w, "Failed to get recommendations from Gemini", http.StatusInternalServerError)
		return
	}

	// 5) Build parameters table for frontend
	params := []HealthParameter{
		{Name: "SpO₂", Value: fmt.Sprintf("%.1f", avg.Spo2), Unit: "%", Flag: ""},
		{Name: "Temperature", Value: fmt.Sprintf("%.1f", avg.Temp), Unit: "°F", Flag: ""},
		{Name: "ECG", Value: fmt.Sprintf("%.1f", avg.Ecg), Unit: "rel", Flag: ""},
		{Name: "GSR", Value: fmt.Sprintf("%.1f", avg.Gsr), Unit: "rel", Flag: ""},
	}

	// Attach to same struct used for PDF flow
	resp := GeminiRecommendation{
		PatientName:        patientName,
		Parameters:         params,
		DietRecommendation: rec.DietRecommendation,
		DoctorCategory:     rec.DoctorCategory,
		Notes:              rec.Notes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
