package handlers

import (
	"encoding/json"
	"net/http"

	"brand-manager/apierr"
	"brand-manager/contrast"
)

// contrastCheckRequest is the expected JSON body for POST /api/v1/contrast/check.
type contrastCheckRequest struct {
	Foreground string `json:"foreground"`
	Background string `json:"background"`
}

// brandContrastRequest is the expected JSON body for POST /api/v1/contrast/brand.
type brandContrastRequest struct {
	Primary    string `json:"primary"`
	Secondary  string `json:"secondary"`
	Accent     string `json:"accent"`
	Background string `json:"background"`
	Surface    string `json:"surface"`
	Text       string `json:"text"`
}

// contrastChecker returns a Checker configured from the handler's config.
func (h *Handlers) contrastChecker() *contrast.Checker {
	return contrast.NewChecker(h.cfg)
}

// CheckContrast handles POST /api/v1/contrast/check.
// Checks a single foreground/background pair for WCAG AA compliance.
// [REQ:BM-REQ-WCAG-CALC] [REQ:BM-REQ-WCAG-VALIDATE]
func (h *Handlers) CheckContrast(w http.ResponseWriter, r *http.Request) {
	var req contrastCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.Write(w, apierr.Validation("invalid request body"))
		return
	}
	if req.Foreground == "" || req.Background == "" {
		apierr.Write(w, apierr.Validation("foreground and background are required"))
		return
	}

	result, err := h.contrastChecker().CheckPair(req.Foreground, req.Background)
	if err != nil {
		apierr.Write(w, apierr.Validation("invalid color value: "+err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// CheckBrandContrast handles POST /api/v1/contrast/brand.
// Validates all standard WCAG AA pairings for a brand's color palette.
// [REQ:BM-REQ-WCAG-VALIDATE] [REQ:BM-REQ-WCAG-REJECT]
func (h *Handlers) CheckBrandContrast(w http.ResponseWriter, r *http.Request) {
	var req brandContrastRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.Write(w, apierr.Validation("invalid request body"))
		return
	}

	result, err := h.contrastChecker().CheckBrandColors(
		req.Primary, req.Secondary, req.Accent,
		req.Background, req.Surface, req.Text,
	)
	if err != nil {
		apierr.Write(w, apierr.Validation("invalid color value: "+err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, result)
}
