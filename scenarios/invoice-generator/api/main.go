package main

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "math"
    "net/http"
    "os"
    "time"

    "github.com/google/uuid"
    "github.com/gorilla/mux"
    _ "github.com/lib/pq"
    "github.com/rs/cors"
)

var db *sql.DB
var invoiceProcessor *InvoiceProcessor

type Invoice struct {
    ID             string          `json:"id"`
    CompanyID      string          `json:"company_id,omitempty"`
    ClientID       string          `json:"client_id,omitempty"`
    InvoiceNumber  string          `json:"invoice_number"`
    Status         string          `json:"status"`
    IssueDate      string          `json:"issue_date"`
    DueDate        string          `json:"due_date"`
    Currency       string          `json:"currency"`
    Subtotal       float64         `json:"subtotal"`
    TaxAmount      float64         `json:"tax_amount"`
    DiscountAmount float64         `json:"discount_amount"`
    TotalAmount    float64         `json:"total_amount"`
    PaidAmount     float64         `json:"paid_amount"`
    BalanceDue     float64         `json:"balance_due"`
    TaxRate        float64         `json:"tax_rate,omitempty"`
    TaxName        string          `json:"tax_name,omitempty"`
    Notes          string          `json:"notes,omitempty"`
    Terms          string          `json:"terms,omitempty"`
    Items          []InvoiceItem   `json:"items,omitempty"`
    Client         *Client         `json:"client,omitempty"`
    CreatedAt      string          `json:"created_at"`
    UpdatedAt      string          `json:"updated_at"`
}

type InvoiceItem struct {
    ID          string  `json:"id,omitempty"`
    InvoiceID   string  `json:"invoice_id,omitempty"`
    ItemOrder   int     `json:"item_order"`
    Description string  `json:"description"`
    Quantity    float64 `json:"quantity"`
    UnitPrice   float64 `json:"unit_price"`
    Unit        string  `json:"unit,omitempty"`
    TaxRate     float64 `json:"tax_rate,omitempty"`
    LineTotal   float64 `json:"line_total"`
}

type Client struct {
    ID           string  `json:"id"`
    CompanyID    string  `json:"company_id,omitempty"`
    Name         string  `json:"name"`
    Email        string  `json:"email,omitempty"`
    Phone        string  `json:"phone,omitempty"`
    AddressLine1 string  `json:"address_line1,omitempty"`
    AddressLine2 string  `json:"address_line2,omitempty"`
    City         string  `json:"city,omitempty"`
    StateProvince string `json:"state_province,omitempty"`
    PostalCode   string  `json:"postal_code,omitempty"`
    Country      string  `json:"country,omitempty"`
    IsActive     bool    `json:"is_active"`
    CreatedAt    string  `json:"created_at"`
}

type Company struct {
    ID                  string  `json:"id"`
    Name                string  `json:"name"`
    LogoURL             string  `json:"logo_url,omitempty"`
    AddressLine1        string  `json:"address_line1,omitempty"`
    AddressLine2        string  `json:"address_line2,omitempty"`
    City                string  `json:"city,omitempty"`
    StateProvince       string  `json:"state_province,omitempty"`
    PostalCode          string  `json:"postal_code,omitempty"`
    Country             string  `json:"country,omitempty"`
    Phone               string  `json:"phone,omitempty"`
    Email               string  `json:"email,omitempty"`
    Website             string  `json:"website,omitempty"`
    TaxID               string  `json:"tax_id,omitempty"`
    DefaultPaymentTerms int     `json:"default_payment_terms"`
    DefaultCurrency     string  `json:"default_currency"`
    InvoicePrefix       string  `json:"invoice_prefix"`
    NextInvoiceNumber   int     `json:"next_invoice_number"`
}

func initDB() {
    // Database configuration - support both POSTGRES_URL and individual components
    postgresURL := os.Getenv("POSTGRES_URL")
    if postgresURL == "" {
        // Try to build from individual components
        dbHost := os.Getenv("POSTGRES_HOST")
        dbPort := os.Getenv("POSTGRES_PORT")
        dbUser := os.Getenv("POSTGRES_USER")
        dbPassword := os.Getenv("POSTGRES_PASSWORD")
        dbName := os.Getenv("POSTGRES_DB")
        
        if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
            log.Fatal("❌ Missing database configuration. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
        }
        
        postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
            dbUser, dbPassword, dbHost, dbPort, dbName)
    }
    
    var err error
    db, err = sql.Open("postgres", postgresURL)
    if err != nil {
        log.Fatalf("Failed to open database connection: %v", err)
    }
    
    // Set connection pool settings
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)
    
    // Implement exponential backoff for database connection
    maxRetries := 10
    baseDelay := 1 * time.Second
    maxDelay := 30 * time.Second
    
    log.Println("🔄 Attempting database connection with exponential backoff...")
    log.Printf("📆 Database URL configured")
    
    var pingErr error
    for attempt := 0; attempt < maxRetries; attempt++ {
        pingErr = db.Ping()
        if pingErr == nil {
            log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
            break
        }
        
        // Calculate exponential backoff delay
        delay := time.Duration(math.Min(
            float64(baseDelay) * math.Pow(2, float64(attempt)),
            float64(maxDelay),
        ))
        
        // Add progressive jitter to prevent thundering herd
        jitterRange := float64(delay) * 0.25
        jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
        actualDelay := delay + jitter
        
        log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
        log.Printf("⏳ Waiting %v before next attempt", actualDelay)
        
        // Provide detailed status every few attempts
        if attempt > 0 && attempt % 3 == 0 {
            log.Printf("📈 Retry progress:")
            log.Printf("   - Attempts made: %d/%d", attempt + 1, maxRetries)
            log.Printf("   - Total wait time: ~%v", time.Duration(attempt * 2) * baseDelay)
            log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
        }
        
        time.Sleep(actualDelay)
    }
    
    if pingErr != nil {
        log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
    }
    
    log.Println("🎉 Database connection pool established successfully!")
    
    // Initialize database tables
    if err = initializeDatabase(); err != nil {
        log.Printf("Warning: Database initialization had issues: %v", err)
    }
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status": "healthy",
        "timestamp": time.Now().Format(time.RFC3339),
    })
}

func createInvoiceHandler(w http.ResponseWriter, r *http.Request) {
    var invoice Invoice
    if err := json.NewDecoder(r.Body).Decode(&invoice); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Generate invoice ID
    invoice.ID = uuid.New().String()
    
    // Set defaults
    if invoice.Status == "" {
        invoice.Status = "draft"
    }
    if invoice.Currency == "" {
        invoice.Currency = "USD"
    }
    if invoice.IssueDate == "" {
        invoice.IssueDate = time.Now().Format("2006-01-02")
    }

    // Start transaction
    tx, err := db.Begin()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer tx.Rollback()

    // Insert invoice
    query := `
        INSERT INTO invoices (id, company_id, client_id, invoice_number, status, 
            issue_date, due_date, currency, tax_rate, tax_name, notes, terms,
            subtotal, tax_amount, discount_amount, total_amount)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
        RETURNING created_at, updated_at`
    
    err = tx.QueryRow(query, invoice.ID, invoice.CompanyID, invoice.ClientID,
        invoice.InvoiceNumber, invoice.Status, invoice.IssueDate, invoice.DueDate,
        invoice.Currency, invoice.TaxRate, invoice.TaxName, invoice.Notes, invoice.Terms,
        invoice.Subtotal, invoice.TaxAmount, invoice.DiscountAmount, invoice.TotalAmount).
        Scan(&invoice.CreatedAt, &invoice.UpdatedAt)
    
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Insert invoice items
    for i, item := range invoice.Items {
        item.ID = uuid.New().String()
        item.InvoiceID = invoice.ID
        item.ItemOrder = i
        item.LineTotal = item.Quantity * item.UnitPrice

        _, err = tx.Exec(`
            INSERT INTO invoice_items (id, invoice_id, item_order, description, 
                quantity, unit_price, unit, tax_rate, line_total)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
            item.ID, item.InvoiceID, item.ItemOrder, item.Description,
            item.Quantity, item.UnitPrice, item.Unit, item.TaxRate, item.LineTotal)
        
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
    json.NewEncoder(w).Encode(invoice)
}

func getInvoicesHandler(w http.ResponseWriter, r *http.Request) {
    query := `
        SELECT i.id, i.invoice_number, i.status, i.issue_date, i.due_date,
               i.currency, i.subtotal, i.tax_amount, i.discount_amount,
               i.total_amount, i.paid_amount, i.balance_due,
               i.created_at, i.updated_at,
               c.name as client_name, c.email as client_email
        FROM invoices i
        LEFT JOIN clients c ON i.client_id = c.id
        ORDER BY i.created_at DESC
        LIMIT 100`

    rows, err := db.Query(query)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var invoices []map[string]interface{}
    for rows.Next() {
        var invoice map[string]interface{} = make(map[string]interface{})
        var id, invoiceNumber, status, issueDate, dueDate, currency string
        var subtotal, taxAmount, discountAmount, totalAmount, paidAmount, balanceDue float64
        var createdAt, updatedAt string
        var clientName, clientEmail sql.NullString

        err := rows.Scan(&id, &invoiceNumber, &status, &issueDate, &dueDate,
            &currency, &subtotal, &taxAmount, &discountAmount,
            &totalAmount, &paidAmount, &balanceDue,
            &createdAt, &updatedAt, &clientName, &clientEmail)
        
        if err != nil {
            continue
        }

        invoice["id"] = id
        invoice["invoice_number"] = invoiceNumber
        invoice["status"] = status
        invoice["issue_date"] = issueDate
        invoice["due_date"] = dueDate
        invoice["currency"] = currency
        invoice["subtotal"] = subtotal
        invoice["tax_amount"] = taxAmount
        invoice["discount_amount"] = discountAmount
        invoice["total_amount"] = totalAmount
        invoice["paid_amount"] = paidAmount
        invoice["balance_due"] = balanceDue
        invoice["created_at"] = createdAt
        invoice["updated_at"] = updatedAt
        
        if clientName.Valid {
            invoice["client_name"] = clientName.String
        }
        if clientEmail.Valid {
            invoice["client_email"] = clientEmail.String
        }

        invoices = append(invoices, invoice)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(invoices)
}

func getInvoiceHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    invoiceID := vars["id"]

    var invoice Invoice
    query := `
        SELECT id, invoice_number, status, issue_date, due_date, currency,
               subtotal, tax_amount, discount_amount, total_amount, paid_amount,
               balance_due, tax_rate, tax_name, notes, terms, created_at, updated_at
        FROM invoices WHERE id = $1`
    
    err := db.QueryRow(query, invoiceID).Scan(
        &invoice.ID, &invoice.InvoiceNumber, &invoice.Status,
        &invoice.IssueDate, &invoice.DueDate, &invoice.Currency,
        &invoice.Subtotal, &invoice.TaxAmount, &invoice.DiscountAmount,
        &invoice.TotalAmount, &invoice.PaidAmount, &invoice.BalanceDue,
        &invoice.TaxRate, &invoice.TaxName, &invoice.Notes, &invoice.Terms,
        &invoice.CreatedAt, &invoice.UpdatedAt)
    
    if err == sql.ErrNoRows {
        http.Error(w, "Invoice not found", http.StatusNotFound)
        return
    } else if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Get invoice items
    itemsQuery := `
        SELECT id, item_order, description, quantity, unit_price, unit, tax_rate, line_total
        FROM invoice_items WHERE invoice_id = $1 ORDER BY item_order`
    
    rows, err := db.Query(itemsQuery, invoiceID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    for rows.Next() {
        var item InvoiceItem
        err := rows.Scan(&item.ID, &item.ItemOrder, &item.Description,
            &item.Quantity, &item.UnitPrice, &item.Unit, &item.TaxRate, &item.LineTotal)
        if err != nil {
            continue
        }
        invoice.Items = append(invoice.Items, item)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(invoice)
}

func updateInvoiceStatusHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    invoiceID := vars["id"]

    var req struct {
        Status string `json:"status"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    _, err := db.Exec(`
        UPDATE invoices 
        SET status = $1, updated_at = CURRENT_TIMESTAMP
        WHERE id = $2`,
        req.Status, invoiceID)
    
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func getClientsHandler(w http.ResponseWriter, r *http.Request) {
    query := `
        SELECT id, name, email, phone, address_line1, city, country, is_active, created_at
        FROM clients
        WHERE is_active = true
        ORDER BY name
        LIMIT 100`

    rows, err := db.Query(query)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var clients []Client
    for rows.Next() {
        var client Client
        err := rows.Scan(&client.ID, &client.Name, &client.Email, &client.Phone,
            &client.AddressLine1, &client.City, &client.Country,
            &client.IsActive, &client.CreatedAt)
        if err != nil {
            continue
        }
        clients = append(clients, client)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(clients)
}

func createClientHandler(w http.ResponseWriter, r *http.Request) {
    var client Client
    if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    client.ID = uuid.New().String()
    if client.IsActive == false {
        client.IsActive = true
    }

    query := `
        INSERT INTO clients (id, name, email, phone, address_line1, address_line2,
            city, state_province, postal_code, country, is_active)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        RETURNING created_at`
    
    err := db.QueryRow(query, client.ID, client.Name, client.Email, client.Phone,
        client.AddressLine1, client.AddressLine2, client.City, client.StateProvince,
        client.PostalCode, client.Country, client.IsActive).Scan(&client.CreatedAt)
    
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(client)
}

func processInvoiceHandler(w http.ResponseWriter, r *http.Request) {
    var req InvoiceProcessRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    ctx := r.Context()
    response, err := invoiceProcessor.ProcessInvoice(ctx, req)
    if err != nil {
        http.Error(w, "Failed to process invoice", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    if response.Success {
        w.WriteHeader(http.StatusCreated)
    } else {
        w.WriteHeader(http.StatusBadRequest)
    }
    json.NewEncoder(w).Encode(response)
}

func trackPaymentHandler(w http.ResponseWriter, r *http.Request) {
    var req PaymentRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    ctx := r.Context()
    response, err := invoiceProcessor.TrackPayment(ctx, req)
    if err != nil {
        http.Error(w, "Failed to track payment", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    if response.Success {
        w.WriteHeader(http.StatusCreated)
    } else {
        w.WriteHeader(http.StatusBadRequest)
    }
    json.NewEncoder(w).Encode(response)
}

func processRecurringInvoicesHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    response, err := invoiceProcessor.ProcessRecurringInvoices(ctx)
    if err != nil {
        http.Error(w, "Failed to process recurring invoices", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func main() {
    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
        fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start invoice-generator

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
        os.Exit(1)
    }
    initDB()
    defer db.Close()
    
    // Initialize InvoiceProcessor
    invoiceProcessor = NewInvoiceProcessor(db)
    
    // Start background processors
    go trackOverdueInvoices()
    go processRecurringInvoices()

    router := mux.NewRouter()

    // Health check
    router.HandleFunc("/health", healthHandler).Methods("GET")

    // Invoice endpoints
    router.HandleFunc("/api/invoices", createInvoiceHandler).Methods("POST")
    router.HandleFunc("/api/invoices", getInvoicesHandler).Methods("GET")
    router.HandleFunc("/api/invoices/{id}", getInvoiceHandler).Methods("GET")
    router.HandleFunc("/api/invoices/{id}/status", updateInvoiceStatusHandler).Methods("PUT")
    router.HandleFunc("/api/invoices/{id}/pdf", downloadPDFHandler).Methods("GET")
    router.HandleFunc("/api/invoices/generate-pdf", generatePDFHandler).Methods("POST")
    
    // Payment endpoints
    router.HandleFunc("/api/payments", recordPaymentHandler).Methods("POST")
    router.HandleFunc("/api/payments/invoice/{invoice_id}", getPaymentsHandler).Methods("GET")
    router.HandleFunc("/api/payments/summary", getPaymentSummaryHandler).Methods("GET")
    router.HandleFunc("/api/payments/{id}/refund", refundPaymentHandler).Methods("POST")
    
    // Recurring invoice endpoints
    router.HandleFunc("/api/recurring", createRecurringInvoiceHandler).Methods("POST")
    router.HandleFunc("/api/recurring", getRecurringInvoicesHandler).Methods("GET")
    router.HandleFunc("/api/recurring/{id}", updateRecurringInvoiceHandler).Methods("PUT")
    router.HandleFunc("/api/recurring/{id}", deleteRecurringInvoiceHandler).Methods("DELETE")

    // Client endpoints
    router.HandleFunc("/api/clients", getClientsHandler).Methods("GET")
    router.HandleFunc("/api/clients", createClientHandler).Methods("POST")

    // Invoice Processor endpoints  
    router.HandleFunc("/api/process/invoice", processInvoiceHandler).Methods("POST")
    router.HandleFunc("/api/process/payment", trackPaymentHandler).Methods("POST")
    router.HandleFunc("/api/process/recurring", processRecurringInvoicesHandler).Methods("GET", "POST")

    // Configure CORS
    c := cors.New(cors.Options{
        AllowedOrigins: []string{"*"},
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowedHeaders: []string{"*"},
    })

    handler := c.Handler(router)

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

    log.Printf("Invoice Generator API starting on port %s", port)
    if err := http.ListenAndServe(":"+port, handler); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}

