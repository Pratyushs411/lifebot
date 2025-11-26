package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type Report struct {
	ID                 string    `json:"id"`
	CreatedAt          time.Time `json:"createdAt"`
	Source             string    `json:"source"`
	RawPreview         string    `json:"rawPreview"`
	Spo2               float64   `json:"spo2"`
	Gsr                float64   `json:"gsr"`
	Bp                 string    `json:"bp"`
	Temp               float64   `json:"temp"`
	HeartRate          float64   `json:"heartRate"`
	DietRecommendation string    `json:"dietRecommendation"`
	DoctorCategory     string    `json:"doctorCategory"`
	Notes              string    `json:"notes"`

	Hemoglobin    string `json:"hemoglobin"`
	PlateletCount string `json:"plateletCount"`
	ESR           string `json:"esr"`
	WidalStatus   string `json:"widalStatus"`
	MalariaStatus string `json:"malariaStatus"`
}

// Save a report under /reports/<id>
func SaveReportToFirebase(rep *Report) error {
	// ensure CreatedAt is set
	if rep.CreatedAt.IsZero() {
		rep.CreatedAt = time.Now()
	}
	if rep.ID == "" {
		rep.ID = rep.CreatedAt.Format("20060102150405")
	}
	path := fmt.Sprintf("reports/%s", rep.ID)
	return FirebaseSet(path, rep)
}

// Get latest report by CreatedAt
func GetLatestReport() (*Report, error) {
	var raw map[string]json.RawMessage

	if err := FirebaseGet("reports", &raw); err != nil {
		return nil, fmt.Errorf("firebase get reports failed: %w", err)
	}

	if len(raw) == 0 {
		return nil, nil // no reports yet
	}

	var latest *Report

	for id, blob := range raw {
		var r Report
		if err := json.Unmarshal(blob, &r); err != nil {
			// Some old/bad data might not match Report structure → skip it
			log.Println("Skipping invalid report entry", id, "error:", err)
			continue
		}
		r.ID = id

		if latest == nil || r.CreatedAt.After(latest.CreatedAt) {
			tmp := r
			latest = &tmp
		}
	}

	return latest, nil
}
