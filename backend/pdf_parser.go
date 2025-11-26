package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

// ExtractTextFromPDF reads the uploaded PDF and returns all text.
func ExtractTextFromPDF(r io.Reader) (string, error) {
	// Read the uploaded file into memory
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return "", fmt.Errorf("failed to read PDF bytes: %w", err)
	}

	// Create a PDF reader from bytes
	pdfReader, err := model.NewPdfReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return "", fmt.Errorf("failed to create PDF reader: %w", err)
	}

	// Check encryption properly (2 return values)
	enc, err := pdfReader.IsEncrypted()
	if err != nil {
		return "", fmt.Errorf("failed checking encryption: %w", err)
	}

	if enc {
		ok, err := pdfReader.Decrypt([]byte(""))
		if err != nil {
			return "", fmt.Errorf("failed to decrypt PDF (empty password): %w", err)
		}
		if !ok {
			return "", fmt.Errorf("PDF appears to be password-protected and cannot be read")
		}
	}

	// Get page count
	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return "", fmt.Errorf("failed to get page count: %w", err)
	}

	var sb strings.Builder

	// Extract text page by page
	for i := 1; i <= numPages; i++ {
		page, err := pdfReader.GetPage(i)
		if err != nil {
			continue
		}

		ex, err := extractor.New(page)
		if err != nil {
			continue
		}

		text, err := ex.ExtractText()
		if err != nil {
			continue
		}

		sb.WriteString(text)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
