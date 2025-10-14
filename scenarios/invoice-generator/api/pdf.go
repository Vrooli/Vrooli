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

	"github.com/gorilla/mux"
)

// PDFRequest represents a PDF generation request
type PDFRequest struct {
	InvoiceID string `json:"invoice_id,omitempty"`
	Template  string `json:"template"`
	Format    string `json:"format"`
	Invoice   struct {
		CompanyName    string        `json:"company_name"`
		CompanyAddress string        `json:"company_address"`
		CompanyEmail   string        `json:"company_email"`
		CompanyPhone   string        `json:"company_phone"`
		CompanyLogo    string        `json:"company_logo"`
		ClientName     string        `json:"client_name"`
		ClientEmail    string        `json:"client_email"`
		ClientAddress  string        `json:"client_address"`
		InvoiceNumber  string        `json:"invoice_number"`
		IssueDate      string        `json:"issue_date"`
		DueDate        string        `json:"due_date"`
		Items          []InvoiceItem `json:"items"`
		Currency       string        `json:"currency"`
		PaymentTerms   string        `json:"payment_terms"`
		Notes          string        `json:"notes"`
		BankDetails    string        `json:"bank_details"`
		Subtotal       float64       `json:"subtotal"`
		TaxAmount      float64       `json:"tax_amount"`
		TotalAmount    float64       `json:"total_amount"`
	} `json:"invoice"`
}

// GeneratePDFHandler generates a PDF for an invoice
func generatePDFHandler(w http.ResponseWriter, r *http.Request) {
	var req PDFRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If invoice_id provided, fetch invoice data
	if req.InvoiceID != "" {
		invoice, err := getInvoiceByID(req.InvoiceID)
		if err != nil {
			http.Error(w, "Invoice not found", http.StatusNotFound)
			return
		}
		
		// Populate request with invoice data
		req.Invoice.InvoiceNumber = invoice.InvoiceNumber
		req.Invoice.IssueDate = invoice.IssueDate
		req.Invoice.DueDate = invoice.DueDate
		req.Invoice.Currency = invoice.Currency
		req.Invoice.Notes = invoice.Notes
		req.Invoice.Items = invoice.Items
		req.Invoice.Subtotal = invoice.Subtotal
		req.Invoice.TaxAmount = invoice.TaxAmount
		req.Invoice.TotalAmount = invoice.TotalAmount
		
		// Get client info if available
		if invoice.ClientID != "" {
			client, _ := getClientByID(invoice.ClientID)
			if client != nil {
				req.Invoice.ClientName = client.Name
				req.Invoice.ClientEmail = client.Email
				req.Invoice.ClientAddress = formatAddress(client)
			}
		}
		
		// Get company info
		company, _ := getDefaultCompany()
		if company != nil {
			req.Invoice.CompanyName = company.Name
			req.Invoice.CompanyEmail = company.Email
			req.Invoice.CompanyPhone = company.Phone
			req.Invoice.CompanyAddress = formatCompanyAddress(company)
		}
	}

	// Generate HTML content
	htmlContent := generateInvoiceHTML(req)
	
	// For now, return HTML (can be converted to PDF with external service)
	if req.Format == "html" {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(htmlContent))
		return
	}
	
	// Return PDF generation info
	response := map[string]interface{}{
		"status": "success",
		"message": "PDF generation initiated",
		"invoice_id": req.InvoiceID,
		"format": req.Format,
		"template": req.Template,
		"generated_at": time.Now().Format(time.RFC3339),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GenerateInvoiceHTML generates HTML for invoice
func generateInvoiceHTML(req PDFRequest) string {
	var itemsHTML strings.Builder
	for _, item := range req.Invoice.Items {
		itemsHTML.WriteString(fmt.Sprintf(`
			<tr>
				<td>%s</td>
				<td>%.2f</td>
				<td>$%.2f</td>
				<td>$%.2f</td>
			</tr>`,
			item.Description,
			item.Quantity,
			item.UnitPrice,
			item.LineTotal,
		))
	}
	
	template := req.Template
	if template == "" {
		template = "professional"
	}
	
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Invoice %s</title>
	<style>
		body { font-family: Arial, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
		.header { display: flex; justify-content: space-between; margin-bottom: 40px; }
		.company-info { text-align: left; }
		.invoice-info { text-align: right; }
		.client-info { margin-bottom: 30px; }
		table { width: 100%%; border-collapse: collapse; margin-bottom: 20px; }
		th, td { padding: 10px; text-align: left; border-bottom: 1px solid #ddd; }
		th { background-color: #f4f4f4; font-weight: bold; }
		.totals { text-align: right; margin-top: 20px; }
		.totals-row { display: flex; justify-content: flex-end; margin: 5px 0; }
		.totals-label { width: 150px; text-align: right; margin-right: 20px; }
		.totals-value { width: 100px; text-align: right; }
		.total-amount { font-size: 1.2em; font-weight: bold; border-top: 2px solid #333; padding-top: 10px; }
		.notes { margin-top: 40px; padding: 15px; background-color: #f9f9f9; border-radius: 5px; }
		.payment-info { margin-top: 30px; }
		h1 { color: #333; }
		.%s { }
	</style>
</head>
<body>
	<div class="header">
		<div class="company-info">
			<h2>%s</h2>
			<p>%s</p>
			<p>%s</p>
			<p>%s</p>
		</div>
		<div class="invoice-info">
			<h1>INVOICE</h1>
			<p><strong>Invoice #:</strong> %s</p>
			<p><strong>Date:</strong> %s</p>
			<p><strong>Due Date:</strong> %s</p>
		</div>
	</div>
	
	<div class="client-info">
		<h3>Bill To:</h3>
		<p><strong>%s</strong></p>
		<p>%s</p>
		<p>%s</p>
	</div>
	
	<table>
		<thead>
			<tr>
				<th>Description</th>
				<th>Quantity</th>
				<th>Unit Price</th>
				<th>Amount</th>
			</tr>
		</thead>
		<tbody>
			%s
		</tbody>
	</table>
	
	<div class="totals">
		<div class="totals-row">
			<span class="totals-label">Subtotal:</span>
			<span class="totals-value">$%.2f</span>
		</div>
		<div class="totals-row">
			<span class="totals-label">Tax:</span>
			<span class="totals-value">$%.2f</span>
		</div>
		<div class="totals-row total-amount">
			<span class="totals-label">Total:</span>
			<span class="totals-value">$%.2f</span>
		</div>
	</div>
	
	<div class="notes">
		<h4>Notes:</h4>
		<p>%s</p>
	</div>
	
	<div class="payment-info">
		<h4>Payment Information:</h4>
		<p><strong>Terms:</strong> %s</p>
		<p>%s</p>
	</div>
</body>
</html>`,
		req.Invoice.InvoiceNumber,
		template,
		req.Invoice.CompanyName,
		req.Invoice.CompanyAddress,
		req.Invoice.CompanyEmail,
		req.Invoice.CompanyPhone,
		req.Invoice.InvoiceNumber,
		req.Invoice.IssueDate,
		req.Invoice.DueDate,
		req.Invoice.ClientName,
		req.Invoice.ClientEmail,
		req.Invoice.ClientAddress,
		itemsHTML.String(),
		req.Invoice.Subtotal,
		req.Invoice.TaxAmount,
		req.Invoice.TotalAmount,
		req.Invoice.Notes,
		req.Invoice.PaymentTerms,
		req.Invoice.BankDetails,
	)
}

// DownloadPDFHandler handles PDF download requests
func downloadPDFHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	invoiceID := vars["id"]
	
	invoice, err := getInvoiceByID(invoiceID)
	if err != nil {
		http.Error(w, "Invoice not found", http.StatusNotFound)
		return
	}
	
	// Generate PDF request
	req := PDFRequest{
		InvoiceID: invoiceID,
		Template: "professional",
		Format: "pdf",
	}
	
	// Generate HTML
	htmlContent := generateInvoiceHTML(req)

	// Convert HTML to PDF using browserless
	pdfBytes, err := convertHTMLToPDF(htmlContent)
	if err != nil {
		http.Error(w, fmt.Sprintf("PDF generation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Set headers for PDF download
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=invoice-%s.pdf", invoice.InvoiceNumber))

	// Return actual PDF bytes
	w.Write(pdfBytes)
}

// Helper functions
func formatAddress(client *Client) string {
	parts := []string{}
	if client.AddressLine1 != "" {
		parts = append(parts, client.AddressLine1)
	}
	if client.AddressLine2 != "" {
		parts = append(parts, client.AddressLine2)
	}
	if client.City != "" || client.StateProvince != "" || client.PostalCode != "" {
		cityLine := strings.TrimSpace(fmt.Sprintf("%s, %s %s", client.City, client.StateProvince, client.PostalCode))
		parts = append(parts, cityLine)
	}
	if client.Country != "" {
		parts = append(parts, client.Country)
	}
	return strings.Join(parts, "\n")
}

func formatCompanyAddress(company *Company) string {
	parts := []string{}
	if company.AddressLine1 != "" {
		parts = append(parts, company.AddressLine1)
	}
	if company.AddressLine2 != "" {
		parts = append(parts, company.AddressLine2)
	}
	if company.City != "" || company.StateProvince != "" || company.PostalCode != "" {
		cityLine := strings.TrimSpace(fmt.Sprintf("%s, %s %s", company.City, company.StateProvince, company.PostalCode))
		parts = append(parts, cityLine)
	}
	if company.Country != "" {
		parts = append(parts, company.Country)
	}
	return strings.Join(parts, "\n")
}

// convertHTMLToPDF converts HTML content to PDF using browserless
func convertHTMLToPDF(htmlContent string) ([]byte, error) {
	browserlessURL := os.Getenv("BROWSERLESS_URL")
	if browserlessURL == "" {
		browserlessURL = "http://localhost:4110"
	}

	// Prepare request body for browserless PDF endpoint
	reqBody := map[string]interface{}{
		"html": htmlContent,
		"options": map[string]interface{}{
			"format":              "A4",
			"printBackground":     true,
			"preferCSSPageSize":   true,
			"displayHeaderFooter": false,
			"margin": map[string]interface{}{
				"top":    "20px",
				"right":  "20px",
				"bottom": "20px",
				"left":   "20px",
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Call browserless PDF generation endpoint
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Post(
		browserlessURL+"/chrome/pdf",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("browserless API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("browserless returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read the PDF bytes
	pdfBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF response: %w", err)
	}

	return pdfBytes, nil
}