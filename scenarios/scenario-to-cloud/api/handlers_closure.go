package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/bundle"
	"scenario-to-cloud/closure"
	"scenario-to-cloud/internal/httputil"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/discovery"
)

// closureServiceOverride lets tests substitute a fixture-backed service. The
// production service is built lazily from the repository the API runs in.
var (
	closureServiceOverride *closure.Service
	closureServiceOnce     sync.Once
	closureServiceValue    *closure.Service
	closureServiceErr      error
)

// closureService returns the process-wide closure service.
func closureService() (*closure.Service, error) {
	if closureServiceOverride != nil {
		return closureServiceOverride, nil
	}
	closureServiceOnce.Do(func() {
		closureServiceValue, closureServiceErr = newDefaultClosureService()
	})
	return closureServiceValue, closureServiceErr
}

// newDefaultClosureService binds the repository catalog, the analyzer over
// service discovery and the shared host requirement resolver.
func newDefaultClosureService() (*closure.Service, error) {
	repoRoot, err := bundle.FindRepoRootFromCWD()
	if err != nil {
		return nil, fmt.Errorf("find repo root: %w", err)
	}
	catalog, err := closure.NewRepoCatalog(repoRoot)
	if err != nil {
		return nil, err
	}
	return &closure.Service{
		RepoRoot: repoRoot,
		Catalog:  catalog,
		Analyzer: closure.NewHTTPAnalyzer(func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "scenario-dependency-analyzer")
		}),
		HostRequirements: closure.ControlPlaneHostRequirements{},
		DefaultPlatform:  closure.Platform{OS: "linux", Arch: "amd64"},
	}, nil
}

// registerClosureRoutes mounts the closure endpoints. Both return the closure
// document itself: its reasons and unsupported list are the explanation, so
// API, CLI and UI show the same text.
func (s *Server) registerClosureRoutes(api *mux.Router) {
	api.HandleFunc("/closure", s.handleGetClosure).Methods("GET")
	api.HandleFunc("/closure/explain", s.handleExplainClosure).Methods("POST")
}

// handleGetClosure resolves the closure for ?scenario=&environment=&os=&arch=[&scope=].
func (s *Server) handleGetClosure(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	request := closure.Request{
		ScenarioID:  strings.TrimSpace(query.Get("scenario")),
		Environment: strings.TrimSpace(query.Get("environment")),
		OS:          strings.TrimSpace(query.Get("os")),
		Arch:        strings.TrimSpace(query.Get("arch")),
		Scope:       strings.TrimSpace(query.Get("scope")),
	}
	s.writeClosure(w, r, request)
}

// handleExplainClosure resolves the closure for a JSON closure.Request,
// including operator overrides and target capacity.
func (s *Server) handleExplainClosure(w http.ResponseWriter, r *http.Request) {
	var request closure.Request
	if !httputil.DecodeRequestBody(w, r, &request) {
		return
	}
	s.writeClosure(w, r, request)
}

func (s *Server) writeClosure(w http.ResponseWriter, r *http.Request, request closure.Request) {
	if err := validateClosureRequest(request); err != nil {
		apierrors.Write(w, err)
		return
	}
	service, err := closureService()
	if err != nil {
		apierrors.Write(w, &apierrors.Error{Code: apierrors.CodeClosureUnavailable, Message: "closure catalogs are not available", Details: map[string]any{"cause": err.Error()}})
		return
	}
	resolved, err := service.Derive(r.Context(), request)
	if err != nil {
		apierrors.Write(w, closureAPIError(err))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, resolved)
}

func validateClosureRequest(request closure.Request) error {
	missing := []string{}
	if request.ScenarioID == "" {
		missing = append(missing, "scenario")
	}
	if request.OS == "" {
		missing = append(missing, "os")
	}
	if request.Arch == "" {
		missing = append(missing, "arch")
	}
	if len(missing) > 0 {
		return &apierrors.Error{Code: apierrors.CodeInvalidRequest, Message: "scenario, os and arch are required", Details: map[string]any{"missing": missing}}
	}
	return nil
}

// closureAPIError maps a typed closure error onto the API error model. The
// closure's own codes and details pass through unchanged.
func closureAPIError(err error) *apierrors.Error {
	var typed *closure.Error
	if errors.As(err, &typed) {
		return &apierrors.Error{Code: typed.Code, Message: typed.Message, Details: typed.Details, HTTPStatus: typed.Status()}
	}
	return apierrors.Internal("failed to derive closure", err)
}
