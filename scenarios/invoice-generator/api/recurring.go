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

// RecurringInvoice represents a recurring invoice template
type RecurringInvoice struct {
	ID             string        `json:"id"`
	CompanyID      string        `json:"company_id"`
	ClientID       string        `json:"client_id"`
	TemplateName   string        `json:"template_name"`
	Frequency      string        `json:"frequency"` // monthly, quarterly, yearly
	NextDate       string        `json:"next_date"`
	EndDate        string        `json:"end_date,omitempty"`
	IsActive       bool          `json:"is_active"`
	Currency       string        `json:"currency"`
	TaxRate        float64       `json:"tax_rate"`
	TaxName        string        `json:"tax_name"`
	Notes          string        `json:"notes"`
	Terms          string        `json:"terms"`
	Items          []InvoiceItem `json:"items"`
	DaysDue        int           `json:"days_due"` // Days after issue date
	AutoSend       bool          `json:"auto_send"`
	LastGenerated  string        `json:"last_generated,omitempty"`
	TotalGenerated int           `json:"total_generated"`
	CreatedAt      string        `json:"created_at"`
	UpdatedAt      string        `json:"updated_at"`
}

// CreateRecurringInvoiceHandler creates a new recurring invoice template
func createRecurringInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	var recurring RecurringInvoice
	if err := json.NewDecoder(r.Body).Decode(&recurring); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	recurring.ID = uuid.New().String()
	recurring.IsActive = true
	recurring.TotalGenerated = 0
	
	if recurring.Currency == "" {
		recurring.Currency = "USD"
	}
	if recurring.DaysDue == 0 {
		recurring.DaysDue = 30
	}
	
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	
	// Insert recurring invoice
	query := `
		INSERT INTO recurring_invoices (
			id, company_id, client_id, template_name, frequency, next_date, 
			end_date, is_active, currency, tax_rate, tax_name, notes, terms,
			days_due, auto_send
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING created_at, updated_at`
	
	err = tx.QueryRow(query,
		recurring.ID, recurring.CompanyID, recurring.ClientID, recurring.TemplateName,
		recurring.Frequency, recurring.NextDate, recurring.EndDate, recurring.IsActive,
		recurring.Currency, recurring.TaxRate, recurring.TaxName, recurring.Notes,
		recurring.Terms, recurring.DaysDue, recurring.AutoSend).
		Scan(&recurring.CreatedAt, &recurring.UpdatedAt)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Insert recurring invoice items
	for i, item := range recurring.Items {
		item.ID = uuid.New().String()
		_, err = tx.Exec(`
			INSERT INTO recurring_invoice_items (
				id, recurring_invoice_id, item_order, description, 
				quantity, unit_price, unit, tax_rate
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			item.ID, recurring.ID, i, item.Description,
			item.Quantity, item.UnitPrice, item.Unit, item.TaxRate)
		
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	
	if err = tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recurring)
}

// GetRecurringInvoicesHandler retrieves all recurring invoices
func getRecurringInvoicesHandler(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT r.id, r.template_name, r.frequency, r.next_date, r.is_active,
			   r.auto_send, r.total_generated, r.last_generated,
			   c.name as client_name
		FROM recurring_invoices r
		LEFT JOIN clients c ON r.client_id = c.id
		ORDER BY r.next_date ASC`
	
	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var invoices []map[string]interface{}
	for rows.Next() {
		invoice := make(map[string]interface{})
		var id, templateName, frequency, nextDate string
		var isActive, autoSend bool
		var totalGenerated int
		var lastGenerated, clientName sql.NullString
		
		err := rows.Scan(&id, &templateName, &frequency, &nextDate, &isActive,
			&autoSend, &totalGenerated, &lastGenerated, &clientName)
		if err != nil {
			continue
		}
		
		invoice["id"] = id
		invoice["template_name"] = templateName
		invoice["frequency"] = frequency
		invoice["next_date"] = nextDate
		invoice["is_active"] = isActive
		invoice["auto_send"] = autoSend
		invoice["total_generated"] = totalGenerated
		
		if lastGenerated.Valid {
			invoice["last_generated"] = lastGenerated.String
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
		SELECT id, company_id, client_id, template_name, frequency, next_date,
			   end_date, currency, tax_rate, tax_name, notes, terms,
			   days_due, auto_send
		FROM recurring_invoices
		WHERE is_active = true 
			AND next_date <= $1
			AND (end_date IS NULL OR end_date >= $1)`,
		today)
	
	if err != nil {
		log.Printf("Error fetching recurring invoices: %v", err)
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var recurring RecurringInvoice
		var endDate sql.NullString
		
		err := rows.Scan(&recurring.ID, &recurring.CompanyID, &recurring.ClientID,
			&recurring.TemplateName, &recurring.Frequency, &recurring.NextDate,
			&endDate, &recurring.Currency, &recurring.TaxRate, &recurring.TaxName,
			&recurring.Notes, &recurring.Terms, &recurring.DaysDue, &recurring.AutoSend)
		
		if err != nil {
			log.Printf("Error scanning recurring invoice: %v", err)
			continue
		}
		
		if endDate.Valid {
			recurring.EndDate = endDate.String
		}
		
		// Get items for this recurring invoice
		itemRows, err := db.Query(`
			SELECT description, quantity, unit_price, unit, tax_rate
			FROM recurring_invoice_items
			WHERE recurring_invoice_id = $1
			ORDER BY item_order`,
			recurring.ID)
		
		if err != nil {
			log.Printf("Error fetching recurring invoice items: %v", err)
			continue
		}
		
		var items []InvoiceItem
		var subtotal float64
		for itemRows.Next() {
			var item InvoiceItem
			err := itemRows.Scan(&item.Description, &item.Quantity,
				&item.UnitPrice, &item.Unit, &item.TaxRate)
			if err != nil {
				continue
			}
			item.LineTotal = item.Quantity * item.UnitPrice
			subtotal += item.LineTotal
			items = append(items, item)
		}
		itemRows.Close()
		
		// Generate invoice number
		invoiceNumber := generateInvoiceNumber(recurring.CompanyID)
		
		// Calculate dates
		issueDate := time.Now()
		dueDate := issueDate.AddDate(0, 0, recurring.DaysDue)
		
		// Calculate totals
		taxAmount := subtotal * (recurring.TaxRate / 100)
		totalAmount := subtotal + taxAmount
		
		// Create the invoice
		newInvoice := Invoice{
			ID:            uuid.New().String(),
			CompanyID:     recurring.CompanyID,
			ClientID:      recurring.ClientID,
			InvoiceNumber: invoiceNumber,
			Status:        "draft",
			IssueDate:     issueDate.Format("2006-01-02"),
			DueDate:       dueDate.Format("2006-01-02"),
			Currency:      recurring.Currency,
			Subtotal:      subtotal,
			TaxAmount:     taxAmount,
			TotalAmount:   totalAmount,
			BalanceDue:    totalAmount,
			TaxRate:       recurring.TaxRate,
			TaxName:       recurring.TaxName,
			Notes:         recurring.Notes,
			Terms:         recurring.Terms,
			Items:         items,
		}
		
		// Insert the generated invoice
		err = createGeneratedInvoice(newInvoice, recurring.ID)
		if err != nil {
			log.Printf("Error creating invoice from recurring template %s: %v", 
				recurring.TemplateName, err)
			continue
		}
		
		// Update recurring invoice
		nextDate := calculateNextDate(recurring.NextDate, recurring.Frequency)
		_, err = db.Exec(`
			UPDATE recurring_invoices
			SET next_date = $1,
				last_generated = CURRENT_DATE,
				total_generated = total_generated + 1,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = $2`,
			nextDate, recurring.ID)
		
		if err != nil {
			log.Printf("Error updating recurring invoice: %v", err)
		}
		
		// Auto-send if configured
		if recurring.AutoSend {
			go sendInvoiceEmail(newInvoice)
		}
		
		log.Printf("Generated invoice %s from recurring template %s", 
			invoiceNumber, recurring.TemplateName)
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
func calculateNextDate(currentDate, frequency string) string {
	date, err := time.Parse("2006-01-02", currentDate)
	if err != nil {
		return currentDate
	}
	
	switch frequency {
	case "weekly":
		date = date.AddDate(0, 0, 7)
	case "biweekly":
		date = date.AddDate(0, 0, 14)
	case "monthly":
		date = date.AddDate(0, 1, 0)
	case "quarterly":
		date = date.AddDate(0, 3, 0)
	case "yearly":
		date = date.AddDate(1, 0, 0)
	default:
		date = date.AddDate(0, 1, 0) // Default to monthly
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