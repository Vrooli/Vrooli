package selection

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/setup/v1"
	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	internalselection "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/selection"
	"google.golang.org/protobuf/encoding/protojson"
)

// handoffRESTRequest is the stable, proto-free bridge-facing envelope. The
// generated Connect route remains canonical for typed callers; this route is
// an intentionally narrow compatibility adapter for Bridge's SSH onboarding
// transport.
type handoffRESTRequest struct {
	Target           string          `json:"target"`
	MachineID        string          `json:"machine_id"`
	NodeID           string          `json:"node_id"`
	NodeKind         string          `json:"node_kind"`
	DesiredSelection json.RawMessage `json:"desired_selection,omitempty"`
}

// selectionRESTProjection keeps the bridge transport independent of generated
// protobuf implementation details while preserving every setup/v1 field.
type selectionRESTProjection struct {
	SchemaVersion                 string                                    `json:"schema_version,omitempty"`
	Target                        string                                    `json:"target,omitempty"`
	Scenarios                     []string                                  `json:"scenarios,omitempty"`
	OptionalResources             []string                                  `json:"optional_resources,omitempty"`
	CoreSeed                      []string                                  `json:"core_seed,omitempty"`
	TrustedBase                   []string                                  `json:"trusted_base,omitempty"`
	HostTools                     []string                                  `json:"host_tools,omitempty"`
	HostSafeguards                []string                                  `json:"host_safeguards,omitempty"`
	CredentialAddresses           []string                                  `json:"credential_addresses,omitempty"`
	TrustPosture                  string                                    `json:"trust_posture,omitempty"`
	UpdateControl                 string                                    `json:"update_control,omitempty"`
	SessionMode                   string                                    `json:"session_mode,omitempty"`
	OperatingMode                 map[string]string                         `json:"operating_mode,omitempty"`
	Apply                         bool                                      `json:"apply,omitempty"`
	CapacityPosture               string                                    `json:"capacity_posture,omitempty"`
	TransientHeadroomReserveBytes uint64                                    `json:"transient_headroom_reserve_bytes,omitempty"`
	ResourceCapacity              map[string]resourceCapacityRESTProjection `json:"resource_capacity,omitempty"`
	FieldPresence                 map[string]string                         `json:"field_presence,omitempty"`
}

type resourceCapacityRESTProjection struct {
	Rung             string            `json:"rung,omitempty"`
	Tunables         map[string]string `json:"tunables,omitempty"`
	GPUIndex         uint32            `json:"gpu_index,omitempty"`
	Priority         string            `json:"priority,omitempty"`
	YieldWhenIdle    bool              `json:"yield_when_idle,omitempty"`
	IdleGraceSeconds uint32            `json:"idle_grace_seconds,omitempty"`
}

// RESTHandoffHandler mounts the stable /api/v2/handoff adapter consumed by
// Bridge. It uses the same service and authorization boundary as Connect, so
// the two routes cannot develop different selection semantics.
func RESTHandoffHandler(service internalselection.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := authz.RequireMutation(r.Context()); err != nil {
			writeRESTError(w, err)
			return
		}
		var request handoffRESTRequest
		decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeRESTError(w, connect.NewError(connect.CodeInvalidArgument, err))
			return
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			writeRESTError(w, connect.NewError(connect.CodeInvalidArgument, errors.New("request must contain exactly one JSON object")))
			return
		}
		protoRequest := &selectionv1.CreateHandoffRequest{Target: request.Target, MachineId: request.MachineID, NodeId: request.NodeID, NodeKind: request.NodeKind}
		if len(request.DesiredSelection) > 0 && string(request.DesiredSelection) != "null" {
			selection := &setupv1.Selection{}
			if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(request.DesiredSelection, selection); err != nil {
				writeRESTError(w, connect.NewError(connect.CodeInvalidArgument, err))
				return
			}
			protoRequest.DesiredSelection = selection
		}
		response, err := service.Handoff(r.Context(), protoRequest)
		if err != nil {
			writeRESTError(w, err)
			return
		}
		selection := response.GetSelection()
		if selection == nil {
			writeRESTError(w, connect.NewError(connect.CodeInternal, errors.New("onboarding returned no selection")))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(selectionRESTProjectionFromProto(selection))
	})
}

func selectionRESTProjectionFromProto(selection *setupv1.Selection) selectionRESTProjection {
	projection := selectionRESTProjection{
		SchemaVersion: selection.GetSchemaVersion(), Target: selection.GetTarget(),
		Scenarios: append([]string(nil), selection.GetScenarios()...), OptionalResources: append([]string(nil), selection.GetOptionalResources()...),
		CoreSeed: append([]string(nil), selection.GetCoreSeed()...), TrustedBase: append([]string(nil), selection.GetTrustedBase()...),
		HostTools: append([]string(nil), selection.GetHostTools()...), HostSafeguards: append([]string(nil), selection.GetHostSafeguards()...),
		CredentialAddresses: append([]string(nil), selection.GetCredentialAddresses()...), TrustPosture: selection.GetTrustPosture(),
		UpdateControl: selection.GetUpdateControl(), SessionMode: selection.GetSessionMode(), OperatingMode: cloneStringMap(selection.GetOperatingMode()),
		Apply: selection.GetApply(), CapacityPosture: selection.GetCapacityPosture(), TransientHeadroomReserveBytes: selection.GetTransientHeadroomReserveBytes(),
		ResourceCapacity: map[string]resourceCapacityRESTProjection{}, FieldPresence: map[string]string{},
	}
	for name, value := range selection.GetResourceCapacity() {
		if value == nil {
			continue
		}
		projection.ResourceCapacity[name] = resourceCapacityRESTProjection{Rung: value.GetRung(), Tunables: cloneStringMap(value.GetTunables()), GPUIndex: value.GetGpuIndex(), Priority: value.GetPriority(), YieldWhenIdle: value.GetYieldWhenIdle(), IdleGraceSeconds: value.GetIdleGraceSeconds()}
	}
	for name, value := range selection.GetFieldPresence() {
		projection.FieldPresence[name] = value.String()
	}
	return projection
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func writeRESTError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch connect.CodeOf(err) {
	case connect.CodeInvalidArgument:
		status = http.StatusBadRequest
	case connect.CodeUnauthenticated:
		status = http.StatusUnauthorized
	case connect.CodePermissionDenied:
		status = http.StatusForbidden
	case connect.CodeFailedPrecondition:
		status = http.StatusPreconditionFailed
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": strings.TrimSpace(err.Error())})
}
