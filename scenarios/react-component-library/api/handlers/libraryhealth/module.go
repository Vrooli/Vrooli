// Package libraryhealth exposes the Stats read composition over Connect-RPC.
//
// It owns no tables. internal/libraryhealth composes overview, quality,
// activity and services; this package maps the snapshot onto the generated
// proto and never returns a zero where a source was unavailable.
package libraryhealth

import (
	"context"
	"database/sql"
	"errors"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	libraryhealthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/libraryhealth"
	libraryhealthconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/libraryhealth/libraryhealth_v1connect"

	domain "react-component-library/internal/libraryhealth"
	"react-component-library/internal/module"
)

// Handler adapts the read composition onto the generated Connect interface.
type Handler struct{ service *domain.Service }

// NewHandler wires a composed service.
func NewHandler(service *domain.Service) *Handler { return &Handler{service: service} }

// Module mounts LibraryHealthService with the default DB-backed providers.
func Module(db *sql.DB) module.Module {
	return ModuleFromProviders(domain.DBProviders(db))
}

// ModuleFromProviders lets the caller supply the capability registry or a
// test double for any section.
func ModuleFromProviders(providers domain.Providers) module.Module {
	return ModuleFromService(domain.NewService(providers))
}

// ModuleFromService is the constructor tests use when they need a custom
// clock or fully injected service.
func ModuleFromService(service *domain.Service) module.Module {
	path, handler := libraryhealthconnect.NewLibraryHealthServiceHandler(NewHandler(service))
	return module.Module{
		Name: "libraryhealth",
		Mount: func(router *mux.Router) {
			connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — library health owns no tables. The empty provider keeps
// the modules registry uniform without implying this package stores anything.
func Schema() string { return "" }

// GetLibraryHealth returns every section with per-section availability.
func (h *Handler) GetLibraryHealth(ctx context.Context, req *connect.Request[libraryhealthv1.GetLibraryHealthRequest]) (*connect.Response[libraryhealthv1.GetLibraryHealthResponse], error) {
	if h == nil || h.service == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("library health service is not configured"))
	}
	health := h.service.Get(ctx, int(req.Msg.GetActivityDays()))
	return connect.NewResponse(&libraryhealthv1.GetLibraryHealthResponse{Health: toProtoHealth(health)}), nil
}

func toProtoHealth(health domain.Health) *libraryhealthv1.LibraryHealth {
	return &libraryhealthv1.LibraryHealth{
		Overview: toProtoSection(health.Overview),
		Quality:  toProtoSection(health.Quality),
		Activity: toProtoSection(health.Activity),
		Services: toProtoSection(health.Services),
		Counts: &libraryhealthv1.HealthOverview{
			Assets:       int32(health.Counts.Assets),
			Published:    int32(health.Counts.Published),
			Workspaces:   int32(health.Counts.Workspaces),
			FindingsOpen: int32(health.Counts.FindingsOpen),
			CoverageRows: int32(health.Counts.CoverageRows),
		},
		GeneratedAt: health.GeneratedAt,
	}
}

func toProtoSection(section domain.Section) *libraryhealthv1.HealthSection {
	metrics := make([]*libraryhealthv1.HealthMetric, 0, len(section.Metrics))
	for _, metric := range section.Metrics {
		metrics = append(metrics, &libraryhealthv1.HealthMetric{
			Key:    metric.Key,
			Label:  metric.Label,
			Value:  int32(metric.Value),
			Total:  int32(metric.Total),
			Detail: metric.Detail,
		})
	}
	return &libraryhealthv1.HealthSection{
		State:       toProtoState(section.State),
		Reason:      section.Reason,
		Metrics:     metrics,
		GeneratedAt: section.GeneratedAt,
	}
}

func toProtoState(state domain.SectionState) libraryhealthv1.SectionState {
	switch state {
	case domain.StateAvailable:
		return libraryhealthv1.SectionState_SECTION_STATE_AVAILABLE
	case domain.StateEmpty:
		return libraryhealthv1.SectionState_SECTION_STATE_EMPTY
	case domain.StatePartial:
		return libraryhealthv1.SectionState_SECTION_STATE_PARTIAL
	case domain.StateUnavailable:
		return libraryhealthv1.SectionState_SECTION_STATE_UNAVAILABLE
	default:
		return libraryhealthv1.SectionState_SECTION_STATE_UNSPECIFIED
	}
}

// Endpoints is the static surface consumed by `make endpoints`.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "library_health_get",
		Path:        libraryhealthconnect.LibraryHealthServiceGetLibraryHealthProcedure,
		Method:      "POST",
		Summary:     "Get the library health snapshot",
		Description: "Returns overview, quality, activity and services sections, each with available/empty/partial/unavailable state and a reason when degraded. activity_days defaults to 30 and is capped at 365. Unavailable sources carry no zero-valued metrics.",
		Category:    "library-health",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"activity_days": "integer (0 means 30, max 365)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"health": "LibraryHealth {overview, quality, activity, services, counts, generated_at}"}},
	},
}
