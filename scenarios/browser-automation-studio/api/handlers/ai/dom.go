package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/constants"
)

// DOMHandler handles DOM tree extraction operations
type DOMHandler struct {
	log *logrus.Logger
}

// NewDOMHandler creates a new DOM handler
func NewDOMHandler(log *logrus.Logger) *DOMHandler {
	return &DOMHandler{log: log}
}

// ExtractDOMTree extracts the DOM tree from a given URL
func (h *DOMHandler) ExtractDOMTree(ctx context.Context, url string) (string, error) {
	// Normalize URL - add protocol if missing
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// JavaScript to extract DOM tree with selectors - simplified and robust
	script := `
	try {
		// Wait a bit for React to render
		await new Promise(resolve => setTimeout(resolve, 2000));

		const body = document.body || document.documentElement;
		const result = {
			tagName: "BODY",
			id: null,
			className: null,
			text: null,
			selector: "body",
			children: []
		};

		if (!body) {
			return result;
		}

		// Simple extraction - just get immediate children
		const children = Array.from(body.children || []);
		for (let i = 0; i < Math.min(children.length, 10); i++) {
			const child = children[i];
			if (child && child.tagName && !['SCRIPT', 'STYLE', 'NOSCRIPT'].includes(child.tagName)) {
				const childData = {
					tagName: child.tagName,
					id: child.id || null,
					className: child.className || null,
					text: (child.textContent || "").substring(0, 50).trim() || null,
					selector: child.id ? "#" + child.id : child.tagName.toLowerCase()
				};
				result.children.push(childData);
			}
		}

		return result;
	} catch (e) {
		return {
			tagName: "ERROR",
			selector: "error",
			text: e.message,
			children: []
		};
	}`

	// Create temp file for output
	tmpFile, err := os.CreateTemp("", "dom-extract-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Use browserless extract to get the DOM
	extractCmd := exec.CommandContext(ctx, "resource-browserless", "extract", url,
		"--script", script,
		"--output", tmpFile.Name())

	output, err := extractCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to extract DOM: %w, output: %s", err, string(output))
	}

	// Read the extracted DOM data
	domData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read DOM data: %w", err)
	}

	// Return the extracted DOM data directly (browserless now returns proper JSON)
	return string(domData), nil
}

// GetDOMTree handles POST /api/v1/dom-tree - returns the DOM structure for Browser Inspector
func (h *DOMHandler) GetDOMTree(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.WithError(err).Error("Failed to decode DOM tree request")
		RespondError(w, ErrInvalidRequest)
		return
	}

	if req.URL == "" {
		RespondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "url"}))
		return
	}

	h.log.WithField("url", req.URL).Info("Extracting DOM tree")

	ctx, cancel := context.WithTimeout(r.Context(), constants.ElementAnalysisTimeout)
	defer cancel()

	domData, err := h.ExtractDOMTree(ctx, req.URL)
	if err != nil {
		h.log.WithError(err).Error("Failed to extract DOM tree")
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "extract_dom", "error": err.Error()}))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(domData))
}
