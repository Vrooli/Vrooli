package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"math"
)

type InvoiceProcessor struct {
	db *sql.DB
}

type InvoiceProcessRequest struct {
	Action  string      `json:"action"` // create, update, read
	Invoice InvoiceData `json:"invoice"`
}

type InvoiceData struct {
	InvoiceNumber  string        `json:"invoice_number"`
	ClientName     string        `json:"client_name"`
	ClientEmail    string        `json:"client_email"`
	ClientAddress  string        `json:"client_address"`
	IssueDate      string        `json:"issue_date"`
	DueDate        string        `json:"due_date"`
	Items          []InvoiceItem `json:"items"`
	Currency       string        `json:"currency"`
	PaymentTerms   string        `json:"payment_terms"`
	Notes          string        `json:"notes"`
	Status         string        `json:"status"`
	Subtotal       float64       `json:"subtotal"`
	TotalTax       float64       `json:"total_tax"`
	Total          float64       `json:"total"`
	CreatedAt      string        `json:"created_at"`
}

type ProcessedInvoiceItem struct {
	InvoiceItem
	ItemTotal    float64 `json:"item_total"`
	TaxAmount    float64 `json:"tax_amount"`
	TotalWithTax float64 `json:"total_with_tax"`
}

type InvoiceProcessResponse struct {
	Success bool        `json:"success"`
	Invoice InvoiceData `json:"invoice"`
	Message string      `json:"message"`
	Error   string      `json:"error,omitempty"`
}

type PaymentRequest struct {
	Action        string      `json:"action"` // record_payment, update_payment, get_payments
	InvoiceNumber string      `json:"invoice_number"`
	Payment       PaymentData `json:"payment,omitempty"`
}

type PaymentData struct {
	ID            int     `json:"id,omitempty"`
	Amount        float64 `json:"amount"`
	PaymentDate   string  `json:"payment_date"`
	PaymentMethod string  `json:"payment_method"`
	Reference     string  `json:"reference"`
	Notes         string  `json:"notes"`
}

type PaymentResponse struct {
	Success          bool          `json:"success"`
	Payment          PaymentData   `json:"payment,omitempty"`
	Payments         []PaymentData `json:"payments,omitempty"`
	InvoiceStatus    string        `json:"invoice_status"`
	RemainingBalance float64       `json:"remaining_balance"`
	Message          string        `json:"message"`
	Error            string        `json:"error,omitempty"`
}

type RecurringInvoice struct {
	ID               int     `json:"id"`
	ClientName       string  `json:"client_name"`
	ClientEmail      string  `json:"client_email"`
	ClientAddress    string  `json:"client_address"`
	TemplateData     string  `json:"template_data"` // JSON string of invoice template
	Frequency        string  `json:"frequency"`     // monthly, quarterly, yearly
	NextInvoiceDate  string  `json:"next_invoice_date"`
	LastGeneratedAt  string  `json:"last_generated_at"`
	IsActive         bool    `json:"is_active"`
	CreatedAt        string  `json:"created_at"`
}

type RecurringInvoiceResponse struct {
	Success           bool               `json:"success"`
	RecurringInvoices []RecurringInvoice `json:"recurring_invoices,omitempty"`
	GeneratedInvoices []InvoiceData      `json:"generated_invoices,omitempty"`
	Message           string             `json:"message"`
	Error             string             `json:"error,omitempty"`
}

func NewInvoiceProcessor(db *sql.DB) *InvoiceProcessor {
	return &InvoiceProcessor{
		db: db,
	}
}

// ProcessInvoice handles invoice creation, updates, and validation (replaces invoice-processor workflow)
func (ip *InvoiceProcessor) ProcessInvoice(ctx context.Context, req InvoiceProcessRequest) (*InvoiceProcessResponse, error) {
	// Validate input
	if req.Action == "" {
		req.Action = "create"
	}

	invoice := req.Invoice

	// Validate required fields
	if invoice.ClientName == "" {
		return &InvoiceProcessResponse{
			Success: false,
			Error:   "client_name is required",
		}, nil
	}

	if len(invoice.Items) == 0 {
		return &InvoiceProcessResponse{
			Success: false,
			Error:   "invoice must have at least one item",
		}, nil
	}

	// Set defaults
	if invoice.InvoiceNumber == "" {
		invoice.InvoiceNumber = fmt.Sprintf("INV-%d", time.Now().Unix())
	}

	if invoice.IssueDate == "" {
		invoice.IssueDate = time.Now().Format("2006-01-02")
	}

	if invoice.DueDate == "" {
		dueDate := time.Now().AddDate(0, 0, 30) // 30 days from now
		invoice.DueDate = dueDate.Format("2006-01-02")
	}

	if invoice.Currency == "" {
		invoice.Currency = "USD"
	}

	if invoice.Status == "" {
		invoice.Status = "draft"
	}

	if invoice.CreatedAt == "" {
		invoice.CreatedAt = time.Now().Format(time.RFC3339)
	}

	// Calculate totals
	_, subtotal, totalTax := ip.calculateInvoiceTotals(invoice.Items)
	invoice.Subtotal = subtotal
	invoice.TotalTax = totalTax
	invoice.Total = subtotal + totalTax

	// Process based on action
	switch req.Action {
	case "create":
		err := ip.createInvoice(ctx, invoice)
		if err != nil {
			return &InvoiceProcessResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to create invoice: %v", err),
			}, nil
		}

		return &InvoiceProcessResponse{
			Success: true,
			Invoice: invoice,
			Message: "Invoice created successfully",
		}, nil

	case "update":
		err := ip.updateInvoice(ctx, invoice)
		if err != nil {
			return &InvoiceProcessResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to update invoice: %v", err),
			}, nil
		}

		return &InvoiceProcessResponse{
			Success: true,
			Invoice: invoice,
			Message: "Invoice updated successfully",
		}, nil

	case "read":
		storedInvoice, err := ip.getInvoice(ctx, invoice.InvoiceNumber)
		if err != nil {
			return &InvoiceProcessResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to retrieve invoice: %v", err),
			}, nil
		}

		return &InvoiceProcessResponse{
			Success: true,
			Invoice: *storedInvoice,
			Message: "Invoice retrieved successfully",
		}, nil

	default:
		return &InvoiceProcessResponse{
			Success: false,
			Error:   fmt.Sprintf("unknown action: %s", req.Action),
		}, nil
	}
}

// TrackPayment handles payment recording and tracking (replaces payment-tracker workflow)
func (ip *InvoiceProcessor) TrackPayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	if req.InvoiceNumber == "" {
		return &PaymentResponse{
			Success: false,
			Error:   "invoice_number is required",
		}, nil
	}

	// Get invoice to check current status
	invoice, err := ip.getInvoice(ctx, req.InvoiceNumber)
	if err != nil {
		return &PaymentResponse{
			Success: false,
			Error:   fmt.Sprintf("invoice not found: %v", err),
		}, nil
	}

	switch req.Action {
	case "record_payment":
		if req.Payment.Amount <= 0 {
			return &PaymentResponse{
				Success: false,
				Error:   "payment amount must be greater than 0",
			}, nil
		}

		// Set defaults for payment
		payment := req.Payment
		if payment.PaymentDate == "" {
			payment.PaymentDate = time.Now().Format("2006-01-02")
		}
		if payment.PaymentMethod == "" {
			payment.PaymentMethod = "other"
		}

		// Record payment
		query := `
			INSERT INTO payments (invoice_number, amount, payment_date, payment_method, reference, notes, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id`

		err := ip.db.QueryRowContext(ctx, query,
			req.InvoiceNumber, payment.Amount, payment.PaymentDate,
			payment.PaymentMethod, payment.Reference, payment.Notes, time.Now()).
			Scan(&payment.ID)

		if err != nil {
			return &PaymentResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to record payment: %v", err),
			}, nil
		}

		// Calculate new balance and update invoice status
		totalPaid := ip.getTotalPaid(ctx, req.InvoiceNumber)
		remainingBalance := invoice.Total - totalPaid
		
		var newStatus string
		if remainingBalance <= 0 {
			newStatus = "paid"
		} else if totalPaid > 0 {
			newStatus = "partially_paid"
		} else {
			newStatus = invoice.Status
		}

		// Update invoice status if changed
		if newStatus != invoice.Status {
			_, err = ip.db.ExecContext(ctx,
				"UPDATE invoices SET status = $1, updated_at = $2 WHERE invoice_number = $3",
				newStatus, time.Now(), req.InvoiceNumber)
			if err != nil {
				log.Printf("Warning: Failed to update invoice status: %v", err)
			}
		}

		return &PaymentResponse{
			Success:          true,
			Payment:          payment,
			InvoiceStatus:    newStatus,
			RemainingBalance: math.Max(0, remainingBalance),
			Message:          "Payment recorded successfully",
		}, nil

	case "get_payments":
		payments, err := ip.getPayments(ctx, req.InvoiceNumber)
		if err != nil {
			return &PaymentResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to get payments: %v", err),
			}, nil
		}

		totalPaid := ip.getTotalPaid(ctx, req.InvoiceNumber)
		remainingBalance := invoice.Total - totalPaid

		return &PaymentResponse{
			Success:          true,
			Payments:         payments,
			InvoiceStatus:    invoice.Status,
			RemainingBalance: math.Max(0, remainingBalance),
			Message:          "Payments retrieved successfully",
		}, nil

	default:
		return &PaymentResponse{
			Success: false,
			Error:   fmt.Sprintf("unknown action: %s", req.Action),
		}, nil
	}
}

// ProcessRecurringInvoices handles recurring invoice generation (replaces recurring-invoice-handler workflow)
func (ip *InvoiceProcessor) ProcessRecurringInvoices(ctx context.Context) (*RecurringInvoiceResponse, error) {
	// Get due recurring invoices
	query := `
		SELECT id, client_name, client_email, client_address, template_data, 
		       frequency, next_invoice_date, last_generated_at, is_active, created_at
		FROM recurring_invoices 
		WHERE is_active = true AND next_invoice_date <= CURRENT_DATE`

	rows, err := ip.db.QueryContext(ctx, query)
	if err != nil {
		return &RecurringInvoiceResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to query recurring invoices: %v", err),
		}, nil
	}
	defer rows.Close()

	var recurringInvoices []RecurringInvoice
	var generatedInvoices []InvoiceData

	for rows.Next() {
		var ri RecurringInvoice
		var templateData string

		err := rows.Scan(&ri.ID, &ri.ClientName, &ri.ClientEmail, &ri.ClientAddress,
			&templateData, &ri.Frequency, &ri.NextInvoiceDate, &ri.LastGeneratedAt,
			&ri.IsActive, &ri.CreatedAt)

		if err != nil {
			continue
		}

		ri.TemplateData = templateData

		// Parse template data to create new invoice
		var template InvoiceData
		if err := json.Unmarshal([]byte(templateData), &template); err != nil {
			log.Printf("Failed to parse template data for recurring invoice %d: %v", ri.ID, err)
			continue
		}

		// Generate new invoice
		newInvoice := template
		newInvoice.InvoiceNumber = fmt.Sprintf("REC-%d-%d", ri.ID, time.Now().Unix())
		newInvoice.IssueDate = time.Now().Format("2006-01-02")
		newInvoice.ClientName = ri.ClientName
		newInvoice.ClientEmail = ri.ClientEmail
		newInvoice.ClientAddress = ri.ClientAddress

		// Set due date based on payment terms
		dueDate := time.Now().AddDate(0, 0, 30) // Default 30 days
		newInvoice.DueDate = dueDate.Format("2006-01-02")
		newInvoice.Status = "sent"
		newInvoice.CreatedAt = time.Now().Format(time.RFC3339)

		// Calculate totals
		_, subtotal, totalTax := ip.calculateInvoiceTotals(newInvoice.Items)
		newInvoice.Subtotal = subtotal
		newInvoice.TotalTax = totalTax
		newInvoice.Total = subtotal + totalTax

		// Create the invoice
		err = ip.createInvoice(ctx, newInvoice)
		if err != nil {
			log.Printf("Failed to create recurring invoice: %v", err)
			continue
		}

		generatedInvoices = append(generatedInvoices, newInvoice)

		// Calculate next invoice date
		nextDate := ip.calculateNextInvoiceDate(ri.NextInvoiceDate, ri.Frequency)

		// Update recurring invoice record
		_, err = ip.db.ExecContext(ctx,
			`UPDATE recurring_invoices 
			 SET next_invoice_date = $1, last_generated_at = $2, updated_at = $3
			 WHERE id = $4`,
			nextDate, time.Now(), time.Now(), ri.ID)

		if err != nil {
			log.Printf("Failed to update recurring invoice %d: %v", ri.ID, err)
		}

		recurringInvoices = append(recurringInvoices, ri)
	}

	return &RecurringInvoiceResponse{
		Success:           true,
		RecurringInvoices: recurringInvoices,
		GeneratedInvoices: generatedInvoices,
		Message:           fmt.Sprintf("Processed %d recurring invoices, generated %d new invoices", len(recurringInvoices), len(generatedInvoices)),
	}, nil
}

// Helper methods

func (ip *InvoiceProcessor) calculateInvoiceTotals(items []InvoiceItem) ([]ProcessedInvoiceItem, float64, float64) {
	var processedItems []ProcessedInvoiceItem
	var subtotal, totalTax float64

	for _, item := range items {
		quantity := item.Quantity
		if quantity == 0 {
			quantity = 1
		}

		unitPrice := item.UnitPrice
		taxRate := item.TaxRate

		itemTotal := float64(quantity) * unitPrice
		taxAmount := itemTotal * taxRate
		totalWithTax := itemTotal + taxAmount

		processedItems = append(processedItems, ProcessedInvoiceItem{
			InvoiceItem:  item,
			ItemTotal:    itemTotal,
			TaxAmount:    taxAmount,
			TotalWithTax: totalWithTax,
		})

		subtotal += itemTotal
		totalTax += taxAmount
	}

	return processedItems, subtotal, totalTax
}

func (ip *InvoiceProcessor) createInvoice(ctx context.Context, invoice InvoiceData) error {
	_, _ = json.Marshal(invoice.Items)

	query := `
		INSERT INTO invoices (
			invoice_number, client_name, client_email, client_address,
			issue_date, due_date, subtotal, tax_amount, total_amount,
			currency, status, terms, notes, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := ip.db.ExecContext(ctx, query,
		invoice.InvoiceNumber, invoice.ClientName, invoice.ClientEmail, invoice.ClientAddress,
		invoice.IssueDate, invoice.DueDate, invoice.Subtotal, invoice.TotalTax, invoice.Total,
		invoice.Currency, invoice.Status, invoice.PaymentTerms, invoice.Notes, time.Now())

	if err != nil {
		return err
	}

	// Insert invoice items
	for _, item := range invoice.Items {
		_, err = ip.db.ExecContext(ctx,
			`INSERT INTO invoice_items (invoice_number, description, quantity, unit_price, tax_rate, total_amount)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			invoice.InvoiceNumber, item.Description, item.Quantity, item.UnitPrice, item.TaxRate,
			float64(item.Quantity)*item.UnitPrice)

		if err != nil {
			log.Printf("Failed to insert invoice item: %v", err)
		}
	}

	return nil
}

func (ip *InvoiceProcessor) updateInvoice(ctx context.Context, invoice InvoiceData) error {
	query := `
		UPDATE invoices SET
			client_name = $1, client_email = $2, client_address = $3,
			due_date = $4, subtotal = $5, tax_amount = $6, total_amount = $7,
			status = $8, terms = $9, notes = $10, updated_at = $11
		WHERE invoice_number = $12`

	_, err := ip.db.ExecContext(ctx, query,
		invoice.ClientName, invoice.ClientEmail, invoice.ClientAddress,
		invoice.DueDate, invoice.Subtotal, invoice.TotalTax, invoice.Total,
		invoice.Status, invoice.PaymentTerms, invoice.Notes, time.Now(), invoice.InvoiceNumber)

	return err
}

func (ip *InvoiceProcessor) getInvoice(ctx context.Context, invoiceNumber string) (*InvoiceData, error) {
	var invoice InvoiceData

	query := `
		SELECT invoice_number, client_name, client_email, client_address,
		       issue_date, due_date, subtotal, tax_amount, total_amount,
		       currency, status, terms, notes, created_at
		FROM invoices WHERE invoice_number = $1`

	err := ip.db.QueryRowContext(ctx, query, invoiceNumber).Scan(
		&invoice.InvoiceNumber, &invoice.ClientName, &invoice.ClientEmail, &invoice.ClientAddress,
		&invoice.IssueDate, &invoice.DueDate, &invoice.Subtotal, &invoice.TotalTax, &invoice.Total,
		&invoice.Currency, &invoice.Status, &invoice.PaymentTerms, &invoice.Notes, &invoice.CreatedAt)

	if err != nil {
		return nil, err
	}

	// Get invoice items
	itemsQuery := `
		SELECT description, quantity, unit_price, tax_rate, total_amount
		FROM invoice_items WHERE invoice_number = $1`

	rows, err := ip.db.QueryContext(ctx, itemsQuery, invoiceNumber)
	if err != nil {
		return &invoice, nil // Return invoice without items if items query fails
	}
	defer rows.Close()

	for rows.Next() {
		var item InvoiceItem
		var totalAmount float64

		err := rows.Scan(&item.Description, &item.Quantity, &item.UnitPrice, &item.TaxRate, &totalAmount)
		if err != nil {
			continue
		}

		invoice.Items = append(invoice.Items, item)
	}

	return &invoice, nil
}

func (ip *InvoiceProcessor) getTotalPaid(ctx context.Context, invoiceNumber string) float64 {
	var total float64

	err := ip.db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(amount), 0) FROM payments WHERE invoice_number = $1",
		invoiceNumber).Scan(&total)

	if err != nil {
		return 0
	}

	return total
}

func (ip *InvoiceProcessor) getPayments(ctx context.Context, invoiceNumber string) ([]PaymentData, error) {
	query := `
		SELECT id, amount, payment_date, payment_method, reference, notes
		FROM payments WHERE invoice_number = $1
		ORDER BY payment_date DESC`

	rows, err := ip.db.QueryContext(ctx, query, invoiceNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []PaymentData

	for rows.Next() {
		var p PaymentData

		err := rows.Scan(&p.ID, &p.Amount, &p.PaymentDate, &p.PaymentMethod, &p.Reference, &p.Notes)
		if err != nil {
			continue
		}

		payments = append(payments, p)
	}

	return payments, nil
}

func (ip *InvoiceProcessor) calculateNextInvoiceDate(currentDate, frequency string) string {
	date, err := time.Parse("2006-01-02", currentDate)
	if err != nil {
		return time.Now().AddDate(0, 1, 0).Format("2006-01-02") // Default to 1 month
	}

	switch frequency {
	case "weekly":
		return date.AddDate(0, 0, 7).Format("2006-01-02")
	case "monthly":
		return date.AddDate(0, 1, 0).Format("2006-01-02")
	case "quarterly":
		return date.AddDate(0, 3, 0).Format("2006-01-02")
	case "yearly":
		return date.AddDate(1, 0, 0).Format("2006-01-02")
	default:
		return date.AddDate(0, 1, 0).Format("2006-01-02") // Default monthly
	}
}