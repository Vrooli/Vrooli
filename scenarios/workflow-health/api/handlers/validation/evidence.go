package validation

// Browser evidence retains its original outcome and a separate applicability
// projection. Test Genie owns byte identities; this provider owns case scope.
import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"workflow-health/internal/workflows"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariov1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation/validation_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type evidenceScope struct {
	ID           string   `json:"id"`
	CasePatterns []string `json:"case_patterns"`
	Inputs       []string `json:"inputs"`
}

type evidenceConfig struct {
	CommonInputs []string        `json:"common_inputs"`
	Scopes       []evidenceScope `json:"scopes"`
}

type evidenceSnapshot struct {
	Scope    string                                     `json:"scope"`
	Cases    []string                                   `json:"cases"`
	Request  *validationv1.ResolveSourceIdentityRequest `json:"request,omitempty"`
	Identity string                                     `json:"identity,omitempty"`
	State    string                                     `json:"state"`
	Reason   string                                     `json:"reason,omitempty"`
}

type evidenceIdentityResolver func(context.Context, *validationv1.ResolveSourceIdentityRequest) (string, error)

func resolveEvidenceIdentity(ctx context.Context, request *validationv1.ResolveSourceIdentityRequest) (string, error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "test-genie")
	if err != nil {
		return "", err
	}
	client := validationconnect.NewValidationServiceClient(&http.Client{Timeout: 45 * time.Second}, base)
	response, err := client.ResolveSourceIdentity(ctx, connect.NewRequest(request))
	if err != nil {
		return "", err
	}
	if response.Msg.GetIdentity().GetIdentity() == "" {
		return "", fmt.Errorf("identity owner returned no content identity")
	}
	return response.Msg.GetIdentity().GetIdentity(), nil
}

func evidenceRuntimeFresh(ctx context.Context, scenario string) error {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "vrooli", "scenario", "freshness", scenario, "--json").Output()
	if err != nil {
		return fmt.Errorf("runtime freshness unavailable: %w", err)
	}
	var report cliv1.ScenarioFreshnessResponse
	if err := protojson.Unmarshal(output, &report); err != nil {
		return err
	}
	if !report.GetSuccess() || report.GetScenario() != scenario || report.GetStale() {
		return fmt.Errorf("target build is stale or freshness is unavailable")
	}
	return nil
}

// snapshotEvidence runs before BAS, never after the fact on historical output.
// Missing declarations use the complete owner-resolved scenario/build closure.
func (h *connectHandler) snapshotEvidence(ctx context.Context, scenario, target string, cases []string) []evidenceSnapshot {
	if h.deps.RepoRoot == "" {
		return nil
	}
	if target == "" {
		target = filepath.Join(h.deps.RepoRoot, "scenarios", scenario)
	}
	if len(cases) == 0 {
		catalog, err := workflows.ScanScenario(target)
		if err != nil {
			return []evidenceSnapshot{{Scope: "scenario", State: "unknown", Reason: err.Error()}}
		}
		for _, item := range catalog.Cases {
			cases = append(cases, item.Path)
		}
	}
	var document struct {
		Workflow struct {
			Evidence evidenceConfig `json:"evidence"`
		} `json:"workflow"`
	}
	data, readErr := os.ReadFile(filepath.Join(target, ".vrooli", "testing.json"))
	if readErr == nil {
		readErr = json.Unmarshal(data, &document)
	}
	if os.IsNotExist(readErr) {
		readErr = nil
	}
	canonical := filepath.Join(h.deps.RepoRoot, "scenarios", scenario)
	actual, err := filepath.Abs(target)
	if err != nil || filepath.Clean(actual) != filepath.Clean(canonical) {
		readErr = fmt.Errorf("evidence target path does not match its repository scenario")
	}
	ids := map[string]bool{}
	for _, scope := range document.Workflow.Evidence.Scopes {
		if ids[scope.ID] {
			readErr = fmt.Errorf("duplicate evidence scope id %q", scope.ID)
		}
		ids[scope.ID] = true
	}
	check := h.deps.EvidenceRuntimeFresh
	if check == nil {
		check = evidenceRuntimeFresh
	}
	if err := check(ctx, scenario); err != nil {
		readErr = err
	}
	groups := map[string]*evidenceSnapshot{}
	for _, name := range cases {
		selected := evidenceScope{ID: "scenario"}
		var matches int
		for _, scope := range document.Workflow.Evidence.Scopes {
			for _, pattern := range scope.CasePatterns {
				matched, err := path.Match(pattern, name)
				if err != nil {
					readErr = fmt.Errorf("invalid evidence case pattern %q", pattern)
				}
				if matched {
					selected = scope
					matches++
					break
				}
			}
		}
		if matches > 1 {
			readErr = fmt.Errorf("ambiguous evidence scope for %s", name)
		}
		if matches == 1 && (selected.ID == "" || selected.ID == "scenario" || len(selected.Inputs) == 0 || len(document.Workflow.Evidence.CommonInputs) == 0) {
			readErr = fmt.Errorf("evidence scope needs unique id, inputs and common_inputs")
		}
		group := groups[selected.ID]
		if group == nil {
			request := &validationv1.ResolveSourceIdentityRequest{ContentInputs: []*validationv1.ContentInputRoot{{Name: "case-source", Root: "scenarios/" + scenario}}}
			if matches == 0 {
				request.Targets = []*commonv1.ValidationTarget{{Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO, Id: scenario}}
				request.ContentInputs[0].Selections = []*validationv1.InputSelection{{Glob: "**", Required: true}}
			} else {
				inputs := append(slices.Clone(document.Workflow.Evidence.CommonInputs), selected.Inputs...)
				inputs = append(inputs, "scenarios/"+scenario+"/.vrooli/testing.json")
				dependency := &validationv1.ContentInputRoot{Name: "declared-dependencies", Root: ".", Dependency: true}
				for _, glob := range inputs {
					dependency.Selections = append(dependency.Selections, &validationv1.InputSelection{Glob: glob, Required: true})
				}
				request.ContentInputs = append(request.ContentInputs, dependency)
			}
			group = &evidenceSnapshot{Scope: selected.ID, Request: request, State: "unknown"}
			groups[selected.ID] = group
		}
		group.Cases = append(group.Cases, name)
		if matches != 0 {
			group.Request.ContentInputs[0].Selections = append(group.Request.ContentInputs[0].Selections, &validationv1.InputSelection{Glob: name, Required: true})
		}
	}
	// Include referenced BAS fragments/seeds, rather than hashing only the entry
	// case and silently reusing evidence after a dependency changed.
	catalog, catalogErr := workflows.ScanScenario(target)
	if catalogErr != nil {
		readErr = catalogErr
	}
	var out []evidenceSnapshot
	for _, group := range groups {
		if group.Scope != "scenario" && catalog != nil {
			seen := map[string]bool{}
			for _, name := range group.Cases {
				seen[name] = true
			}
			for changed := true; changed; {
				changed = false
				for _, edge := range catalog.DependencyEdges {
					dependency := edge.ToPath
					if dependency != "" && !strings.HasPrefix(dependency, "bas/") {
						dependency = "bas/" + dependency
					}
					if seen[edge.FromPath] && dependency != "" && !seen[dependency] {
						seen[dependency] = true
						changed = true
						group.Request.ContentInputs[0].Selections = append(group.Request.ContentInputs[0].Selections, &validationv1.InputSelection{Glob: dependency, Required: true})
					}
				}
			}
		}
		if readErr != nil {
			group.Reason = readErr.Error()
		} else {
			identity, err := h.evidenceResolver()(ctx, group.Request)
			group.Identity = identity
			if err != nil {
				group.Reason = err.Error()
			} else {
				group.State = "captured"
			}
		}
		out = append(out, *group)
	}
	slices.SortFunc(out, func(a, b evidenceSnapshot) int { return strings.Compare(a.Scope, b.Scope) })
	return out
}

func (h *connectHandler) evidenceResolver() evidenceIdentityResolver {
	if h.deps.EvidenceIdentity != nil {
		return h.deps.EvidenceIdentity
	}
	return resolveEvidenceIdentity
}

func (h *connectHandler) qualifyEvidence(ctx context.Context, snapshots []evidenceSnapshot) []evidenceSnapshot {
	for i := range snapshots {
		snapshot := &snapshots[i]
		if snapshot.State != "captured" {
			continue
		}
		current, err := h.evidenceResolver()(ctx, snapshot.Request)
		switch {
		case err != nil:
			snapshot.State, snapshot.Reason = "unknown", err.Error()
		case current != snapshot.Identity:
			snapshot.State, snapshot.Reason = "stale", "source dependencies changed during execution"
		default:
			snapshot.State = "qualified"
		}
	}
	return snapshots
}

// currentEvidence decorates a cloned read projection. It never rewrites the
// retained terminal result or turns a failed run into a successful execution.
func (h *connectHandler) currentEvidence(ctx context.Context, run *scenariov1.ValidationRun) *scenariov1.ValidationRun {
	if run.GetTerminalResult().GetNativeDetail() == nil {
		return run
	}
	var native structpb.Struct
	if err := run.GetTerminalResult().GetNativeDetail().UnmarshalTo(&native); err != nil {
		return run
	}
	fields := native.AsMap()
	encoded, _ := json.Marshal(fields["evidence_snapshots"])
	var snapshots []evidenceSnapshot
	if json.Unmarshal(encoded, &snapshots) != nil {
		return run
	}
	var assessed []any
	for _, snapshot := range snapshots {
		state, reason := "unknown", snapshot.Reason
		current := ""
		if snapshot.State == "qualified" && snapshot.Identity != "" && snapshot.Request != nil {
			var err error
			current, err = h.evidenceResolver()(ctx, snapshot.Request)
			switch {
			case err != nil:
				reason = err.Error()
			case current != snapshot.Identity:
				state, reason = "stale", "declared source dependencies changed; rerun affected cases"
			default:
				state, reason = "current", "captured source dependency identity is unchanged"
			}
		}
		assessed = append(assessed, map[string]any{"scope": snapshot.Scope, "cases": stringValues(snapshot.Cases), "state": state, "reason": reason, "captured_identity": snapshot.Identity, "current_identity": current})
	}
	fields["current_evidence"] = map[string]any{"schema_version": 1, "scopes": assessed, "historical_without_snapshot": len(snapshots) == 0}
	updated, err := structpb.NewStruct(fields)
	if err != nil {
		return run
	}
	result := proto.Clone(run).(*scenariov1.ValidationRun)
	result.TerminalResult.NativeDetail, err = anypb.New(updated)
	if err != nil {
		return run
	}
	return result
}
