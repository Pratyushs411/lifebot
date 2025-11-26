package main

import (
	"regexp"
	"strings"
)

type LabReport struct {
	Hemoglobin    string `json:"hemoglobin"`
	PlateletCount string `json:"plateletCount"`
	ESR           string `json:"esr"`
	Widal         string `json:"widal"`
	Malaria       string `json:"malaria"`
}

func ParseLabReport(text string) LabReport {
	clean := strings.ToUpper(text)

	find := func(pattern string) string {
		re := regexp.MustCompile(pattern)
		match := re.FindStringSubmatch(clean)
		if len(match) >= 2 {
			return strings.TrimSpace(match[1])
		}
		return "N/A"
	}

	return LabReport{
		Hemoglobin:    find(`HEMOGLOBIN[^0-9]*([0-9.]+)`),
		PlateletCount: find(`PLATELET[^0-9]*([0-9]+)`),
		ESR:           find(`ESR[^0-9]*([0-9]+)`),
		Widal:         find(`WIDAL[^A-Z]*([A-Z ]+)`),
		Malaria:       find(`MALARIA[^A-Z]*([A-Z ]+)`),
	}
}
