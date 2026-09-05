package main

import (
	"net/http"
	"strings"
)

// glossaryEntry maps a technical term to a plain language description.
type glossaryEntry struct {
	Term        string `json:"term"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// defaultGlossary provides plain language translations for common Vrooli terms.
var defaultGlossary = []glossaryEntry{
	{Term: "resource", Description: "A local service that your applications can use, like a database or AI model", Category: "core"},
	{Term: "scenario", Description: "A complete application or microservice built from resources and other scenarios", Category: "core"},
	{Term: "service.json", Description: "The main configuration file that defines how your application is set up", Category: "config"},
	{Term: "postgres", Description: "A powerful relational database for storing structured data", Category: "database"},
	{Term: "redis", Description: "A fast in-memory data store for caching and messaging", Category: "database"},
	{Term: "ollama", Description: "A local AI model runner that lets you use language models without cloud services", Category: "ai"},
	{Term: "qdrant", Description: "A vector database for storing and searching AI embeddings", Category: "database"},
	{Term: "vault", Description: "A secrets management tool that securely stores passwords and API keys", Category: "security"},
	{Term: "health check", Description: "An automatic test that verifies a service is running correctly", Category: "operations"},
	{Term: "port", Description: "A numbered channel that services use to communicate (like a phone extension)", Category: "networking"},
	{Term: "dependency", Description: "A service that another service needs to work properly", Category: "core"},
	{Term: "lifecycle", Description: "The stages a service goes through: setup, start, run, and stop", Category: "operations"},
	{Term: "API", Description: "Application Programming Interface - how software components talk to each other", Category: "core"},
	{Term: "endpoint", Description: "A specific URL where a service accepts requests", Category: "networking"},
}

func (s *Server) handleGlossary(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q"))

	if q == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"entries": defaultGlossary,
			"count":   len(defaultGlossary),
		})
		return
	}

	// Filter by search term
	var results []glossaryEntry
	for _, entry := range defaultGlossary {
		if strings.Contains(strings.ToLower(entry.Term), q) ||
			strings.Contains(strings.ToLower(entry.Description), q) {
			results = append(results, entry)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"entries": results,
		"count":   len(results),
		"query":   q,
	})
}
