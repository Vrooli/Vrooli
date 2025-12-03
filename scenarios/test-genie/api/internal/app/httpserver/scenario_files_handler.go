package httpserver

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"test-genie/internal/scenarios"
)

func (s *Server) handleListScenarioFiles(w http.ResponseWriter, r *http.Request) {
	if s.scenarios == nil {
		s.writeError(w, http.StatusInternalServerError, "scenario directory service unavailable")
		return
	}
	params := mux.Vars(r)
	name := strings.TrimSpace(params["name"])
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "scenario name is required")
		return
	}

	query := r.URL.Query()
	path := strings.TrimSpace(query.Get("path"))
	search := strings.TrimSpace(query.Get("search"))
	limit := 0
	if rawLimit := strings.TrimSpace(query.Get("limit")); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil {
			limit = parsed
		}
	}
	includeHidden := strings.TrimSpace(query.Get("includeHidden")) == "1"

	result, err := s.scenarios.ListFilesWithMeta(r.Context(), name, scenarios.FileListOptions{
		Path:          path,
		Search:        search,
		Limit:         limit,
		IncludeHidden: includeHidden,
	})
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":       result.Items,
		"count":       len(result.Items),
		"hiddenCount": result.HiddenCount,
	})
}
