package capabilities

import (
	"context"
	"fmt"
	"path/filepath"

	"connectrpc.com/connect"
	"experience-manager/internal/capstatus"
	"experience-manager/internal/module"
	"experience-manager/internal/reconcile"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/capabilities"
	capabilitiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/capabilities/capabilities_v1connect"
)

type handler struct{ repoRoot string }

func Module(repoRoot string) module.Module {
	path, service := capabilitiesconnect.NewCapabilityStatusServiceHandler(&handler{repoRoot: repoRoot})
	return module.Module{
		Name:      "capability-status",
		Mount:     func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: service}) },
		Endpoints: []module.EndpointDescriptor{{ID: "capabilities_status", Path: capabilitiesconnect.CapabilityStatusServiceGetStatusProcedure, Method: "POST", Summary: "Report derived capability provability", Category: "capabilities"}},
	}
}

func (h *handler) GetStatus(_ context.Context, _ *connect.Request[capabilitiesv1.GetStatusRequest]) (*connect.Response[capabilitiesv1.GetStatusResponse], error) {
	reg, err := capstatus.Load(filepath.Join(h.repoRoot, "scenarios", "experience-manager", "capabilities"))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load capability registry: %w", err))
	}
	axes := map[string][]string{}
	profiles := reconcile.DefaultCaptureProfiles
	if loaded, loadErr := reconcile.CaptureProfilesFromAxes(filepath.Join(h.repoRoot, "scenarios", "experience-manager", "capabilities", "axes.json"), 12); loadErr == nil {
		profiles = loaded
	}
	for _, support := range reconcile.WiredAxesFromProfiles(profiles) {
		axes[support.Axis] = support.Values
	}
	report := capstatus.Derive(reg, capstatus.Support{Axes: axes, Evidence: reconcile.AvailableEvidence(), ClaimTypes: reconcile.ImplementedClaimTypes()})
	resp := &capabilitiesv1.GetStatusResponse{Provable: int32(report.ProvableTotal), Total: int32(len(report.Results))}
	for _, result := range report.Results {
		cap := findCapability(reg, result.Capability)
		row := &capabilitiesv1.CapabilityStatus{Id: result.Capability, Title: result.Title, Status: string(result.Status), Provable: result.Status == capstatus.StatusProvable, BlockingAxis: firstBlocker(result.Blockers, "axis "), BlockingEvidence: firstBlocker(result.Blockers, "evidence ")}
		if cap != nil {
			row.Facets = append(row.Facets, cap.Facets...)
		}
		resp.Capabilities = append(resp.Capabilities, row)
	}
	return connect.NewResponse(resp), nil
}

func findCapability(reg capstatus.Registry, id string) *capstatus.Capability {
	for i := range reg.Capabilities {
		if reg.Capabilities[i].ID == id {
			return &reg.Capabilities[i]
		}
	}
	return nil
}

func firstBlocker(blockers []string, prefix string) string {
	for _, blocker := range blockers {
		if len(blocker) >= len(prefix) && blocker[:len(prefix)] == prefix {
			return blocker
		}
	}
	return ""
}
