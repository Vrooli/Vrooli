package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"knowledge-observatory/internal/docschema"
)

// TemplateListItem represents a template in the list response.
type TemplateListItem struct {
	DocType      string `json:"doc_type"`
	ExpectedPath string `json:"expected_path"`
	Purpose      string `json:"purpose"`
}

// TemplateDetailResponse represents a single template response.
type TemplateDetailResponse struct {
	DocType      string `json:"doc_type"`
	ExpectedPath string `json:"expected_path"`
	Purpose      string `json:"purpose"`
	Content      string `json:"content"`
}

func (s *Server) handleDocsTemplateList(w http.ResponseWriter, r *http.Request) {
	types := docschema.ListTemplateDocTypes()
	items := make([]TemplateListItem, 0, len(types))
	for _, dt := range types {
		items = append(items, TemplateListItem{
			DocType:      string(dt),
			ExpectedPath: dt.ExpectedPath(),
			Purpose:      docschema.TemplatePurpose(dt),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

func (s *Server) handleDocsTemplateGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	docTypeRaw := vars["doc_type"]

	dt, err := docschema.ParseDocType(docTypeRaw)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	content, err := docschema.TemplateForDocType(dt)
	if err != nil {
		s.respondError(w, http.StatusNotFound, err.Error())
		return
	}

	response := TemplateDetailResponse{
		DocType:      string(dt),
		ExpectedPath: dt.ExpectedPath(),
		Purpose:      docschema.TemplatePurpose(dt),
		Content:      content,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
