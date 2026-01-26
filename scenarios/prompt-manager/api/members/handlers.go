// Package members provides types and operations for member management.
package members

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Handlers provides HTTP handlers for member operations.
type Handlers struct {
	store MemberStore
}

// NewHandlers creates a new members handler.
func NewHandlers(store MemberStore) *Handlers {
	return &Handlers{store: store}
}

// List handles GET /members - returns all members.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	members, err := h.store.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responses := make([]Response, 0, len(members))
	for _, m := range members {
		responses = append(responses, toResponse(m))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// Get handles GET /members/{id} - returns a single member.
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	member, err := h.store.FindByID(id)
	if err != nil {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toResponse(*member))
}

// Create handles POST /members - creates a new member.
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

	member := Member{
		ID:          id,
		Name:        req.Name,
		BodyColor:   req.BodyColor,
		HeadColor:   req.HeadColor,
		AccentColor: req.AccentColor,
		Skills:      skills,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.store.Create(member); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toResponse(member))
}

// Update handles PUT /members/{id} - updates an existing member.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	member, err := h.store.FindByID(id)
	if err != nil {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}

	// Update fields
	if req.Name != nil {
		member.Name = *req.Name
	}
	if req.BodyColor != nil {
		if !isValidHexColor(*req.BodyColor) {
			http.Error(w, "Invalid bodyColor format (expected hex color)", http.StatusBadRequest)
			return
		}
		member.BodyColor = *req.BodyColor
	}
	if req.HeadColor != nil {
		if !isValidHexColor(*req.HeadColor) {
			http.Error(w, "Invalid headColor format (expected hex color)", http.StatusBadRequest)
			return
		}
		member.HeadColor = *req.HeadColor
	}
	if req.AccentColor != nil {
		if !isValidHexColor(*req.AccentColor) {
			http.Error(w, "Invalid accentColor format (expected hex color)", http.StatusBadRequest)
			return
		}
		member.AccentColor = *req.AccentColor
	}
	if req.Skills != nil {
		member.Skills = req.Skills
	}

	member.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := h.store.Update(*member); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toResponse(*member))
}

// Delete handles DELETE /members/{id} - deletes a member.
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

func toResponse(m Member) Response {
	skills := m.Skills
	if skills == nil {
		skills = []string{}
	}
	return Response{
		ID:          m.ID,
		Name:        m.Name,
		BodyColor:   m.BodyColor,
		HeadColor:   m.HeadColor,
		AccentColor: m.AccentColor,
		Skills:      skills,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
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
