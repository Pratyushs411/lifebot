package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"google.golang.org/genai"
)

type GeminiRecommendation struct {
	DietRecommendation string `json:"dietRecommendation"`
	DoctorCategory     string `json:"doctorCategory"`
	Notes              string `json:"notes"`
}

// GetGeminiRecommendationsFromPDF sends the raw PDF bytes + prompt to Gemini.
func GetGeminiRecommendationsFromPDF(ctx context.Context, pdfData []byte) (*GeminiRecommendation, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY env var is not set")
	}

	// Create Gemini client for Gemini API backend (no Close() needed).
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Prompt: ask for JSON only (dietRecommendation, doctorCategory, notes)
	prompt := `
You are a medical assistant.
You will be given a medical report PDF.

Read the PDF and return a JSON object ONLY (no extra text) in this exact format:

{
  "dietRecommendation": "short, patient-friendly diet advice in 2-3 sentences",
  "doctorCategory": "what type of doctor the patient should consult (e.g. General Physician, Cardiologist, Endocrinologist, etc.)",
  "notes": "brief additional notes about what to watch out for, in 2-4 sentences"
}
`

	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: prompt},
				{
					InlineData: &genai.Blob{
						Data:     pdfData,
						MIMEType: "application/pdf",
					},
				},
			},
		},
	}

	const modelName = "gemini-2.5-flash-lite"

	result, err := client.Models.GenerateContent(ctx, modelName, contents, &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("Gemini GenerateContent failed: %w", err)
	}

	// Handy helper: join all candidate text into one string
	text := result.Text()
	if text == "" {
		return nil, fmt.Errorf("Gemini returned empty response")
	}

	// Try to parse JSON exactly into our struct.
	var rec GeminiRecommendation
	if err := json.Unmarshal([]byte(text), &rec); err != nil {
		// If model didn't obey pure JSON, fall back: put raw into notes
		return &GeminiRecommendation{
			DietRecommendation: "",
			DoctorCategory:     "",
			Notes:              text,
		}, nil
	}

	return &rec, nil
}
