package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// RecurringInvoiceTemplate represents a recurring invoice template
type RecurringInvoiceTemplate struct {
	ID                string        `json:"id"`
	CompanyID         string        `json:"company_id"`
	ClientID          string        `json:"client_id"`
	TemplateID        string        `json:"template_id,omitempty"`
	Frequency         string        `json:"frequency"` // daily, weekly, monthly, quarterly, yearly
	IntervalCount     int           `json:"interval_count"` // Every X periods
	StartDate         string        `json:"start_date"`
	EndDate           string        `json:"end_date,omitempty"`
	NextInvoiceDate   string        `json:"next_invoice_date"`
	IsActive          bool          `json:"is_active"`
	PaymentTerms      int           `json:"payment_terms,omitempty"`
	TaxRate           float64       `json:"tax_rate"`
	DiscountRate      float64       `json:"discount_rate,omitempty"`
	DiscountType      string        `json:"discount_type,omitempty"`
	Notes             string        `json:"notes"`
	Terms             string        `json:"terms"`
	Items             []InvoiceItem `json:"items"`
	LastGeneratedDate string        `json:"last_generated_date,omitempty"`
	TotalGenerated    int           `json:"total_generated"`
	CreatedAt         string        `json:"created_at"`
	UpdatedAt         string        `json:"updated_at"`
}

// CreateRecurringInvoiceHandler creates a new recurring invoice template
func createRecurringInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	var recurring RecurringInvoiceTemplate
	if err := json.NewDecoder(r.Body).Decode(&recurring); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	recurring.ID = uuid.New().String()
	recurring.IsActive = true
	recurring.TotalGenerated = 0

	if recurring.IntervalCount == 0 {
		recurring.IntervalCount = 1
	}
	if recurring.PaymentTerms == 0 {
		recurring.PaymentTerms = 30
	}

	// Convert items to JSONB
	itemsJSON, err := json.Marshal(recurring.Items)
	if err != nil {
		http.Error(w, "Failed to marshal items: " + err.Error(), http.StatusInternalServerError)
		return
	}

	// Insert recurring invoice
	query := `
		INSERT INTO recurring_invoices (
			id, company_id, client_id, template_id, frequency, interval_count,
			start_date, end_date, next_invoice_date, is_active, payment_terms,
			tax_rate, discount_rate, discount_type, notes, terms, items
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING created_at, updated_at`

	err = db.QueryRow(query,
		recurring.ID, recurring.CompanyID, recurring.ClientID, recurring.TemplateID,
		recurring.Frequency, recurring.IntervalCount, recurring.StartDate,
		recurring.EndDate, recurring.NextInvoiceDate, recurring.IsActive,
		recurring.PaymentTerms, recurring.TaxRate, recurring.DiscountRate,
		recurring.DiscountType, recurring.Notes, recurring.Terms, itemsJSON).
		Scan(&recurring.CreatedAt, &recurring.UpdatedAt)

	if err != nil {
		http.Error(w, "Failed to create recurring invoice: " + err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recurring)
}

// GetRecurringInvoicesHandler retrieves all recurring invoices
func getRecurringInvoicesHandler(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT r.id, r.template_id, r.frequency, r.next_invoice_date, r.is_active,
			   r.interval_count, r.last_generated_date,
			   c.name as client_name
		FROM recurring_invoices r
		LEFT JOIN clients c ON r.client_id = c.id
		ORDER BY r.next_invoice_date ASC`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var invoices []map[string]interface{}
	for rows.Next() {
		invoice := make(map[string]interface{})
		var id, frequency string
		var templateID sql.NullString
		var nextInvoiceDate sql.NullString
		var isActive bool
		var intervalCount int
		var lastGeneratedDate, clientName sql.NullString

		err := rows.Scan(&id, &templateID, &frequency, &nextInvoiceDate, &isActive,
			&intervalCount, &lastGeneratedDate, &clientName)
		if err != nil {
			continue
		}

		invoice["id"] = id
		if templateID.Valid {
			invoice["template_id"] = templateID.String
		}
		invoice["frequency"] = frequency
		if nextInvoiceDate.Valid {
			invoice["next_invoice_date"] = nextInvoiceDate.String
		}
		invoice["is_active"] = isActive
		invoice["interval_count"] = intervalCount

		if lastGeneratedDate.Valid {
			invoice["last_generated_date"] = lastGeneratedDate.String
		}
		if clientName.Valid {
			invoice["client_name"] = clientName.String
		}
		
		invoices = append(invoices, invoice)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invoices)
}

// ProcessRecurringInvoices generates invoices from recurring templates
func processRecurringInvoices() {
	ticker := time.NewTicker(24 * time.Hour) // Check daily
	defer ticker.Stop()
	
	// Process immediately on startup
	generateDueInvoices()
	
	for range ticker.C {
		generateDueInvoices()
	}
}

// GenerateDueInvoices creates invoices from recurring templates that are due
func generateDueInvoices() {
	today := time.Now().Format("2006-01-02")

	// Get all active recurring invoices due today or earlier
	rows, err := db.Query(`
		SELECT id, company_id, client_id, template_id, frequency, interval_count,
			   next_invoice_date, end_date, payment_terms, tax_rate,
			   discount_rate, discount_type, notes, terms, items
		FROM recurring_invoices
		WHERE is_active = true
			AND next_invoice_date <= $1
			AND (end_date IS NULL OR end_date >= $1)`,
		today)
	
	if err != nil {
		log.Printf("Error fetching recurring invoices: %v", err)
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var recurring RecurringInvoiceTemplate
		var endDate, nextDate sql.NullString
		var templateID, discountType sql.NullString
		var discountRate sql.NullFloat64
		var itemsJSON []byte

		err := rows.Scan(&recurring.ID, &recurring.CompanyID, &recurring.ClientID,
			&templateID, &recurring.Frequency, &recurring.IntervalCount,
			&nextDate, &endDate, &recurring.PaymentTerms, &recurring.TaxRate,
			&discountRate, &discountType, &recurring.Notes, &recurring.Terms, &itemsJSON)

		if err != nil {
			log.Printf("Error scanning recurring invoice: %v", err)
			continue
		}

		if endDate.Valid {
			recurring.EndDate = endDate.String
		}
		if nextDate.Valid {
			recurring.NextInvoiceDate = nextDate.String
		}
		if templateID.Valid {
			recurring.TemplateID = templateID.String
		}
		if discountRate.Valid {
			recurring.DiscountRate = discountRate.Float64
		}
		if discountType.Valid {
			recurring.DiscountType = discountType.String
		}

		// Parse items from JSONB
		var items []InvoiceItem
		if err := json.Unmarshal(itemsJSON, &items); err != nil {
			log.Printf("Error parsing recurring invoice items: %v", err)
			continue
		}

		// Calculate subtotal
		var subtotal float64
		for i := range items {
			items[i].LineTotal = items[i].Quantity * items[i].UnitPrice
			subtotal += items[i].LineTotal
		}
		
		// Generate invoice number
		invoiceNumber := generateInvoiceNumber(recurring.CompanyID)
		
		// Calculate dates
		issueDate := time.Now()
		dueDate := issueDate.AddDate(0, 0, recurring.PaymentTerms)

		// Calculate totals (apply discount if present)
		discountAmount := 0.0
		if recurring.DiscountRate > 0 {
			if recurring.DiscountType == "percentage" {
				discountAmount = subtotal * (recurring.DiscountRate / 100)
			} else {
				discountAmount = recurring.DiscountRate
			}
		}
		subtotalAfterDiscount := subtotal - discountAmount
		taxAmount := subtotalAfterDiscount * (recurring.TaxRate / 100)
		totalAmount := subtotalAfterDiscount + taxAmount

		// Create the invoice
		newInvoice := Invoice{
			ID:             uuid.New().String(),
			CompanyID:      recurring.CompanyID,
			ClientID:       recurring.ClientID,
			InvoiceNumber:  invoiceNumber,
			Status:         "draft",
			IssueDate:      issueDate.Format("2006-01-02"),
			DueDate:        dueDate.Format("2006-01-02"),
			Currency:       "USD", // Use default from company
			Subtotal:       subtotal,
			TaxAmount:      taxAmount,
			DiscountAmount: discountAmount,
			TotalAmount:    totalAmount,
			BalanceDue:     totalAmount,
			TaxRate:        recurring.TaxRate,
			Notes:          recurring.Notes,
			Terms:          recurring.Terms,
			Items:          items,
		}

		// Insert the generated invoice
		err = createGeneratedInvoice(newInvoice, recurring.ID)
		if err != nil {
			log.Printf("Error creating invoice from recurring template %s: %v",
				recurring.ID, err)
			continue
		}

		// Update recurring invoice
		nextInvoiceDate := calculateNextDate(recurring.NextInvoiceDate, recurring.Frequency, recurring.IntervalCount)
		_, err2 := db.Exec(`
			UPDATE recurring_invoices
			SET next_invoice_date = $1,
				last_generated_date = CURRENT_DATE,
				total_generated = total_generated + 1,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $2`,
			nextInvoiceDate, recurring.ID)

		if err2 != nil {
			log.Printf("Error updating recurring invoice: %v", err2)
		}

		log.Printf("Generated invoice %s from recurring template %s",
			invoiceNumber, recurring.ID)
	}
}

// UpdateRecurringInvoiceHandler updates a recurring invoice template
func updateRecurringInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	recurringID := vars["id"]
	
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Build update query dynamically
	query := "UPDATE recurring_invoices SET updated_at = CURRENT_TIMESTAMP"
	params := []interface{}{}
	paramCount := 1
	
	if val, ok := updates["is_active"]; ok {
		query += fmt.Sprintf(", is_active = $%d", paramCount)
		params = append(params, val)
		paramCount++
	}
	
	if val, ok := updates["frequency"]; ok {
		query += fmt.Sprintf(", frequency = $%d", paramCount)
		params = append(params, val)
		paramCount++
	}
	
	if val, ok := updates["next_date"]; ok {
		query += fmt.Sprintf(", next_date = $%d", paramCount)
		params = append(params, val)
		paramCount++
	}
	
	if val, ok := updates["auto_send"]; ok {
		query += fmt.Sprintf(", auto_send = $%d", paramCount)
		params = append(params, val)
		paramCount++
	}
	
	query += fmt.Sprintf(" WHERE id = $%d", paramCount)
	params = append(params, recurringID)
	
	_, err := db.Exec(query, params...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": "Recurring invoice updated",
	})
}

// DeleteRecurringInvoiceHandler deactivates a recurring invoice
func deleteRecurringInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	recurringID := vars["id"]
	
	_, err := db.Exec(`
		UPDATE recurring_invoices 
		SET is_active = false, 
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`,
		recurringID)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// Helper functions
func calculateNextDate(currentDate, frequency string, intervalCount int) string {
	date, err := time.Parse("2006-01-02", currentDate)
	if err != nil {
		return currentDate
	}

	if intervalCount == 0 {
		intervalCount = 1
	}

	switch frequency {
	case "daily":
		date = date.AddDate(0, 0, intervalCount)
	case "weekly":
		date = date.AddDate(0, 0, 7*intervalCount)
	case "monthly":
		date = date.AddDate(0, intervalCount, 0)
	case "quarterly":
		date = date.AddDate(0, 3*intervalCount, 0)
	case "yearly":
		date = date.AddDate(intervalCount, 0, 0)
	default:
		date = date.AddDate(0, intervalCount, 0) // Default to monthly
	}

	return date.Format("2006-01-02")
}

func createGeneratedInvoice(invoice Invoice, recurringID string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	// Insert invoice
	query := `
		INSERT INTO invoices (
			id, company_id, client_id, invoice_number, status, 
			issue_date, due_date, currency, tax_rate, tax_name, 
			notes, terms, subtotal, tax_amount, discount_amount, 
			total_amount, balance_due, recurring_invoice_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`
	
	_, err = tx.Exec(query,
		invoice.ID, invoice.CompanyID, invoice.ClientID, invoice.InvoiceNumber,
		invoice.Status, invoice.IssueDate, invoice.DueDate, invoice.Currency,
		invoice.TaxRate, invoice.TaxName, invoice.Notes, invoice.Terms,
		invoice.Subtotal, invoice.TaxAmount, invoice.DiscountAmount,
		invoice.TotalAmount, invoice.BalanceDue, recurringID)
	
	if err != nil {
		return err
	}
	
	// Insert invoice items
	for i, item := range invoice.Items {
		item.ID = uuid.New().String()
		_, err = tx.Exec(`
			INSERT INTO invoice_items (
				id, invoice_id, item_order, description, 
				quantity, unit_price, unit, tax_rate, line_total
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			item.ID, invoice.ID, i, item.Description,
			item.Quantity, item.UnitPrice, item.Unit, item.TaxRate, item.LineTotal)
		
		if err != nil {
			return err
		}
	}
	
	return tx.Commit()
}

func generateInvoiceNumber(companyID string) string {
	var prefix string
	var nextNumber int
	
	// Get company invoice settings
	err := db.QueryRow(`
		SELECT invoice_prefix, next_invoice_number 
		FROM companies WHERE id = $1`,
		companyID).Scan(&prefix, &nextNumber)
	
	if err != nil {
		// Default if company not found
		prefix = "INV"
		nextNumber = int(time.Now().Unix() % 100000)
	}
	
	// Update next invoice number
	db.Exec(`
		UPDATE companies 
		SET next_invoice_number = next_invoice_number + 1 
		WHERE id = $1`,
		companyID)
	
	return fmt.Sprintf("%s-%05d", prefix, nextNumber)
}

func sendInvoiceEmail(invoice Invoice) {
	// Implementation for sending invoice via email
	log.Printf("Invoice %s sent to client %s", invoice.InvoiceNumber, invoice.ClientID)
}