package handler

import (
	"net/http"

	"tunnel-manager/domain"
)

func HandlePortAudit(auditor PortAuditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results, err := auditor.Audit()
		if err != nil {
			writeError(w, err)
			return
		}
		if results == nil {
			results = []domain.PortAuditResult{}
		}

		// Compute summary
		violations := 0
		for _, r := range results {
			if r.Status != "compliant" {
				violations++
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"results":    results,
			"total":      len(results),
			"violations": violations,
			"compliant":  len(results) - violations,
		})
	}
}
