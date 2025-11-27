package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"google.golang.org/genai"
)

// One health parameter extracted from the report.
type HealthParameter struct {
	Name  string `json:"name"`  // e.g. "Hemoglobin"
	Value string `json:"value"` // e.g. "10.7"
	Unit  string `json:"unit"`  // e.g. "g/dL"
	Flag  string `json:"flag"`  // e.g. "low", "high", "normal", or "unknown"
}

// Full response we want from Gemini.
type GeminiRecommendation struct {
	PatientName        string            `json:"patientName"`
	Parameters         []HealthParameter `json:"parameters"`
	DietRecommendation string            `json:"dietRecommendation"`
	DoctorCategory     string            `json:"doctorCategory"`
	Notes              string            `json:"notes"`
}

// GetGeminiRecommendationsFromPDF sends the raw PDF bytes + prompt to Gemini.
func GetGeminiRecommendationsFromPDF(ctx context.Context, pdfData []byte) (*GeminiRecommendation, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY env var is not set")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Prompt: ask for structured JSON with patient + parameters + recommendations.
	prompt := `
You are a medical assistant.

You will receive a medical report PDF.
Your task is to:
1) Identify the patient's name if present.
2) Extract key health/lab parameters with their numeric values and units.
3) Give a short diet recommendation.
4) Suggest a suitable doctor category.
5) Add brief notes about overall condition.

Return ONLY a JSON object in this exact structure (no extra text):

{
  "patientName": "Full name of the patient, or empty string if unknown",
  "parameters": [
    {
      "name": "Hemoglobin",
      "value": "10.7",
      "unit": "g/dL",
      "flag": "low"
    },
    {
      "name": "Platelet count",
      "value": "1.50",
      "unit": "lakh/mm3",
      "flag": "normal"
    }
  ],
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

	text := result.Text()
	if text == "" {
		return nil, fmt.Errorf("Gemini returned empty response")
	}

	var rec GeminiRecommendation
	if err := json.Unmarshal([]byte(text), &rec); err != nil {
		// If model didn't obey pure JSON, fall back and at least show its raw text.
		return &GeminiRecommendation{
			PatientName:        "",
			Parameters:         nil,
			DietRecommendation: "",
			DoctorCategory:     "",
			Notes:              text,
		}, nil
	}

	return &rec, nil
}

// GetGeminiRecommendationsForVitals uses averaged live sensor values (from ESP-32)
// to get diet recommendation, doctor category and notes from Gemini.
//
// avg comes from your live.go (type LiveSample) and contains:
//
//	Spo2 (percent), Temp (Fahrenheit), Ecg (relative), Gsr (relative).
func GetGeminiRecommendationsForVitals(ctx context.Context, avg *LiveSample) (*GeminiRecommendation, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY env var is not set")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// NOTE: temperature from ESP is in Fahrenheit (same as your ESP code).
	prompt := fmt.Sprintf(`
You are a medical assistant.

You will be given the patient's current vital signs from wearable sensors:

SpO2: %.1f %%
Temperature: %.1f °F
ECG (relative value): %.1f
GSR (relative value): %.1f

Based on these values:

1) Give a short, patient-friendly diet recommendation in 2–3 sentences.
2) Suggest the most appropriate doctor category (e.g. General Physician, Cardiologist, Endocrinologist, etc.).
3) Provide brief notes in 2–4 sentences about what to monitor and when to seek urgent care.

Return ONLY a JSON object with this structure (no extra text):

{
  "dietRecommendation": "string",
  "doctorCategory": "string",
  "notes": "string"
}
`, avg.Spo2, avg.Temp, avg.Ecg, avg.Gsr)

	contents := []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: prompt},
			},
		},
	}

	const modelName = "gemini-2.5-flash-lite"

	result, err := client.Models.GenerateContent(ctx, modelName, contents, &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("Gemini GenerateContent (vitals) failed: %w", err)
	}

	text := result.Text()
	if text == "" {
		return nil, fmt.Errorf("Gemini (vitals) returned empty response")
	}

	// Expect JSON: { "dietRecommendation": "...", "doctorCategory": "...", "notes": "..." }
	var tmp struct {
		DietRecommendation string `json:"dietRecommendation"`
		DoctorCategory     string `json:"doctorCategory"`
		Notes              string `json:"notes"`
	}

	if err := json.Unmarshal([]byte(text), &tmp); err != nil {
		// Fallback: if it didn't obey pure JSON, at least return raw text in Notes
		return &GeminiRecommendation{
			PatientName:        "",
			Parameters:         nil,
			DietRecommendation: "",
			DoctorCategory:     "",
			Notes:              text,
		}, nil
	}

	// We only fill the recommendation fields here;
	// patient name and parameters are set by liveReadHandler.
	return &GeminiRecommendation{
		PatientName:        "",
		Parameters:         nil,
		DietRecommendation: tmp.DietRecommendation,
		DoctorCategory:     tmp.DoctorCategory,
		Notes:              tmp.Notes,
	}, nil
}
