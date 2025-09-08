package main

import (
	"database/sql"
	"log"
)

// GetInvoiceByID retrieves an invoice by ID
func getInvoiceByID(id string) (*Invoice, error) {
	var invoice Invoice
	query := `
		SELECT id, company_id, client_id, invoice_number, status, issue_date, due_date,
			   currency, subtotal, tax_amount, discount_amount, total_amount, 
			   paid_amount, balance_due, tax_rate, tax_name, notes, terms,
			   created_at, updated_at
		FROM invoices WHERE id = $1`
	
	err := db.QueryRow(query, id).Scan(
		&invoice.ID, &invoice.CompanyID, &invoice.ClientID, &invoice.InvoiceNumber,
		&invoice.Status, &invoice.IssueDate, &invoice.DueDate, &invoice.Currency,
		&invoice.Subtotal, &invoice.TaxAmount, &invoice.DiscountAmount,
		&invoice.TotalAmount, &invoice.PaidAmount, &invoice.BalanceDue,
		&invoice.TaxRate, &invoice.TaxName, &invoice.Notes, &invoice.Terms,
		&invoice.CreatedAt, &invoice.UpdatedAt)
	
	if err != nil {
		return nil, err
	}
	
	// Get invoice items
	itemsQuery := `
		SELECT id, item_order, description, quantity, unit_price, unit, tax_rate, line_total
		FROM invoice_items WHERE invoice_id = $1 ORDER BY item_order`
	
	rows, err := db.Query(itemsQuery, id)
	if err != nil {
		return &invoice, nil
	}
	defer rows.Close()
	
	for rows.Next() {
		var item InvoiceItem
		err := rows.Scan(&item.ID, &item.ItemOrder, &item.Description,
			&item.Quantity, &item.UnitPrice, &item.Unit, &item.TaxRate, &item.LineTotal)
		if err != nil {
			continue
		}
		item.InvoiceID = id
		invoice.Items = append(invoice.Items, item)
	}
	
	return &invoice, nil
}

// GetClientByID retrieves a client by ID
func getClientByID(id string) (*Client, error) {
	var client Client
	query := `
		SELECT id, company_id, name, email, phone, address_line1, address_line2,
			   city, state_province, postal_code, country, is_active, created_at
		FROM clients WHERE id = $1`
	
	err := db.QueryRow(query, id).Scan(
		&client.ID, &client.CompanyID, &client.Name, &client.Email, &client.Phone,
		&client.AddressLine1, &client.AddressLine2, &client.City,
		&client.StateProvince, &client.PostalCode, &client.Country,
		&client.IsActive, &client.CreatedAt)
	
	if err != nil {
		return nil, err
	}
	
	return &client, nil
}

// GetDefaultCompany retrieves the default company
func getDefaultCompany() (*Company, error) {
	var company Company
	query := `
		SELECT id, name, logo_url, address_line1, address_line2, city,
			   state_province, postal_code, country, phone, email, website,
			   tax_id, default_payment_terms, default_currency, 
			   invoice_prefix, next_invoice_number
		FROM companies
		LIMIT 1`
	
	err := db.QueryRow(query).Scan(
		&company.ID, &company.Name, &company.LogoURL,
		&company.AddressLine1, &company.AddressLine2, &company.City,
		&company.StateProvince, &company.PostalCode, &company.Country,
		&company.Phone, &company.Email, &company.Website,
		&company.TaxID, &company.DefaultPaymentTerms, &company.DefaultCurrency,
		&company.InvoicePrefix, &company.NextInvoiceNumber)
	
	if err != nil {
		// Return a default company if none exists
		return &Company{
			ID:                  "default",
			Name:                "Your Company",
			DefaultCurrency:     "USD",
			DefaultPaymentTerms: 30,
			InvoicePrefix:       "INV",
			NextInvoiceNumber:   1,
		}, nil
	}
	
	return &company, nil
}

// InitializeDatabase ensures all required tables exist
func initializeDatabase() error {
	// Create companies table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS companies (
			id VARCHAR(36) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			logo_url TEXT,
			address_line1 VARCHAR(255),
			address_line2 VARCHAR(255),
			city VARCHAR(100),
			state_province VARCHAR(100),
			postal_code VARCHAR(20),
			country VARCHAR(100),
			phone VARCHAR(50),
			email VARCHAR(255),
			website VARCHAR(255),
			tax_id VARCHAR(50),
			default_payment_terms INTEGER DEFAULT 30,
			default_currency VARCHAR(3) DEFAULT 'USD',
			invoice_prefix VARCHAR(10) DEFAULT 'INV',
			next_invoice_number INTEGER DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}
	
	// Create clients table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS clients (
			id VARCHAR(36) PRIMARY KEY,
			company_id VARCHAR(36),
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255),
			phone VARCHAR(50),
			address_line1 VARCHAR(255),
			address_line2 VARCHAR(255),
			city VARCHAR(100),
			state_province VARCHAR(100),
			postal_code VARCHAR(20),
			country VARCHAR(100),
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}
	
	// Create invoices table with additional columns
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS invoices (
			id VARCHAR(36) PRIMARY KEY,
			company_id VARCHAR(36),
			client_id VARCHAR(36),
			invoice_number VARCHAR(50) UNIQUE NOT NULL,
			status VARCHAR(20) DEFAULT 'draft',
			issue_date DATE,
			due_date DATE,
			currency VARCHAR(3) DEFAULT 'USD',
			tax_rate DECIMAL(5,2) DEFAULT 0,
			tax_name VARCHAR(50),
			notes TEXT,
			terms TEXT,
			subtotal DECIMAL(10,2) DEFAULT 0,
			tax_amount DECIMAL(10,2) DEFAULT 0,
			discount_amount DECIMAL(10,2) DEFAULT 0,
			total_amount DECIMAL(10,2) DEFAULT 0,
			paid_amount DECIMAL(10,2) DEFAULT 0,
			balance_due DECIMAL(10,2) DEFAULT 0,
			recurring_invoice_id VARCHAR(36),
			last_reminder_sent DATE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}
	
	// Create invoice_items table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS invoice_items (
			id VARCHAR(36) PRIMARY KEY,
			invoice_id VARCHAR(36) NOT NULL,
			item_order INTEGER,
			description TEXT,
			quantity DECIMAL(10,2),
			unit_price DECIMAL(10,2),
			unit VARCHAR(50),
			tax_rate DECIMAL(5,2) DEFAULT 0,
			line_total DECIMAL(10,2),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}
	
	// Create payments table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS payments (
			id VARCHAR(36) PRIMARY KEY,
			invoice_id VARCHAR(36) NOT NULL,
			amount DECIMAL(10,2) NOT NULL,
			payment_date DATE,
			payment_method VARCHAR(50),
			reference VARCHAR(100),
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}
	
	// Create refunds table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS refunds (
			id VARCHAR(36) PRIMARY KEY,
			payment_id VARCHAR(36) NOT NULL,
			amount DECIMAL(10,2) NOT NULL,
			reason TEXT,
			refund_date DATE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}
	
	// Create recurring_invoices table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS recurring_invoices (
			id VARCHAR(36) PRIMARY KEY,
			company_id VARCHAR(36),
			client_id VARCHAR(36),
			template_name VARCHAR(255),
			frequency VARCHAR(20),
			next_date DATE,
			end_date DATE,
			is_active BOOLEAN DEFAULT true,
			currency VARCHAR(3) DEFAULT 'USD',
			tax_rate DECIMAL(5,2) DEFAULT 0,
			tax_name VARCHAR(50),
			notes TEXT,
			terms TEXT,
			days_due INTEGER DEFAULT 30,
			auto_send BOOLEAN DEFAULT false,
			last_generated DATE,
			total_generated INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}
	
	// Create recurring_invoice_items table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS recurring_invoice_items (
			id VARCHAR(36) PRIMARY KEY,
			recurring_invoice_id VARCHAR(36) NOT NULL,
			item_order INTEGER,
			description TEXT,
			quantity DECIMAL(10,2),
			unit_price DECIMAL(10,2),
			unit VARCHAR(50),
			tax_rate DECIMAL(5,2) DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`)
	if err != nil {
		return err
	}
	
	// Create default company if none exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM companies").Scan(&count)
	if count == 0 {
		_, err = db.Exec(`
			INSERT INTO companies (id, name, default_currency, invoice_prefix)
			VALUES ('default', 'Your Company', 'USD', 'INV')`)
		if err != nil {
			log.Printf("Warning: Could not create default company: %v", err)
		}
	}
	
	log.Println("Database initialization completed")
	return nil
}