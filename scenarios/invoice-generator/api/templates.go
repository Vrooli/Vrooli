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

// InvoiceTemplate represents a customizable invoice template
type InvoiceTemplate struct {
	ID           string                 `json:"id"`
	CompanyID    string                 `json:"company_id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	TemplateData map[string]interface{} `json:"template_data"`
	IsDefault    bool                   `json:"is_default"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
}

// TemplateData structure for storing template configuration
type TemplateData struct {
	// Layout settings
	Layout struct {
		PageSize    string `json:"page_size"`    // A4, Letter, etc.
		Orientation string `json:"orientation"`  // portrait, landscape
		Margins     struct {
			Top    int `json:"top"`
			Right  int `json:"right"`
			Bottom int `json:"bottom"`
			Left   int `json:"left"`
		} `json:"margins"`
	} `json:"layout"`

	// Branding
	Branding struct {
		PrimaryColor   string `json:"primary_color"`
		SecondaryColor string `json:"secondary_color"`
		LogoURL        string `json:"logo_url,omitempty"`
		LogoPosition   string `json:"logo_position"` // left, center, right
		ShowLogo       bool   `json:"show_logo"`
	} `json:"branding"`

	// Typography
	Typography struct {
		FontFamily  string  `json:"font_family"`
		FontSize    int     `json:"font_size"`
		LineHeight  float64 `json:"line_height"`
		HeaderStyle string  `json:"header_style"` // bold, normal, uppercase
	} `json:"typography"`

	// Content sections
	Sections struct {
		ShowCompanyInfo  bool `json:"show_company_info"`
		ShowClientInfo   bool `json:"show_client_info"`
		ShowInvoiceNumber bool `json:"show_invoice_number"`
		ShowDates        bool `json:"show_dates"`
		ShowPaymentTerms bool `json:"show_payment_terms"`
		ShowNotes        bool `json:"show_notes"`
		ShowTax          bool `json:"show_tax"`
		ShowDiscount     bool `json:"show_discount"`
	} `json:"sections"`

	// Table styling
	Table struct {
		HeaderBgColor string `json:"header_bg_color"`
		HeaderColor   string `json:"header_color"`
		RowEvenBg     string `json:"row_even_bg"`
		RowOddBg      string `json:"row_odd_bg"`
		BorderColor   string `json:"border_color"`
		ShowBorders   bool   `json:"show_borders"`
	} `json:"table"`

	// Footer
	Footer struct {
		ShowFooter  bool   `json:"show_footer"`
		Text        string `json:"text,omitempty"`
		ShowPageNum bool   `json:"show_page_number"`
	} `json:"footer"`
}

// GetDefaultTemplateData returns a professional default template
func GetDefaultTemplateData() TemplateData {
	var td TemplateData

	// Layout
	td.Layout.PageSize = "A4"
	td.Layout.Orientation = "portrait"
	td.Layout.Margins.Top = 30
	td.Layout.Margins.Right = 30
	td.Layout.Margins.Bottom = 30
	td.Layout.Margins.Left = 30

	// Branding
	td.Branding.PrimaryColor = "#2563EB"
	td.Branding.SecondaryColor = "#64748B"
	td.Branding.LogoPosition = "left"
	td.Branding.ShowLogo = true

	// Typography
	td.Typography.FontFamily = "Arial, sans-serif"
	td.Typography.FontSize = 11
	td.Typography.LineHeight = 1.5
	td.Typography.HeaderStyle = "bold"

	// Sections
	td.Sections.ShowCompanyInfo = true
	td.Sections.ShowClientInfo = true
	td.Sections.ShowInvoiceNumber = true
	td.Sections.ShowDates = true
	td.Sections.ShowPaymentTerms = true
	td.Sections.ShowNotes = true
	td.Sections.ShowTax = true
	td.Sections.ShowDiscount = true

	// Table
	td.Table.HeaderBgColor = "#2563EB"
	td.Table.HeaderColor = "#FFFFFF"
	td.Table.RowEvenBg = "#F8FAFC"
	td.Table.RowOddBg = "#FFFFFF"
	td.Table.BorderColor = "#E2E8F0"
	td.Table.ShowBorders = true

	// Footer
	td.Footer.ShowFooter = true
	td.Footer.Text = "Thank you for your business!"
	td.Footer.ShowPageNum = true

	return td
}

// ListTemplatesHandler returns all invoice templates for a company
func ListTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	companyID := r.URL.Query().Get("company_id")
	if companyID == "" {
		// Get default company
		var defaultCompanyID string
		err := db.QueryRow("SELECT id FROM companies LIMIT 1").Scan(&defaultCompanyID)
		if err != nil {
			http.Error(w, "No company found", http.StatusNotFound)
			return
		}
		companyID = defaultCompanyID
	}

	query := `
		SELECT id, company_id, name, description, template_data, is_default, created_at, updated_at
		FROM invoice_templates
		WHERE company_id = $1
		ORDER BY is_default DESC, created_at DESC
	`

	rows, err := db.Query(query, companyID)
	if err != nil {
		log.Printf("Error fetching templates: %v", err)
		http.Error(w, "Failed to fetch templates", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	templates := []InvoiceTemplate{}
	for rows.Next() {
		var t InvoiceTemplate
		var templateDataJSON []byte
		var description sql.NullString

		err := rows.Scan(&t.ID, &t.CompanyID, &t.Name, &description, &templateDataJSON, &t.IsDefault, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning template: %v", err)
			continue
		}

		if description.Valid {
			t.Description = description.String
		}

		if err := json.Unmarshal(templateDataJSON, &t.TemplateData); err != nil {
			log.Printf("Error unmarshaling template data: %v", err)
			continue
		}

		templates = append(templates, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// GetTemplateHandler returns a specific template by ID
func GetTemplateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templateID := vars["id"]

	var t InvoiceTemplate
	var templateDataJSON []byte
	var description sql.NullString

	query := `
		SELECT id, company_id, name, description, template_data, is_default, created_at, updated_at
		FROM invoice_templates
		WHERE id = $1
	`

	err := db.QueryRow(query, templateID).Scan(&t.ID, &t.CompanyID, &t.Name, &description, &templateDataJSON, &t.IsDefault, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error fetching template: %v", err)
		http.Error(w, "Failed to fetch template", http.StatusInternalServerError)
		return
	}

	if description.Valid {
		t.Description = description.String
	}

	if err := json.Unmarshal(templateDataJSON, &t.TemplateData); err != nil {
		log.Printf("Error unmarshaling template data: %v", err)
		http.Error(w, "Invalid template data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

// CreateTemplateHandler creates a new invoice template
func CreateTemplateHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CompanyID    string                 `json:"company_id,omitempty"`
		Name         string                 `json:"name"`
		Description  string                 `json:"description,omitempty"`
		TemplateData map[string]interface{} `json:"template_data"`
		IsDefault    bool                   `json:"is_default"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Template name is required", http.StatusBadRequest)
		return
	}

	// Get default company if not specified
	companyID := req.CompanyID
	if companyID == "" {
		var defaultCompanyID string
		err := db.QueryRow("SELECT id FROM companies LIMIT 1").Scan(&defaultCompanyID)
		if err != nil {
			http.Error(w, "No company found", http.StatusNotFound)
			return
		}
		companyID = defaultCompanyID
	}

	// If no template data provided, use default
	templateData := req.TemplateData
	if templateData == nil || len(templateData) == 0 {
		defaultData := GetDefaultTemplateData()
		dataBytes, _ := json.Marshal(defaultData)
		json.Unmarshal(dataBytes, &templateData)
	}

	templateDataJSON, err := json.Marshal(templateData)
	if err != nil {
		http.Error(w, "Invalid template data", http.StatusBadRequest)
		return
	}

	// If this is marked as default, unset other defaults first
	if req.IsDefault {
		_, err := db.Exec("UPDATE invoice_templates SET is_default = false WHERE company_id = $1", companyID)
		if err != nil {
			log.Printf("Error unsetting default templates: %v", err)
		}
	}

	templateID := uuid.New().String()
	query := `
		INSERT INTO invoice_templates (id, company_id, name, description, template_data, is_default, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	var createdAt, updatedAt time.Time
	err = db.QueryRow(query, templateID, companyID, req.Name, req.Description, templateDataJSON, req.IsDefault, now, now).Scan(&templateID, &createdAt, &updatedAt)
	if err != nil {
		log.Printf("Error creating template: %v", err)
		http.Error(w, "Failed to create template", http.StatusInternalServerError)
		return
	}

	response := InvoiceTemplate{
		ID:           templateID,
		CompanyID:    companyID,
		Name:         req.Name,
		Description:  req.Description,
		TemplateData: templateData,
		IsDefault:    req.IsDefault,
		CreatedAt:    createdAt.Format(time.RFC3339),
		UpdatedAt:    updatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// UpdateTemplateHandler updates an existing template
func UpdateTemplateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templateID := vars["id"]

	var req struct {
		Name         string                 `json:"name,omitempty"`
		Description  string                 `json:"description,omitempty"`
		TemplateData map[string]interface{} `json:"template_data,omitempty"`
		IsDefault    *bool                  `json:"is_default,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if template exists
	var companyID string
	err := db.QueryRow("SELECT company_id FROM invoice_templates WHERE id = $1", templateID).Scan(&companyID)
	if err == sql.ErrNoRows {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error checking template: %v", err)
		http.Error(w, "Failed to update template", http.StatusInternalServerError)
		return
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argCount := 1

	if req.Name != "" {
		updates = append(updates, fmt.Sprintf("name = $%d", argCount))
		args = append(args, req.Name)
		argCount++
	}

	if req.Description != "" {
		updates = append(updates, fmt.Sprintf("description = $%d", argCount))
		args = append(args, req.Description)
		argCount++
	}

	if req.TemplateData != nil {
		templateDataJSON, err := json.Marshal(req.TemplateData)
		if err != nil {
			http.Error(w, "Invalid template data", http.StatusBadRequest)
			return
		}
		updates = append(updates, fmt.Sprintf("template_data = $%d", argCount))
		args = append(args, templateDataJSON)
		argCount++
	}

	if req.IsDefault != nil {
		// If setting as default, unset others first
		if *req.IsDefault {
			_, err := db.Exec("UPDATE invoice_templates SET is_default = false WHERE company_id = $1", companyID)
			if err != nil {
				log.Printf("Error unsetting default templates: %v", err)
			}
		}
		updates = append(updates, fmt.Sprintf("is_default = $%d", argCount))
		args = append(args, *req.IsDefault)
		argCount++
	}

	if len(updates) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	// Add updated_at
	updates = append(updates, fmt.Sprintf("updated_at = $%d", argCount))
	args = append(args, time.Now())
	argCount++

	// Add template ID for WHERE clause
	args = append(args, templateID)

	query := "UPDATE invoice_templates SET " + updates[0]
	for i := 1; i < len(updates); i++ {
		query += ", " + updates[i]
	}
	query += fmt.Sprintf(" WHERE id = $%d", argCount)

	_, err = db.Exec(query, args...)
	if err != nil {
		log.Printf("Error updating template: %v", err)
		http.Error(w, "Failed to update template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"template_id": templateID,
		"message":     "Template updated successfully",
	})
}

// DeleteTemplateHandler deletes a template
func DeleteTemplateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	templateID := vars["id"]

	// Check if it's the default template
	var isDefault bool
	err := db.QueryRow("SELECT is_default FROM invoice_templates WHERE id = $1", templateID).Scan(&isDefault)
	if err == sql.ErrNoRows {
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error checking template: %v", err)
		http.Error(w, "Failed to delete template", http.StatusInternalServerError)
		return
	}

	if isDefault {
		http.Error(w, "Cannot delete default template. Set another template as default first.", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("DELETE FROM invoice_templates WHERE id = $1", templateID)
	if err != nil {
		log.Printf("Error deleting template: %v", err)
		http.Error(w, "Failed to delete template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Template deleted successfully",
	})
}

// GetDefaultTemplateHandler returns the default template for a company
func GetDefaultTemplateHandler(w http.ResponseWriter, r *http.Request) {
	companyID := r.URL.Query().Get("company_id")
	if companyID == "" {
		// Get default company
		var defaultCompanyID string
		err := db.QueryRow("SELECT id FROM companies LIMIT 1").Scan(&defaultCompanyID)
		if err != nil {
			http.Error(w, "No company found", http.StatusNotFound)
			return
		}
		companyID = defaultCompanyID
	}

	var t InvoiceTemplate
	var templateDataJSON []byte
	var description sql.NullString

	query := `
		SELECT id, company_id, name, description, template_data, is_default, created_at, updated_at
		FROM invoice_templates
		WHERE company_id = $1 AND is_default = true
		LIMIT 1
	`

	err := db.QueryRow(query, companyID).Scan(&t.ID, &t.CompanyID, &t.Name, &description, &templateDataJSON, &t.IsDefault, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		// No default template, create one
		defaultData := GetDefaultTemplateData()
		dataBytes, _ := json.Marshal(defaultData)

		templateID := uuid.New().String()
		now := time.Now()

		insertQuery := `
			INSERT INTO invoice_templates (id, company_id, name, description, template_data, is_default, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id, created_at, updated_at
		`

		var createdAt, updatedAt time.Time
		err = db.QueryRow(insertQuery, templateID, companyID, "Default Professional Template", "Standard invoice template with professional styling", dataBytes, true, now, now).Scan(&templateID, &createdAt, &updatedAt)
		if err != nil {
			log.Printf("Error creating default template: %v", err)
			http.Error(w, "Failed to create default template", http.StatusInternalServerError)
			return
		}

		json.Unmarshal(dataBytes, &t.TemplateData)
		t.ID = templateID
		t.CompanyID = companyID
		t.Name = "Default Professional Template"
		t.Description = "Standard invoice template with professional styling"
		t.IsDefault = true
		t.CreatedAt = createdAt.Format(time.RFC3339)
		t.UpdatedAt = updatedAt.Format(time.RFC3339)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(t)
		return
	}

	if err != nil {
		log.Printf("Error fetching default template: %v", err)
		http.Error(w, "Failed to fetch default template", http.StatusInternalServerError)
		return
	}

	if description.Valid {
		t.Description = description.String
	}

	if err := json.Unmarshal(templateDataJSON, &t.TemplateData); err != nil {
		log.Printf("Error unmarshaling template data: %v", err)
		http.Error(w, "Invalid template data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}
