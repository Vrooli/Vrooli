package main

import (
	"database/sql"
)

// GetInvoiceByID retrieves an invoice by ID
func getInvoiceByID(id string) (*Invoice, error) {
	var invoice Invoice
	var taxRate, taxAmount, discountAmount sql.NullFloat64
	var taxName, notes, terms sql.NullString

	query := `
		SELECT id, company_id, client_id, invoice_number, status, issue_date, due_date,
			   currency, subtotal, tax_amount, discount_amount, total_amount,
			   paid_amount, balance_due, tax_rate, tax_name, notes, terms,
			   created_at, updated_at
		FROM invoices WHERE id = $1`

	err := db.QueryRow(query, id).Scan(
		&invoice.ID, &invoice.CompanyID, &invoice.ClientID, &invoice.InvoiceNumber,
		&invoice.Status, &invoice.IssueDate, &invoice.DueDate, &invoice.Currency,
		&invoice.Subtotal, &taxAmount, &discountAmount,
		&invoice.TotalAmount, &invoice.PaidAmount, &invoice.BalanceDue,
		&taxRate, &taxName, &notes, &terms,
		&invoice.CreatedAt, &invoice.UpdatedAt)

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if taxRate.Valid {
		invoice.TaxRate = taxRate.Float64
	}
	if taxAmount.Valid {
		invoice.TaxAmount = taxAmount.Float64
	}
	if discountAmount.Valid {
		invoice.DiscountAmount = discountAmount.Float64
	}
	if taxName.Valid {
		invoice.TaxName = taxName.String
	}
	if notes.Valid {
		invoice.Notes = notes.String
	}
	if terms.Valid {
		invoice.Terms = terms.String
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
		var unit sql.NullString
		var itemTaxRate sql.NullFloat64

		err := rows.Scan(&item.ID, &item.ItemOrder, &item.Description,
			&item.Quantity, &item.UnitPrice, &unit, &itemTaxRate, &item.LineTotal)
		if err != nil {
			continue
		}
		if unit.Valid {
			item.Unit = unit.String
		}
		if itemTaxRate.Valid {
			item.TaxRate = itemTaxRate.Float64
		}
		item.InvoiceID = id
		invoice.Items = append(invoice.Items, item)
	}

	return &invoice, nil
}

// GetClientByID retrieves a client by ID
func getClientByID(id string) (*Client, error) {
	var client Client
	var companyID, email, phone, addr1, addr2, city, state, postal, country sql.NullString

	query := `
		SELECT id, company_id, name, email, phone, address_line1, address_line2,
			   city, state_province, postal_code, country, is_active, created_at
		FROM clients WHERE id = $1`

	err := db.QueryRow(query, id).Scan(
		&client.ID, &companyID, &client.Name, &email, &phone,
		&addr1, &addr2, &city,
		&state, &postal, &country,
		&client.IsActive, &client.CreatedAt)

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if companyID.Valid {
		client.CompanyID = companyID.String
	}
	if email.Valid {
		client.Email = email.String
	}
	if phone.Valid {
		client.Phone = phone.String
	}
	if addr1.Valid {
		client.AddressLine1 = addr1.String
	}
	if addr2.Valid {
		client.AddressLine2 = addr2.String
	}
	if city.Valid {
		client.City = city.String
	}
	if state.Valid {
		client.StateProvince = state.String
	}
	if postal.Valid {
		client.PostalCode = postal.String
	}
	if country.Valid {
		client.Country = country.String
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

// NOTE: Database schema initialization removed from here.
// Schema is now managed by initialization/postgres/schema.sql
// and loaded by the lifecycle populate script.
// This eliminates the VARCHAR vs UUID type conflict that was breaking invoice creation.