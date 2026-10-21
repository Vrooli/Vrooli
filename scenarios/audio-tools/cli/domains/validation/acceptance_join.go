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

const compositeEvidenceMaxAge = 24 * time.Hour

var compositeRequiredRows = []string{
	"dictation-trust", "provider-parity", "speaker-policy", "provider-evidence",
	"mobile-recovery", "interactive-latency", "corpus-quality", "owned-settlement",
	"explicit-routing", "host-portability", "private-lifecycle", "acceptance-integrity",
	"adapter-maintainability",
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
	resp, err := client.ListQualificationEvidence(context.Background(), connect.NewRequest(&experimentv1.ListQualificationEvidenceRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list owner acceptance evidence", err, nil)
	}
	now := h.now().UTC()
	sources, missing := selectCompositeSources(resp.Msg.GetEvidence(), now)
	if len(missing) > 0 {
		return fmt.Errorf("acceptance join missing fresh passing owner rows: %s", strings.Join(missing, ", "))
	}

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
		"target_revision":     "portable-voice-v1",
		"method":              "audio-tools-acceptance-v1",
		"cohort":              "voice-pilot-en-v1",
		"build":               build,
		"device_profile":      "D1+SIMULATED",
		"policy_revision":     "policy-v1",
		"corpus_revision":     "voice-pilot-en-v1",
		"execution_lane":      "composite",
		"simulated_as_native": false,
		"provider_cell": map[string]string{
			"engine_id": "acceptance-join",
			"model_id":  "portable-voice-v1",
			"strategy":  "composite",
		},
		"acceptance": claims,
		"details": map[string]any{
			"source_artifacts":      refs,
			"claim_execution_lanes": lanes,
			"source_count":          len(sources),
			"max_source_age":        compositeEvidenceMaxAge.String(),
		},
	})
	if err != nil {
		return fmt.Errorf("encode acceptance join: %w", err)
	}
	artifact := "audio-tools:acceptance-join:" + now.Format("20060102T150405.000000000Z") + ":" + hex.EncodeToString(digest[:8])
	_, err = client.RecordQualificationEvidence(context.Background(), connect.NewRequest(&experimentv1.RecordQualificationEvidenceRequest{Evidence: &experimentv1.QualificationEvidence{
		EngineId: "acceptance-join", ModelId: "portable-voice-v1", Strategy: "composite", PolicyProfile: "policy-v1",
		Kind:   experimentv1.QualificationEvidenceKind_QUALIFICATION_EVIDENCE_KIND_INTERVAL_ACCOUNTING,
		Passed: true, ArtifactRef: artifact,
		Notes:       "owner-validated composite acceptance join; per-row source artifacts and simulated/native lanes retained in machine_json",
		MachineJson: string(machine), ObservedAt: timestamppb.New(now),
	}}))
	if err != nil {
		return cliapp.WrapAPIError("record acceptance join", err, nil)
	}
	if ctx.JSON() {
		return json.NewEncoder(ctx.Stdout()).Encode(map[string]any{"passed": true, "artifact_ref": artifact, "source_count": len(sources), "build": build})
	}
	fmt.Fprintf(ctx.Stdout(), "acceptance join passed: %s (%d fresh owner sources)\n", artifact, len(sources))
	return nil
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
			envelope["target_revision"] != "portable-voice-v1" ||
			envelope["method"] != "audio-tools-acceptance-v1" ||
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
