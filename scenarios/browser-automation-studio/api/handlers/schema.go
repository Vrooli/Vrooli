package handlers

import (
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/workflow/validator"
)

// SchemaHandler handles schema-related API endpoints.
type SchemaHandler struct {
	provider validator.SchemaProvider
	log      *logrus.Logger
}

// NewSchemaHandler creates a new schema handler.
func NewSchemaHandler(log *logrus.Logger) (*SchemaHandler, error) {
	provider, err := validator.NewSchemaProvider()
	if err != nil {
		return nil, err
	}
	return &SchemaHandler{
		provider: provider,
		log:      log,
	}, nil
}

// GetWorkflowSchema handles GET /api/v1/schema/workflow
// Query params:
//   - nodes: comma-separated list of node types to include (optional)
func (h *SchemaHandler) GetWorkflowSchema(w http.ResponseWriter, r *http.Request) {
	nodesParam := strings.TrimSpace(r.URL.Query().Get("nodes"))

	var schema []byte
	var err error

	if nodesParam == "" {
		schema, err = h.provider.GetFullSchema()
	} else {
		nodeTypes := parseNodeTypes(nodesParam)
		schema, err = h.provider.GetFilteredSchema(nodeTypes)
	}

	if err != nil {
		h.log.WithError(err).Error("failed to get workflow schema")
		RespondError(w, ErrInternalServer.WithMessage("failed to get workflow schema"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(schema)
}

// GetAvailableNodeTypes handles GET /api/v1/schema/workflow/node-types
func (h *SchemaHandler) GetAvailableNodeTypes(w http.ResponseWriter, r *http.Request) {
	nodeTypes := validator.GetAvailableNodeTypes()
	RespondSuccess(w, http.StatusOK, map[string]any{
		"node_types": nodeTypes,
	})
}

// GetStepDefinitions handles GET /api/v1/schema/steps
// Query params:
//   - types: comma-separated list of step types to include (optional)
//   - cli_only: if "true", only return CLI-supported step types (optional)
func (h *SchemaHandler) GetStepDefinitions(w http.ResponseWriter, r *http.Request) {
	typesParam := strings.TrimSpace(r.URL.Query().Get("types"))
	cliOnly := strings.TrimSpace(r.URL.Query().Get("cli_only")) == "true"

	// Get all definitions or CLI-only based on query param
	var allDefs []validator.StepDefinition
	if cliOnly {
		allDefs = validator.GetCLISupportedStepDefinitions()
	} else {
		allDefs = validator.GetStepDefinitions()
	}

	// Filter by types if specified
	var result []validator.StepDefinition
	if typesParam != "" {
		requestedTypes := parseNodeTypes(typesParam)
		requestedSet := make(map[string]bool)
		for _, t := range requestedTypes {
			requestedSet[t] = true
		}

		for _, def := range allDefs {
			if requestedSet[def.Type] {
				result = append(result, def)
			}
		}
	} else {
		result = allDefs
	}

	RespondSuccess(w, http.StatusOK, map[string]any{
		"steps": result,
	})
}

// parseNodeTypes splits a comma-separated list of node types.
func parseNodeTypes(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
