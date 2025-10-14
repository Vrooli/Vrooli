package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ExtractRequest represents the input for AI extraction
type ExtractRequest struct {
	TextContent   string `json:"text_content"`
	ClientContext string `json:"client_context,omitempty"`
	CurrencyHint  string `json:"currency_hint,omitempty"`
}

// ExtractedData represents the structured data extracted from text
type ExtractedData struct {
	ClientInfo *ExtractedClient   `json:"client_info"`
	LineItems  []ExtractedLineItem `json:"line_items"`
	Dates      *ExtractedDates    `json:"dates"`
	Amounts    *ExtractedAmounts  `json:"amounts"`
}

// ExtractedClient represents client information from text
type ExtractedClient struct {
	Name    string `json:"name"`
	Email   string `json:"email,omitempty"`
	Phone   string `json:"phone,omitempty"`
	Address string `json:"address,omitempty"`
}

// ExtractedLineItem represents an invoice line item from text
type ExtractedLineItem struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Unit        string  `json:"unit,omitempty"`
}

// ExtractedDates represents date information from text
type ExtractedDates struct {
	IssueDate string `json:"issue_date,omitempty"`
	DueDate   string `json:"due_date,omitempty"`
}

// ExtractedAmounts represents monetary information from text
type ExtractedAmounts struct {
	Currency   string  `json:"currency"`
	Subtotal   float64 `json:"subtotal"`
	Tax        float64 `json:"tax,omitempty"`
	TaxRate    float64 `json:"tax_rate,omitempty"`
	Total      float64 `json:"total"`
}

// ExtractResponse represents the AI extraction API response
type ExtractResponse struct {
	ExtractedData   ExtractedData `json:"extracted_data"`
	ConfidenceScore float64       `json:"confidence_score"`
	Suggestions     []string      `json:"suggestions,omitempty"`
}

// OllamaRequest represents a request to Ollama API
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaResponse represents a response from Ollama API
type OllamaResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

// extractInvoiceData handles the /api/invoices/extract endpoint
func extractInvoiceDataHandler(w http.ResponseWriter, r *http.Request) {
	var req ExtractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.TextContent == "" {
		http.Error(w, "text_content is required", http.StatusBadRequest)
		return
	}

	// Extract invoice data using AI
	extractedData, confidence, suggestions, err := extractWithAI(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("AI extraction failed: %v", err), http.StatusInternalServerError)
		return
	}

	response := ExtractResponse{
		ExtractedData:   extractedData,
		ConfidenceScore: confidence,
		Suggestions:     suggestions,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// extractWithAI uses Ollama to extract structured invoice data from unstructured text
func extractWithAI(req ExtractRequest) (ExtractedData, float64, []string, error) {
	// Validate input
	if req.TextContent == "" {
		return ExtractedData{}, 0, nil, fmt.Errorf("text_content is required")
	}

	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	// Build prompt for invoice data extraction
	prompt := buildExtractionPrompt(req)

	// Call Ollama API
	ollamaReq := OllamaRequest{
		Model:  "llama3.2",
		Prompt: prompt,
		Stream: false,
	}

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return ExtractedData{}, 0, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Post(
		ollamaURL+"/api/generate",
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return ExtractedData{}, 0, nil, fmt.Errorf("ollama API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return ExtractedData{}, 0, nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return ExtractedData{}, 0, nil, fmt.Errorf("failed to decode ollama response: %w", err)
	}

	// Parse the AI response to extract structured data
	extractedData, confidence, suggestions := parseAIResponse(ollamaResp.Response, req)

	return extractedData, confidence, suggestions, nil
}

// buildExtractionPrompt creates a prompt for the AI model
func buildExtractionPrompt(req ExtractRequest) string {
	var sb strings.Builder

	sb.WriteString("You are an invoice data extraction assistant. Extract structured invoice information from the following text.\n\n")
	sb.WriteString("Text to analyze:\n")
	sb.WriteString(req.TextContent)
	sb.WriteString("\n\n")

	if req.ClientContext != "" {
		sb.WriteString("Client context: ")
		sb.WriteString(req.ClientContext)
		sb.WriteString("\n\n")
	}

	if req.CurrencyHint != "" {
		sb.WriteString("Expected currency: ")
		sb.WriteString(req.CurrencyHint)
		sb.WriteString("\n\n")
	}

	sb.WriteString("Extract and return ONLY a valid JSON object with this structure:\n")
	sb.WriteString(`{
  "client_info": {
    "name": "string",
    "email": "string (optional)",
    "phone": "string (optional)",
    "address": "string (optional)"
  },
  "line_items": [
    {
      "description": "string",
      "quantity": number,
      "unit_price": number,
      "unit": "string (optional, e.g., hours, units, items)"
    }
  ],
  "dates": {
    "issue_date": "YYYY-MM-DD (optional)",
    "due_date": "YYYY-MM-DD (optional)"
  },
  "amounts": {
    "currency": "USD or other currency code",
    "subtotal": number,
    "tax": number (optional),
    "tax_rate": number (optional, as decimal like 0.08 for 8%)",
    "total": number
  }
}

Rules:
1. Return ONLY valid JSON, no additional text
2. If information is missing, omit the field or use null
3. All monetary amounts should be numbers, not strings
4. For quantities like "10 hours", extract: quantity=10, unit="hours"
5. Calculate totals when possible: line_total = quantity * unit_price
6. If tax rate is mentioned (e.g., "8% tax"), include it as decimal (0.08)
7. Extract dates in YYYY-MM-DD format if mentioned
8. Use ISO 4217 currency codes (USD, EUR, GBP, etc.)

Return the JSON now:
`)

	return sb.String()
}

// parseAIResponse converts the AI's text response into structured data
func parseAIResponse(aiResponse string, req ExtractRequest) (ExtractedData, float64, []string) {
	// Initialize defaults
	extractedData := ExtractedData{
		ClientInfo: &ExtractedClient{},
		LineItems:  []ExtractedLineItem{},
		Dates:      &ExtractedDates{},
		Amounts:    &ExtractedAmounts{Currency: "USD"},
	}
	suggestions := []string{}
	confidence := 0.5 // Default medium confidence

	// Try to extract JSON from the response
	jsonStart := strings.Index(aiResponse, "{")
	jsonEnd := strings.LastIndex(aiResponse, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonStart >= jsonEnd {
		// Fallback: try basic text parsing
		suggestions = append(suggestions, "AI response did not contain valid JSON. Using fallback text parsing.")
		return fallbackTextExtraction(req.TextContent, req.CurrencyHint), 0.3, suggestions
	}

	jsonStr := aiResponse[jsonStart : jsonEnd+1]

	// Parse the JSON response
	var rawData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &rawData); err != nil {
		suggestions = append(suggestions, fmt.Sprintf("JSON parsing failed: %v. Using fallback extraction.", err))
		return fallbackTextExtraction(req.TextContent, req.CurrencyHint), 0.3, suggestions
	}

	// Extract client info
	if clientInfo, ok := rawData["client_info"].(map[string]interface{}); ok {
		if name, ok := clientInfo["name"].(string); ok && name != "" {
			extractedData.ClientInfo.Name = name
			confidence += 0.15
		}
		if email, ok := clientInfo["email"].(string); ok {
			extractedData.ClientInfo.Email = email
		}
		if phone, ok := clientInfo["phone"].(string); ok {
			extractedData.ClientInfo.Phone = phone
		}
		if address, ok := clientInfo["address"].(string); ok {
			extractedData.ClientInfo.Address = address
		}
	}

	// Extract line items
	if lineItems, ok := rawData["line_items"].([]interface{}); ok && len(lineItems) > 0 {
		for _, item := range lineItems {
			if itemMap, ok := item.(map[string]interface{}); ok {
				lineItem := ExtractedLineItem{}
				if desc, ok := itemMap["description"].(string); ok {
					lineItem.Description = desc
				}
				if qty, ok := itemMap["quantity"].(float64); ok {
					lineItem.Quantity = qty
				}
				if price, ok := itemMap["unit_price"].(float64); ok {
					lineItem.UnitPrice = price
				}
				if unit, ok := itemMap["unit"].(string); ok {
					lineItem.Unit = unit
				}

				// Only add if we have at least description and price
				if lineItem.Description != "" && (lineItem.UnitPrice > 0 || lineItem.Quantity > 0) {
					extractedData.LineItems = append(extractedData.LineItems, lineItem)
					confidence += 0.20 / float64(len(lineItems)) // Split confidence across items
				}
			}
		}
	}

	// Extract dates
	if dates, ok := rawData["dates"].(map[string]interface{}); ok {
		if issueDate, ok := dates["issue_date"].(string); ok && issueDate != "" {
			extractedData.Dates.IssueDate = issueDate
		}
		if dueDate, ok := dates["due_date"].(string); ok && dueDate != "" {
			extractedData.Dates.DueDate = dueDate
		}
	}

	// Extract amounts
	if amounts, ok := rawData["amounts"].(map[string]interface{}); ok {
		if currency, ok := amounts["currency"].(string); ok && currency != "" {
			extractedData.Amounts.Currency = currency
			confidence += 0.05
		}
		if subtotal, ok := amounts["subtotal"].(float64); ok {
			extractedData.Amounts.Subtotal = subtotal
			confidence += 0.10
		}
		if tax, ok := amounts["tax"].(float64); ok {
			extractedData.Amounts.Tax = tax
		}
		if taxRate, ok := amounts["tax_rate"].(float64); ok {
			extractedData.Amounts.TaxRate = taxRate
		}
		if total, ok := amounts["total"].(float64); ok {
			extractedData.Amounts.Total = total
			confidence += 0.10
		}
	}

	// Add suggestions based on what's missing
	if extractedData.ClientInfo.Name == "" {
		suggestions = append(suggestions, "Could not extract client name. Please verify client information.")
	}
	if len(extractedData.LineItems) == 0 {
		suggestions = append(suggestions, "No line items extracted. Please add invoice items manually.")
	}
	if extractedData.Amounts.Total == 0 {
		suggestions = append(suggestions, "Could not extract total amount. Please verify pricing.")
	}
	if extractedData.Dates.DueDate == "" {
		suggestions = append(suggestions, "No due date found. Consider adding a payment term.")
	}

	// Cap confidence at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return extractedData, confidence, suggestions
}

// fallbackTextExtraction provides basic extraction when AI parsing fails
func fallbackTextExtraction(text string, currencyHint string) ExtractedData {
	extractedData := ExtractedData{
		ClientInfo: &ExtractedClient{},
		LineItems:  []ExtractedLineItem{},
		Dates:      &ExtractedDates{},
		Amounts:    &ExtractedAmounts{Currency: "USD"},
	}

	if currencyHint != "" {
		extractedData.Amounts.Currency = currencyHint
	}

	// Try to find dollar amounts using simple regex-like patterns
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for amounts like "$1500", "$1,500", "1500 dollars"
		if strings.Contains(line, "$") || strings.Contains(strings.ToLower(line), "dollar") {
			// Extract numbers from the line - look for patterns like $1500 or $1,500
			words := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(line, ",", ""), "$", " $ "))
			for i, word := range words {
				if word == "$" && i+1 < len(words) {
					// Next word after $ should be the amount
					var amount float64
					if _, err := fmt.Sscanf(words[i+1], "%f", &amount); err == nil && amount > 0 {
						if extractedData.Amounts.Total == 0 {
							extractedData.Amounts.Total = amount
							extractedData.Amounts.Subtotal = amount
							break
						}
					}
				} else {
					// Try direct parsing for words containing numbers
					var amount float64
					if _, err := fmt.Sscanf(word, "%f", &amount); err == nil && amount > 0 {
						if extractedData.Amounts.Total == 0 {
							extractedData.Amounts.Total = amount
							extractedData.Amounts.Subtotal = amount
							break
						}
					}
				}
			}
		}

		// Try to find names (first capitalized word after common keywords)
		if extractedData.ClientInfo.Name == "" {
			lowerLine := strings.ToLower(line)
			// Look for patterns like "Bill [Name]", "Invoice [Name]", etc.
			for _, keyword := range []string{"bill ", "invoice ", "for ", "to "} {
				if idx := strings.Index(lowerLine, keyword); idx != -1 {
					remaining := line[idx+len(keyword):]
					words := strings.Fields(remaining)
					if len(words) >= 1 && len(words[0]) > 0 && words[0][0] >= 'A' && words[0][0] <= 'Z' {
						if len(words) >= 2 && words[1][0] >= 'A' && words[1][0] <= 'Z' {
							extractedData.ClientInfo.Name = words[0] + " " + words[1]
						} else {
							extractedData.ClientInfo.Name = words[0]
						}
						break
					}
				}
			}
		}
	}

	// If we extracted an amount, create a generic line item
	if extractedData.Amounts.Total > 0 {
		extractedData.LineItems = append(extractedData.LineItems, ExtractedLineItem{
			Description: "Services rendered",
			Quantity:    1,
			UnitPrice:   extractedData.Amounts.Total,
			Unit:        "item",
		})
	}

	return extractedData
}
