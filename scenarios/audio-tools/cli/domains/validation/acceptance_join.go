package validation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/types/known/timestamppb"

	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	experimentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment/experiment_v1connect"
)

const (
	acceptanceTargetRevision = "portable-voice-v2"
	compositeEvidenceMaxAge  = 24 * time.Hour
)

var compositeRequiredRows = []string{
	"dictation-trust", "provider-parity", "speaker-policy", "provider-evidence",
	"mobile-recovery", "interactive-latency", "corpus-quality", "owned-settlement",
	"explicit-routing", "host-portability", "private-lifecycle", "acceptance-integrity",
	"desktop-dictation-parity", "adapter-maintainability", "efficient-model-selection",
	"advanced-voice-workflows", "standalone-offer-upgrade", "offline-install-lifecycle",
}

type compositeSource struct {
	ArtifactRef string
	ObservedAt  time.Time
	Lane        string
	Claim       map[string]any
}

// runAcceptanceJoin is the owner-side final acceptance producer. The board is
// intentionally a read-only resolver; this command is the only path allowed
// to combine independently owned receipts. It preserves each claim's source
// artifact and execution lane so a composite receipt cannot be mistaken for a
// native run or for live/paid evidence.
func (h *handlers) runAcceptanceJoin(ctx cliapp.RunContext) error {
	if h.core == nil {
		return errors.New("validation run-acceptance-join requires the scenario CLI core")
	}
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	client := experimentconnect.NewExperimentServiceClient(httpClient, baseURL)
	if err := requireLabQualificationEvidenceAPI(context.Background(), client, "validation run-acceptance-join"); err != nil {
		return err
	}
	resp, err := client.ListQualificationEvidence(context.Background(), connect.NewRequest(&experimentv1.ListQualificationEvidenceRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list owner acceptance evidence", err, nil)
	}
	now := h.Now().UTC()
	sources, missing := selectCompositeSources(resp.Msg.GetEvidence(), now)
	machine, build, digest, err := acceptanceJoinMachineJSON(sources, missing)
	if err != nil {
		return fmt.Errorf("encode acceptance join: %w", err)
	}
	artifact := "audio-tools:acceptance-join:" + now.Format("20060102T150405.000000000Z") + ":" + hex.EncodeToString(digest[:8])
	_, err = client.RecordQualificationEvidence(context.Background(), connect.NewRequest(&experimentv1.RecordQualificationEvidenceRequest{Evidence: &experimentv1.QualificationEvidence{
		EngineId: "acceptance-join", ModelId: acceptanceTargetRevision, Strategy: "composite", PolicyProfile: "policy-v1",
		Kind:   experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_ACCEPTANCE_COMPOSITE,
		Passed: true, ArtifactRef: artifact,
		Notes:       "owner-validated composite acceptance join; per-row source artifacts and simulated/native lanes retained in machine_json",
		MachineJson: string(machine), ObservedAt: timestamppb.New(now),
	}}))
	if err != nil {
		return cliapp.WrapAPIError("record acceptance join", err, nil)
	}
	if ctx.JSON() {
		return json.NewEncoder(ctx.Stdout()).Encode(map[string]any{"passed": true, "acceptance_complete": len(missing) == 0, "missing_rows": missing, "artifact_ref": artifact, "source_count": len(sources), "build": build})
	}
	fmt.Fprintf(ctx.Stdout(), "acceptance evidence joined: %s (%d fresh owner sources; %d required rows unavailable)\n", artifact, len(sources), len(missing))
	return nil
}

func acceptanceJoinMachineJSON(sources map[string]compositeSource, missing []string) ([]byte, string, [32]byte, error) {
	refs := make([]string, 0, len(sources))
	lanes := make(map[string]string, len(sources))
	claims := make(map[string]any, len(sources))
	for row, source := range sources {
		refs = append(refs, source.ArtifactRef)
		lanes[row] = source.Lane
		claims[row] = source.Claim
	}
	sort.Strings(refs)
	seed := strings.Join(refs, "\n")
	digest := sha256.Sum256([]byte(seed))
	build := "acceptance-join:" + hex.EncodeToString(digest[:12])
	machine, err := json.Marshal(map[string]any{
		"target_revision":     acceptanceTargetRevision,
		"method":              "audio-tools-acceptance-v2",
		"cohort":              "voice-pilot-en-v1",
		"build":               build,
		"device_profile":      "D1+SIMULATED",
		"policy_revision":     "policy-v1",
		"corpus_revision":     "voice-pilot-en-v1",
		"execution_lane":      "composite",
		"simulated_as_native": false,
		"provider_cell": map[string]string{
			"engine_id":      "acceptance-join",
			"model_id":       acceptanceTargetRevision,
			"strategy":       "composite",
			"device_profile": "D1+SIMULATED",
		},
		"acceptance": claims,
		"details": map[string]any{
			"source_artifacts":      refs,
			"claim_execution_lanes": lanes,
			"source_count":          len(sources),
			"missing_rows":          missing,
			"acceptance_complete":   len(missing) == 0,
			"max_source_age":        compositeEvidenceMaxAge.String(),
		},
	})
	return machine, build, digest, err
}

func selectCompositeSources(items []*experimentv1.QualificationEvidence, now time.Time) (map[string]compositeSource, []string) {
	selected := make(map[string]compositeSource)
	for _, item := range items {
		if item == nil || !item.GetPassed() || strings.TrimSpace(item.GetArtifactRef()) == "" {
			continue
		}
		observed := item.GetObservedAt()
		if observed == nil {
			continue
		}
		at := observed.AsTime()
		if at.After(now) || now.Sub(at) > compositeEvidenceMaxAge {
			continue
		}
		var envelope map[string]any
		if json.Unmarshal([]byte(item.GetMachineJson()), &envelope) != nil ||
			envelope["target_revision"] != acceptanceTargetRevision ||
			envelope["method"] != "audio-tools-acceptance-v2" ||
			envelope["cohort"] != "voice-pilot-en-v1" ||
			envelope["policy_revision"] != "policy-v1" ||
			envelope["corpus_revision"] != "voice-pilot-en-v1" {
			continue
		}
		acceptance, _ := envelope["acceptance"].(map[string]any)
		lane, _ := envelope["execution_lane"].(string)
		if lane == "" {
			if details, ok := envelope["details"].(map[string]any); ok {
				lane, _ = details["execution_lane"].(string)
			}
		}
		if lane == "" {
			lane = "owner-unspecified"
		}
		for _, row := range compositeRequiredRows {
			claim, ok := acceptance[row].(map[string]any)
			if !ok || claim["in_band"] != true || claim["negative_controls_passed"] != true {
				continue
			}
			if lane == "simulated" && (claim["target_id"] != row || claim["target_revision"] != acceptanceTargetRevision || claim["method"] != "audio-tools-acceptance-v2" || claim["receipt_reference"] != item.GetArtifactRef() || claim["execution_lane"] != "simulated" || claim["simulated_as_native"] != false) {
				continue
			}
			if lane == "simulated" && !simulatedClaimChecksMatch(row, claim["checks"]) {
				continue
			}
			if row == "host-portability" && !hostPortabilityMatrixComplete(envelope) {
				continue
			}
			current, exists := selected[row]
			if !exists || at.After(current.ObservedAt) {
				selected[row] = compositeSource{ArtifactRef: item.GetArtifactRef(), ObservedAt: at, Lane: lane, Claim: claim}
			}
		}
	}
	missing := make([]string, 0)
	for _, row := range compositeRequiredRows {
		if _, ok := selected[row]; !ok {
			missing = append(missing, row)
		}
	}
	return selected, missing
}

func simulatedClaimChecksMatch(row string, raw any) bool {
	want := map[string][]string{
		"explicit-routing":        {"route_local_zero_debit", "route_byok_zero_debit", "route_owned_requires_consent"},
		"owned-settlement":        {"settlement_reserve_and_partial_finalize", "settlement_retry_is_idempotent", "settlement_local_byok_zero_debit"},
		"private-lifecycle":       {"lifecycle_authorized_recoverable_metadata_only", "lifecycle_delete_removes_retained_item"},
		"acceptance-integrity":    {"acceptance_rejects_missing_target", "acceptance_rejects_failed_artifact", "acceptance_rejects_simulated_as_native"},
		"adapter-maintainability": {"adapter_versioned_route_metadata", "adapter_metadata_only"},
	}[row]
	gotRaw, ok := raw.([]any)
	if !ok || len(want) == 0 || len(gotRaw) != len(want) {
		return false
	}
	got := make([]string, len(gotRaw))
	for i, value := range gotRaw {
		got[i], ok = value.(string)
		if !ok {
			return false
		}
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// hostPortabilityMatrixComplete prevents a scalar acceptance claim from
// hiding an omitted or under-evidenced required host cell. D3-D6 are
// explicitly outside the required matrix and must remain unclaimed.
func hostPortabilityMatrixComplete(envelope map[string]any) bool {
	details, ok := envelope["details"].(map[string]any)
	if !ok {
		return false
	}
	cpu, ok := details["d1_cpu"].(map[string]any)
	if !ok || !hostCellEvidenceComplete(cpu, hostCellExpectation{
		profile: "D1-linux-amd64-cpu", state: "available", inference: "passed",
		engineID: "whisper-local", modelID: "whisper-small", route: "whisper.cpp-cpu",
		accelerator: "none",
	}) {
		return false
	}
	if !d1AcceleratorCellComplete(details) {
		return false
	}
	d2, ok := details["d2"].(map[string]any)
	return ok && hostCellEvidenceComplete(d2, hostCellExpectation{
		profile: "D2-linux-amd64-constrained", state: "available", inference: "passed",
		engineID: "sherpa-streaming", modelID: "sherpa-onnx-streaming-zipformer-en-2023-06-26",
		route: "sherpa-streaming-zipformer-en-2023-06-26", accelerator: "none",
		resourceCPUs: 2, resourceMemory: 4 * 1024 * 1024 * 1024,
	})
}

type hostCellExpectation struct {
	profile, state, inference, engineID, modelID, route, accelerator string
	resourceCPUs, resourceMemory                                     int64
}

// hostCellEvidenceComplete validates the host/browser observation and its
// matching product-path receipt as one cell. A passing inference smoke alone
// cannot qualify a cell because it does not prove the user-facing route.
func hostCellEvidenceComplete(cell map[string]any, want hostCellExpectation) bool {
	if cell["profile_id"] != want.profile || cell["state"] != want.state || cell["inference"] != want.inference ||
		cell["engine_id"] != want.engineID || cell["model_id"] != want.modelID || cell["selected_route"] != want.route ||
		cell["os"] != "linux" || !nonblankEvidence(cell["os_version"]) || cell["architecture"] != "amd64" ||
		cell["browser"] != "Chromium" || !nonblankEvidence(cell["browser_version"]) ||
		!concreteEvidenceText(cell["device_identity"]) ||
		!nonblankEvidence(cell["engine_version"]) || !nonblankEvidence(cell["model_revision"]) ||
		positiveEvidenceNumber(cell["cpu_count"]) == 0 ||
		positiveEvidenceNumber(cell["memory_total_bytes"]) == 0 {
		return false
	}
	accelerator := cell["accelerator_identity"]
	if want.accelerator == "assigned" {
		if !concreteEvidenceText(accelerator) || accelerator == "none" || accelerator == "cpu-only" {
			return false
		}
	} else if accelerator != want.accelerator {
		return false
	}
	if want.resourceCPUs > 0 && positiveEvidenceNumber(cell["resource_cpu_limit"]) != want.resourceCPUs {
		return false
	}
	if want.resourceMemory > 0 && positiveEvidenceNumber(cell["resource_memory_limit_bytes"]) != want.resourceMemory {
		return false
	}
	artifact, _ := cell["artifact_ref"].(string)
	artifactHash, _ := cell["artifact_sha256"].(string)
	if strings.TrimSpace(artifact) == "" || !matrixSHA256Pattern.MatchString(artifactHash) ||
		!nonblankEvidence(cell["product_path"]) || cell["product_path_status"] != "passed" {
		return false
	}
	product, ok := cell["product_path_receipt"].(map[string]any)
	if !ok || product["outcome"] != "passed" || product["engine_id"] != want.engineID ||
		product["model_id"] != want.modelID || product["selected_route"] != want.route ||
		product["artifact_ref"] != artifact || product["artifact_sha256"] != artifactHash ||
		!concreteEvidenceText(product["visible_transcript_ref"]) {
		return false
	}
	return true
}

// d1AcceleratorCellComplete accepts either a qualified Kyutai inference or a
// properly evidenced explicit-unavailable cell. The fallback must have run
// through the product path and be tied to the same host/browser artifact;
// inventory availability by itself is never evidence of a usable fallback.
func d1AcceleratorCellComplete(details map[string]any) bool {
	cell, ok := details["d1_accelerator"].(map[string]any)
	if !ok {
		return false
	}
	if cell["engine_id"] == "kyutai" && cell["model_id"] == "kyutai/stt-1b-en_fr" && cell["state"] == "available" && cell["inference"] == "passed" {
		return hostCellEvidenceComplete(cell, hostCellExpectation{
			profile: "D1-linux-amd64-accelerator", state: "available", inference: "passed",
			engineID: "kyutai", modelID: "kyutai/stt-1b-en_fr", route: "kyutai", accelerator: "assigned",
		})
	}
	if cell["engine_id"] != "kyutai" || cell["model_id"] != "kyutai/stt-1b-en_fr" || cell["state"] != "unavailable" || cell["inference"] != "not_run" || !nonblankEvidence(cell["reason"]) {
		return false
	}
	if !concreteEvidenceText(cell["engine_version"]) || !concreteEvidenceText(cell["model_revision"]) {
		return false
	}
	path, ok := cell["fallback_product_path"].(map[string]any)
	if !ok || path["outcome"] != "passed" || path["engine_id"] != "sherpa-streaming" || path["model_id"] != "sherpa-onnx-streaming-zipformer-en-2023-06-26" || path["selected_route"] != "sherpa-streaming-zipformer-en-2023-06-26" || path["visible_fallback"] != true || !nonblankEvidence(path["visible_transcript_ref"]) {
		return false
	}
	if cell["artifact_ref"] != path["artifact_ref"] || cell["artifact_sha256"] != path["artifact_sha256"] ||
		cell["accelerator_identity"] != "none" {
		return false
	}
	fallbackCell := make(map[string]any, len(cell)+5)
	for key, value := range cell {
		fallbackCell[key] = value
	}
	fallbackCell["engine_id"] = "sherpa-streaming"
	fallbackCell["model_id"] = "sherpa-onnx-streaming-zipformer-en-2023-06-26"
	fallbackCell["selected_route"] = "sherpa-streaming-zipformer-en-2023-06-26"
	fallbackCell["state"] = "available"
	fallbackCell["inference"] = "passed"
	fallbackCell["product_path_receipt"] = path
	fallbackCell["product_path"] = "visible Sherpa fallback in Dictation Studio"
	fallbackCell["product_path_status"] = "passed"
	fallbackCell["model_revision"] = path["model_revision"]
	fallbackCell["engine_version"] = path["engine_version"]
	return hostCellEvidenceComplete(fallbackCell, hostCellExpectation{
		profile: "D1-linux-amd64-accelerator", state: "available", inference: "passed",
		engineID: "sherpa-streaming", modelID: "sherpa-onnx-streaming-zipformer-en-2023-06-26",
		route: "sherpa-streaming-zipformer-en-2023-06-26", accelerator: "none",
	})
}

func positiveEvidenceNumber(value any) int64 {
	switch number := value.(type) {
	case float64:
		if number > 0 {
			return int64(number)
		}
	case int:
		if number > 0 {
			return int64(number)
		}
	case int64:
		if number > 0 {
			return number
		}
	case uint64:
		if number > 0 && number <= uint64(^uint64(0)>>1) {
			return int64(number)
		}
	}
	return 0
}

func nonblankEvidence(value any) bool {
	s, ok := value.(string)
	return ok && strings.TrimSpace(s) != ""
}

func concreteEvidenceText(value any) bool {
	s, ok := value.(string)
	if !ok {
		return false
	}
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "", "unknown", "unavailable", "none", "unassigned", "placeholder":
		return false
	default:
		return true
	}
}
