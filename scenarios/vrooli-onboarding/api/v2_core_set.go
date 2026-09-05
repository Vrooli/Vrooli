package main

import (
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	apicoreset "github.com/vrooli/api-core/coreset"
	"github.com/vrooli/vrooli/internal/app/supervision"
)

type coreSetResponse struct {
	Available    bool                `json:"available"`
	Seed         []string            `json:"seed"`
	TrustedBase  []string            `json:"trusted_base"`
	Members      []apicoreset.Member `json:"members"`
	MemberCounts map[string]int      `json:"member_counts"`
	LoadErrors   map[string]string   `json:"load_errors,omitempty"`
	Error        string              `json:"error,omitempty"`
}

func (s *Server) handleV2CoreSet(w http.ResponseWriter, r *http.Request) {
	state, err := loadOperatorStateFor(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if state.Core == nil {
		writeJSON(w, http.StatusFailedDependency, map[string]string{"error": "operator state has no core authority"})
		return
	}
	seed := append([]string(nil), state.Core.Seed...)
	if proposed, present := r.URL.Query()["seed"]; present {
		seed = normalizeCoreSeed(proposed)
	}
	authority := apicoreset.Authority{Seed: seed, TrustedBase: append([]string(nil), state.Core.TrustedBase...)}
	if err := authority.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "core authority validation failed: " + err.Error()})
		return
	}

	response := coreSetResponse{Seed: seed, TrustedBase: authority.TrustedBase}
	root := strings.TrimSpace(s.roots.RepoRoot)
	if root == "" {
		response.Error = "closure unavailable outside a repository source tree; the operator seed remains authoritative"
		writeJSON(w, http.StatusOK, response)
		return
	}
	report := supervision.Compute(filepath.Join(root, "scenarios"), authority)
	response.Available = true
	response.Members = report.Members
	response.MemberCounts = report.MemberCounts
	response.LoadErrors = report.LoadErrors
	writeJSON(w, http.StatusOK, response)
}

func normalizeCoreSeed(values []string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if part = strings.ToLower(strings.TrimSpace(part)); part != "" {
				seen[part] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
