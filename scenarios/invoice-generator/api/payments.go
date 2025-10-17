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

// Payment represents a payment record
type Payment struct {
	ID            string  `json:"id"`
	InvoiceID     string  `json:"invoice_id"`
	Amount        float64 `json:"amount"`
	PaymentDate   string  `json:"payment_date"`
	PaymentMethod string  `json:"payment_method"`
	Reference     string  `json:"reference"`
	Notes         string  `json:"notes"`
	CreatedAt     string  `json:"created_at"`
}

// PaymentSummary represents payment analytics
type PaymentSummary struct {
	TotalPaid      float64 `json:"total_paid"`
	TotalPending   float64 `json:"total_pending"`
	TotalOverdue   float64 `json:"total_overdue"`
	RecentPayments []Payment `json:"recent_payments"`
}

// RecordPaymentHandler records a payment for an invoice
func recordPaymentHandler(w http.ResponseWriter, r *http.Request) {
	var payment Payment
	if err := json.NewDecoder(r.Body).Decode(&payment); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	payment.ID = uuid.New().String()
	if payment.PaymentDate == "" {
		payment.PaymentDate = time.Now().Format("2006-01-02")
	}
	
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	
	// Insert payment record
	query := `
		INSERT INTO payments (id, invoice_id, amount, payment_date, payment_method, reference_number, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at`
	
	err = tx.QueryRow(query, payment.ID, payment.InvoiceID, payment.Amount,
		payment.PaymentDate, payment.PaymentMethod, payment.Reference, payment.Notes).
		Scan(&payment.CreatedAt)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Invoice amounts and status are automatically updated by trigger_payment_status trigger
	// which runs after payment INSERT/UPDATE/DELETE

	if err = tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Send payment confirmation
	go sendPaymentNotification(payment)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payment)
}

// GetPaymentsHandler retrieves payments for an invoice
func getPaymentsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	invoiceID := vars["invoice_id"]
	
	query := `
		SELECT id, invoice_id, amount, payment_date, payment_method, reference_number, notes, created_at
		FROM payments
		WHERE invoice_id = $1
		ORDER BY payment_date DESC`
	
	rows, err := db.Query(query, invoiceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var payments []Payment
	for rows.Next() {
		var payment Payment
		err := rows.Scan(&payment.ID, &payment.InvoiceID, &payment.Amount,
			&payment.PaymentDate, &payment.PaymentMethod, &payment.Reference,
			&payment.Notes, &payment.CreatedAt)
		if err != nil {
			continue
		}
		payments = append(payments, payment)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payments)
}

// GetPaymentSummaryHandler returns payment analytics
func getPaymentSummaryHandler(w http.ResponseWriter, r *http.Request) {
	summary := PaymentSummary{}
	
	// Get total paid
	var totalPaid sql.NullFloat64
	db.QueryRow(`
		SELECT SUM(paid_amount) FROM invoices WHERE status IN ('paid', 'partially_paid')
	`).Scan(&totalPaid)
	if totalPaid.Valid {
		summary.TotalPaid = totalPaid.Float64
	}
	
	// Get total pending
	var totalPending sql.NullFloat64
	db.QueryRow(`
		SELECT SUM(balance_due) FROM invoices 
		WHERE status IN ('sent', 'partially_paid') AND due_date >= CURRENT_DATE
	`).Scan(&totalPending)
	if totalPending.Valid {
		summary.TotalPending = totalPending.Float64
	}
	
	// Get total overdue
	var totalOverdue sql.NullFloat64
	db.QueryRow(`
		SELECT SUM(balance_due) FROM invoices 
		WHERE status IN ('sent', 'partially_paid', 'overdue') AND due_date < CURRENT_DATE
	`).Scan(&totalOverdue)
	if totalOverdue.Valid {
		summary.TotalOverdue = totalOverdue.Float64
	}
	
	// Get recent payments
	rows, err := db.Query(`
		SELECT p.id, p.invoice_id, p.amount, p.payment_date, p.payment_method,
			   p.reference_number, p.notes, p.created_at
		FROM payments p
		ORDER BY p.payment_date DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var payment Payment
			err := rows.Scan(&payment.ID, &payment.InvoiceID, &payment.Amount,
				&payment.PaymentDate, &payment.PaymentMethod, &payment.Reference,
				&payment.Notes, &payment.CreatedAt)
			if err == nil {
				summary.RecentPayments = append(summary.RecentPayments, payment)
			}
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// TrackOverdueInvoices checks for overdue invoices and upcoming due dates
func trackOverdueInvoices() {
	// Run immediately on startup, then every 24 hours
	processReminders()

	ticker := time.NewTicker(24 * time.Hour) // Check daily
	defer ticker.Stop()

	for range ticker.C {
		processReminders()
	}
}

func processReminders() {
	// 1. Update overdue invoices
	_, err := db.Exec(`
		UPDATE invoices
		SET status = 'overdue',
			updated_at = CURRENT_TIMESTAMP
		WHERE status IN ('sent', 'partially_paid')
			AND due_date < CURRENT_DATE
			AND balance_due > 0
	`)

	if err != nil {
		log.Printf("Error updating overdue invoices: %v", err)
	}

	// 2. Send overdue reminders (weekly for overdue invoices)
	rows, err := db.Query(`
		SELECT id, invoice_number, client_id, balance_due, due_date
		FROM invoices
		WHERE status = 'overdue'
			AND balance_due > 0
			AND (last_reminder_sent IS NULL
				 OR last_reminder_sent < CURRENT_DATE - INTERVAL '7 days')
	`)

	if err != nil {
		log.Printf("Error fetching overdue invoices: %v", err)
	} else {
		defer rows.Close()

		overdueCount := 0
		for rows.Next() {
			var invoiceID, invoiceNumber, clientID, dueDate string
			var balanceDue float64

			err := rows.Scan(&invoiceID, &invoiceNumber, &clientID, &balanceDue, &dueDate)
			if err != nil {
				continue
			}

			// Send overdue reminder
			go sendOverdueReminder(invoiceID, invoiceNumber, clientID, balanceDue, dueDate)

			// Update last reminder sent
			db.Exec(`
				UPDATE invoices
				SET last_reminder_sent = CURRENT_DATE
				WHERE id = $1
			`, invoiceID)

			overdueCount++
		}

		if overdueCount > 0 {
			log.Printf("Sent %d overdue reminders", overdueCount)
		}
	}

	// 3. Send upcoming due date reminders (7 days before due date, once only)
	upcomingRows, err := db.Query(`
		SELECT id, invoice_number, client_id, balance_due, due_date
		FROM invoices
		WHERE status IN ('sent', 'partially_paid')
			AND balance_due > 0
			AND due_date BETWEEN CURRENT_DATE + INTERVAL '6 days' AND CURRENT_DATE + INTERVAL '8 days'
			AND last_reminder_sent IS NULL
	`)

	if err != nil {
		log.Printf("Error fetching upcoming invoices: %v", err)
	} else {
		defer upcomingRows.Close()

		upcomingCount := 0
		for upcomingRows.Next() {
			var invoiceID, invoiceNumber, clientID, dueDate string
			var balanceDue float64

			err := upcomingRows.Scan(&invoiceID, &invoiceNumber, &clientID, &balanceDue, &dueDate)
			if err != nil {
				continue
			}

			// Send upcoming reminder
			go sendUpcomingReminder(invoiceID, invoiceNumber, clientID, balanceDue, dueDate)

			// Update last reminder sent
			db.Exec(`
				UPDATE invoices
				SET last_reminder_sent = CURRENT_DATE
				WHERE id = $1
			`, invoiceID)

			upcomingCount++
		}

		if upcomingCount > 0 {
			log.Printf("Sent %d upcoming payment reminders", upcomingCount)
		}
	}
}

// RefundPaymentHandler processes refunds
func refundPaymentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	paymentID := vars["id"]
	
	var refundReq struct {
		Amount float64 `json:"amount"`
		Reason string  `json:"reason"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&refundReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	
	// Get original payment
	var originalPayment Payment
	err = tx.QueryRow(`
		SELECT id, invoice_id, amount FROM payments WHERE id = $1
	`, paymentID).Scan(&originalPayment.ID, &originalPayment.InvoiceID, &originalPayment.Amount)
	
	if err != nil {
		http.Error(w, "Payment not found", http.StatusNotFound)
		return
	}
	
	// Validate refund amount
	if refundReq.Amount > originalPayment.Amount {
		http.Error(w, "Refund amount exceeds original payment", http.StatusBadRequest)
		return
	}
	
	// Create refund record
	refundID := uuid.New().String()
	_, err = tx.Exec(`
		INSERT INTO refunds (id, payment_id, amount, reason, refund_date)
		VALUES ($1, $2, $3, $4, CURRENT_DATE)
	`, refundID, paymentID, refundReq.Amount, refundReq.Reason)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Update invoice balance
	_, err = tx.Exec(`
		UPDATE invoices 
		SET paid_amount = paid_amount - $1,
			balance_due = balance_due + $1,
			status = CASE 
				WHEN balance_due + $1 >= total_amount THEN 'sent'
				ELSE 'partially_paid'
			END,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, refundReq.Amount, originalPayment.InvoiceID)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	if err = tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"refund_id": refundID,
		"amount": refundReq.Amount,
		"reason": refundReq.Reason,
	})
}

// ReminderType represents the type of reminder being sent
type ReminderType string

const (
	ReminderTypeOverdue        ReminderType = "overdue"
	ReminderTypeUpcoming       ReminderType = "upcoming"
	ReminderTypePaymentReceived ReminderType = "payment_received"
)

// ReminderRecord represents a payment reminder
type ReminderRecord struct {
	ID           string       `json:"id"`
	InvoiceID    string       `json:"invoice_id"`
	ClientID     string       `json:"client_id"`
	ReminderType ReminderType `json:"reminder_type"`
	SentAt       string       `json:"sent_at"`
	Status       string       `json:"status"` // sent, failed, pending
	Message      string       `json:"message"`
}

// Helper notification functions
func sendPaymentNotification(payment Payment) {
	// Get invoice details
	var invoiceID, clientID string
	err := db.QueryRow(`
		SELECT id, client_id FROM invoices WHERE id = $1
	`, payment.InvoiceID).Scan(&invoiceID, &clientID)

	if err != nil {
		log.Printf("Error fetching invoice for payment notification: %v", err)
		return
	}

	// Create reminder record
	reminderID := uuid.New().String()
	message := "Payment received: $" + formatCurrency(payment.Amount)

	_, err = db.Exec(`
		INSERT INTO payment_reminders (id, invoice_id, client_id, reminder_type, status, message)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, reminderID, invoiceID, clientID, ReminderTypePaymentReceived, "sent", message)

	if err != nil {
		log.Printf("Error creating payment notification record: %v", err)
		return
	}

	log.Printf("Payment notification created for invoice %s: $%.2f", payment.InvoiceID, payment.Amount)
}

func sendOverdueReminder(invoiceID, invoiceNumber, clientID string, balanceDue float64, dueDate string) {
	// Create reminder record
	reminderID := uuid.New().String()
	message := "Invoice " + invoiceNumber + " is overdue. Amount due: $" + formatCurrency(balanceDue) + ". Due date was: " + dueDate

	_, err := db.Exec(`
		INSERT INTO payment_reminders (id, invoice_id, client_id, reminder_type, status, message)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, reminderID, invoiceID, clientID, ReminderTypeOverdue, "sent", message)

	if err != nil {
		log.Printf("Error creating overdue reminder record: %v", err)
		return
	}

	// Update invoice reminder tracking
	_, err = db.Exec(`
		UPDATE invoices
		SET reminder_count = reminder_count + 1,
			last_reminder_date = CURRENT_TIMESTAMP
		WHERE id = $1
	`, invoiceID)

	if err != nil {
		log.Printf("Error updating invoice reminder count: %v", err)
	}

	log.Printf("Overdue reminder created for invoice %s: $%.2f overdue since %s",
		invoiceNumber, balanceDue, dueDate)
}

func sendUpcomingReminder(invoiceID, invoiceNumber, clientID string, balanceDue float64, dueDate string) {
	// Create reminder record for invoices due soon (7 days warning)
	reminderID := uuid.New().String()
	message := "Invoice " + invoiceNumber + " is due soon. Amount: $" + formatCurrency(balanceDue) + ". Due date: " + dueDate

	_, err := db.Exec(`
		INSERT INTO payment_reminders (id, invoice_id, client_id, reminder_type, status, message)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, reminderID, invoiceID, clientID, ReminderTypeUpcoming, "sent", message)

	if err != nil {
		log.Printf("Error creating upcoming reminder record: %v", err)
		return
	}

	log.Printf("Upcoming payment reminder created for invoice %s: $%.2f due on %s",
		invoiceNumber, balanceDue, dueDate)
}

// formatCurrency formats a float as currency string
func formatCurrency(amount float64) string {
	return fmt.Sprintf("%.2f", amount)
}

// GetRemindersHandler retrieves all payment reminders with optional filtering
func getRemindersHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	reminderType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "50"
	}

	// Build query
	query := `
		SELECT r.id, r.invoice_id, r.client_id, r.reminder_type, r.status,
			   r.message, r.sent_at
		FROM payment_reminders r
		WHERE 1=1`

	args := []interface{}{}
	argCount := 1

	if reminderType != "" {
		query += fmt.Sprintf(" AND r.reminder_type = $%d", argCount)
		args = append(args, reminderType)
		argCount++
	}

	if status != "" {
		query += fmt.Sprintf(" AND r.status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	query += fmt.Sprintf(" ORDER BY r.sent_at DESC LIMIT $%d", argCount)
	args = append(args, limit)

	// Execute query
	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var reminders []ReminderRecord
	for rows.Next() {
		var reminder ReminderRecord
		err := rows.Scan(&reminder.ID, &reminder.InvoiceID, &reminder.ClientID,
			&reminder.ReminderType, &reminder.Status, &reminder.Message, &reminder.SentAt)
		if err != nil {
			continue
		}
		reminders = append(reminders, reminder)
	}

	if reminders == nil {
		reminders = []ReminderRecord{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reminders)
}

// GetInvoiceRemindersHandler retrieves reminders for a specific invoice
func getInvoiceRemindersHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	invoiceID := vars["invoice_id"]

	query := `
		SELECT r.id, r.invoice_id, r.client_id, r.reminder_type, r.status,
			   r.message, r.sent_at
		FROM payment_reminders r
		WHERE r.invoice_id = $1
		ORDER BY r.sent_at DESC`

	rows, err := db.Query(query, invoiceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var reminders []ReminderRecord
	for rows.Next() {
		var reminder ReminderRecord
		err := rows.Scan(&reminder.ID, &reminder.InvoiceID, &reminder.ClientID,
			&reminder.ReminderType, &reminder.Status, &reminder.Message, &reminder.SentAt)
		if err != nil {
			continue
		}
		reminders = append(reminders, reminder)
	}

	if reminders == nil {
		reminders = []ReminderRecord{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reminders)
}