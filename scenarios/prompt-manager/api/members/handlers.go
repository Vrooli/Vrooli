// Package avatars provides types and operations for avatar management.
package avatars

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Handlers provides HTTP handlers for avatar operations.
type Handlers struct {
	store *Store
}

// NewHandlers creates a new avatars handler.
func NewHandlers(store *Store) *Handlers {
	return &Handlers{store: store}
}

// List handles GET /avatars - returns all avatars.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	avatars, err := h.store.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responses := make([]Response, 0, len(avatars))
	for _, a := range avatars {
		responses = append(responses, toResponse(a))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// Get handles GET /avatars/{id} - returns a single avatar.
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	avatar, err := h.store.FindByID(id)
	if err != nil {
		http.Error(w, "Avatar not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toResponse(*avatar))
}

// Create handles POST /avatars - creates a new avatar.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Generate ID if not provided
	id := req.ID
	if id == "" {
		id = slugify(req.Name)
	}

	// Validate colors
	if !isValidHexColor(req.BodyColor) {
		http.Error(w, "Invalid bodyColor format (expected hex color)", http.StatusBadRequest)
		return
	}
	if !isValidHexColor(req.HeadColor) {
		http.Error(w, "Invalid headColor format (expected hex color)", http.StatusBadRequest)
		return
	}
	if !isValidHexColor(req.AccentColor) {
		http.Error(w, "Invalid accentColor format (expected hex color)", http.StatusBadRequest)
		return
	}

	now := time.Now().Format(time.RFC3339)
	skills := req.Skills
	if skills == nil {
		skills = []string{}
	}

	avatar := Avatar{
		ID:          id,
		Name:        req.Name,
		BodyColor:   req.BodyColor,
		HeadColor:   req.HeadColor,
		AccentColor: req.AccentColor,
		Skills:      skills,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.store.Create(avatar); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toResponse(avatar))
}

// Update handles PUT /avatars/{id} - updates an existing avatar.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	avatar, err := h.store.FindByID(id)
	if err != nil {
		http.Error(w, "Avatar not found", http.StatusNotFound)
		return
	}

	// Update fields
	if req.Name != nil {
		avatar.Name = *req.Name
	}
	if req.BodyColor != nil {
		if !isValidHexColor(*req.BodyColor) {
			http.Error(w, "Invalid bodyColor format (expected hex color)", http.StatusBadRequest)
			return
		}
		avatar.BodyColor = *req.BodyColor
	}
	if req.HeadColor != nil {
		if !isValidHexColor(*req.HeadColor) {
			http.Error(w, "Invalid headColor format (expected hex color)", http.StatusBadRequest)
			return
		}
		avatar.HeadColor = *req.HeadColor
	}
	if req.AccentColor != nil {
		if !isValidHexColor(*req.AccentColor) {
			http.Error(w, "Invalid accentColor format (expected hex color)", http.StatusBadRequest)
			return
		}
		avatar.AccentColor = *req.AccentColor
	}
	if req.Skills != nil {
		avatar.Skills = req.Skills
	}

	avatar.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := h.store.Update(*avatar); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toResponse(*avatar))
}

// Delete handles DELETE /avatars/{id} - deletes an avatar.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.store.Delete(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

func toResponse(a Avatar) Response {
	skills := a.Skills
	if skills == nil {
		skills = []string{}
	}
	return Response{
		ID:          a.ID,
		Name:        a.Name,
		BodyColor:   a.BodyColor,
		HeadColor:   a.HeadColor,
		AccentColor: a.AccentColor,
		Skills:      skills,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}

var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func isValidHexColor(color string) bool {
	return hexColorRegex.MatchString(color)
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	// Remove non-alphanumeric chars except hyphen
	var result []rune
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result = append(result, r)
		}
	}
	return string(result)
}
