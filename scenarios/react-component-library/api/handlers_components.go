package main

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

// handleGetComponents retrieves all components with optional filtering
func (s *Server) handleGetComponents(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	category := r.URL.Query().Get("category")

	query := `
		SELECT id, library_id, display_name, description, version, file_path,
		       source_path, tags, category, tech_stack, created_at, updated_at
		FROM components
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	if search != "" {
		query += ` AND (display_name ILIKE $` + string(rune(argCount)) + ` OR description ILIKE $` + string(rune(argCount)) + `)`
		args = append(args, "%"+search+"%")
		argCount++
	}

	if category != "" {
		query += ` AND category = $` + string(rune(argCount))
		args = append(args, category)
	}

	query += ` ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to query components", err)
		return
	}
	defer rows.Close()

	components := []Component{}
	for rows.Next() {
		var c Component
		c.Tags = []string{}
		c.TechStack = []string{}
		err := rows.Scan(
			&c.ID, &c.LibraryID, &c.DisplayName, &c.Description, &c.Version,
			&c.FilePath, &c.SourcePath, pq.Array(&c.Tags), &c.Category,
			pq.Array(&c.TechStack), &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, "Failed to scan component", err)
			return
		}
		components = append(components, c)
	}

	s.respondJSON(w, http.StatusOK, components)
}

// handleGetComponent retrieves a single component by ID
func (s *Server) handleGetComponent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var c Component
	c.Tags = []string{}
	c.TechStack = []string{}
	err := s.db.QueryRowContext(r.Context(), `
		SELECT id, library_id, display_name, description, version, file_path,
		       source_path, tags, category, tech_stack, created_at, updated_at
		FROM components
		WHERE id = $1
	`, id).Scan(
		&c.ID, &c.LibraryID, &c.DisplayName, &c.Description, &c.Version,
		&c.FilePath, &c.SourcePath, pq.Array(&c.Tags), &c.Category,
		pq.Array(&c.TechStack), &c.CreatedAt, &c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		s.respondError(w, http.StatusNotFound, "Component not found", nil)
		return
	}
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to query component", err)
		return
	}

	s.respondJSON(w, http.StatusOK, c)
}

// handleGetComponentContent retrieves the source code content of a component
func (s *Server) handleGetComponentContent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_ = vars["id"] // TODO: Use this to load actual file content

	// For now, return a placeholder. In production, this would read from filesystem
	content := ComponentContent{
		Content: "// Component source code would be loaded from file system\n// Path stored in database: file_path field",
	}

	s.respondJSON(w, http.StatusOK, content)
}

// handleCreateComponent creates a new component
func (s *Server) handleCreateComponent(w http.ResponseWriter, r *http.Request) {
	var input Component
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Set default library ID if not provided
	if input.LibraryID == "" {
		input.LibraryID = "00000000-0000-0000-0000-000000000001"
	}

	// Ensure arrays are not nil
	if input.Tags == nil {
		input.Tags = []string{}
	}
	if input.TechStack == nil {
		input.TechStack = []string{}
	}

	var c Component
	// Initialize arrays to avoid nil issues
	c.Tags = []string{}
	c.TechStack = []string{}

	err := s.db.QueryRowContext(r.Context(), `
		INSERT INTO components (library_id, display_name, description, version, file_path,
		                       source_path, tags, category, tech_stack)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, library_id, display_name, description, version, file_path,
		          source_path, tags, category, tech_stack, created_at, updated_at
	`,
		input.LibraryID, input.DisplayName, input.Description, input.Version,
		input.FilePath, input.SourcePath, pq.Array(input.Tags), input.Category,
		pq.Array(input.TechStack),
	).Scan(
		&c.ID, &c.LibraryID, &c.DisplayName, &c.Description, &c.Version,
		&c.FilePath, &c.SourcePath, pq.Array(&c.Tags), &c.Category,
		pq.Array(&c.TechStack), &c.CreatedAt, &c.UpdatedAt,
	)

	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to create component", err)
		return
	}

	s.respondJSON(w, http.StatusCreated, c)
}

// handleUpdateComponent updates an existing component
func (s *Server) handleUpdateComponent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var input Component
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	var c Component
	c.Tags = []string{}
	c.TechStack = []string{}
	err := s.db.QueryRowContext(r.Context(), `
		UPDATE components
		SET display_name = COALESCE(NULLIF($2, ''), display_name),
		    description = COALESCE(NULLIF($3, ''), description),
		    version = COALESCE(NULLIF($4, ''), version),
		    file_path = COALESCE(NULLIF($5, ''), file_path),
		    source_path = COALESCE(NULLIF($6, ''), source_path),
		    tags = COALESCE($7, tags),
		    category = COALESCE($8, category),
		    tech_stack = COALESCE($9, tech_stack),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, library_id, display_name, description, version, file_path,
		          source_path, tags, category, tech_stack, created_at, updated_at
	`,
		id, input.DisplayName, input.Description, input.Version,
		input.FilePath, input.SourcePath, pq.Array(input.Tags), input.Category,
		pq.Array(input.TechStack),
	).Scan(
		&c.ID, &c.LibraryID, &c.DisplayName, &c.Description, &c.Version,
		&c.FilePath, &c.SourcePath, pq.Array(&c.Tags), &c.Category,
		pq.Array(&c.TechStack), &c.CreatedAt, &c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		s.respondError(w, http.StatusNotFound, "Component not found", nil)
		return
	}
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to update component", err)
		return
	}

	s.respondJSON(w, http.StatusOK, c)
}

// handleUpdateComponentContent updates the source code content of a component
func (s *Server) handleUpdateComponentContent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_ = vars["id"] // TODO: Use this to write file content and create version

	var input ComponentContent
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// For now, just return success. In production, this would write to filesystem
	// and create a new version entry
	w.WriteHeader(http.StatusNoContent)
}

// handleSearchComponents performs a search across components
func (s *Server) handleSearchComponents(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		s.respondJSON(w, http.StatusOK, []Component{})
		return
	}

	rows, err := s.db.QueryContext(r.Context(), `
		SELECT id, library_id, display_name, description, version, file_path,
		       source_path, tags, category, tech_stack, created_at, updated_at
		FROM components
		WHERE display_name ILIKE $1
		   OR description ILIKE $1
		   OR $2 = ANY(tags)
		ORDER BY created_at DESC
		LIMIT 50
	`, "%"+query+"%", query)

	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to search components", err)
		return
	}
	defer rows.Close()

	components := []Component{}
	for rows.Next() {
		var c Component
		c.Tags = []string{}
		c.TechStack = []string{}
		err := rows.Scan(
			&c.ID, &c.LibraryID, &c.DisplayName, &c.Description, &c.Version,
			&c.FilePath, &c.SourcePath, pq.Array(&c.Tags), &c.Category,
			pq.Array(&c.TechStack), &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, "Failed to scan component", err)
			return
		}
		components = append(components, c)
	}

	s.respondJSON(w, http.StatusOK, components)
}
