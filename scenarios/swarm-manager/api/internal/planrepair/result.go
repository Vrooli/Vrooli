// Package planrepair defines the typed terminal payload and Plan Manager
// candidate hand-off for the declared plan.repair transition. Workflow
// lifecycle and correlation durability are owned by transitionrunner.
package planrepair

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"swarm-manager/internal/planclient"

	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"

	"google.golang.org/protobuf/encoding/protojson"
)

type TerminalResult struct {
	Outcome       string          `json:"outcome"`
	CandidatePlan json.RawMessage `json:"candidatePlan,omitempty"`
	Summary       string          `json:"summary,omitempty"`
	Reason        string          `json:"reason,omitempty"`
	Unresolved    []string        `json:"unresolvedFindings,omitempty"`
}

type CandidateCanonicalizer interface{ planclient.CandidateClient }

// Canonicalize persists a whole-plan candidate without changing the canonical
// plan reference. The caller supplies values from its current guarded subject
// snapshot; transitionrunner owns the execution correlation itself.
func Canonicalize(ctx context.Context, client CandidateCanonicalizer, planReference, baseContentHash, executionID string, result TerminalResult) (*plansv1.CandidateRevisionPreview, error) {
	if client == nil {
		return nil, fmt.Errorf("plan manager client is required")
	}
	if result.Outcome != "ready" {
		return nil, fmt.Errorf("only ready repair results can be canonicalized")
	}
	if strings.TrimSpace(planReference) == "" || strings.TrimSpace(baseContentHash) == "" || strings.TrimSpace(executionID) == "" {
		return nil, fmt.Errorf("plan reference, base content hash, and execution id are required")
	}
	candidatePlan := &sharedv1.Plan{}
	if err := protojson.Unmarshal(result.CandidatePlan, candidatePlan); err != nil {
		return nil, fmt.Errorf("decode whole-plan repair candidate: %w", err)
	}
	candidate, err := client.CreateCandidateRevision(ctx, planclient.CandidateRevisionInput{
		PlanID: planReference, ExpectedBaseContentHash: baseContentHash,
		ProposalProvenance: "swarm-manager:repair/" + executionID, CandidatePlan: candidatePlan,
	})
	if err != nil {
		return nil, err
	}
	if candidate == nil || strings.TrimSpace(candidate.GetId()) == "" {
		return nil, fmt.Errorf("plan manager candidate creation omitted candidate id")
	}
	return client.PreviewCandidateRevision(ctx, candidate.GetId())
}
