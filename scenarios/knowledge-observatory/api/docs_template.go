package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"

	"knowledge-observatory/internal/doccontract"
	"knowledge-observatory/internal/doctemplates"
	"knowledge-observatory/internal/services/dochealth"
)

// TemplateListItem represents a template in the list response.
type TemplateListItem struct {
	DocType      string `json:"doc_type"`
	ExpectedPath string `json:"expected_path"`
	Purpose      string `json:"purpose"`
	Title        string `json:"title"`
}

// TemplateDetailResponse represents a single template response.
type TemplateDetailResponse struct {
	DocType      string `json:"doc_type"`
	ExpectedPath string `json:"expected_path"`
	Purpose      string `json:"purpose"`
	Content      string `json:"content"`
}

func (s *Server) handleDocsTemplateList(w http.ResponseWriter, r *http.Request) {
	resolver := s.docTemplateResolver()
	templateID, err := s.templateIDForTemplateRequest(r)
	if err != nil {
		s.respondError(w, http.StatusNotFound, err.Error())
		return
	}
	contract, findings, err := resolver.ResolveTemplate(templateID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if err := contractError(findings); err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	items := make([]TemplateListItem, 0, len(contract.Documents))
	for _, doc := range contract.Documents {
		items = append(items, TemplateListItem{
			DocType:      doc.DocType,
			ExpectedPath: doc.ScenarioPath,
			Purpose:      doc.Description,
			Title:        doc.Title,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

func (s *Server) handleDocsTemplateGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	docTypeRaw := vars["doc_type"]

	resolver := s.docTemplateResolver()
	templateID, err := s.templateIDForTemplateRequest(r)
	if err != nil {
		s.respondError(w, http.StatusNotFound, err.Error())
		return
	}
	doc, content, err := resolver.TemplateContent(templateID, docTypeRaw)
	if err != nil {
		s.respondError(w, http.StatusNotFound, err.Error())
		return
	}

	response := TemplateDetailResponse{
		DocType:      doc.DocType,
		ExpectedPath: doc.ScenarioPath,
		Purpose:      doc.Description,
		Content:      content,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) docTemplateResolver() doctemplates.Resolver {
	if s != nil && s.config != nil && strings.TrimSpace(s.config.ScenariosRoot) != "" {
		return doctemplates.NewResolverFromScenariosRoot(s.config.ScenariosRoot)
	}
	return doctemplates.Resolver{}
}

func (s *Server) templateIDForTemplateRequest(r *http.Request) (string, error) {
	templateID := strings.TrimSpace(r.URL.Query().Get("template"))
	if templateID != "" {
		return templateID, nil
	}
	scenario := strings.TrimSpace(r.URL.Query().Get("scenario"))
	if scenario == "" {
		return "", nil
	}
	if strings.Contains(scenario, "/") || strings.Contains(scenario, "\\") || strings.Contains(scenario, "..") {
		return "", dochealth.ErrScenarioNameInvalid
	}
	resolver := s.docTemplateResolver()
	scenariosRoot := ""
	if s != nil && s.config != nil {
		scenariosRoot = strings.TrimSpace(s.config.ScenariosRoot)
	}
	resolved, err := resolver.ResolveScenario(filepath.Join(scenariosRoot, scenario))
	if err != nil {
		return "", err
	}
	return resolved.Source.TemplateID, nil
}

func contractError(findings []doccontract.Finding) error {
	return doccontract.ErrorFromFindings(findings)
}
