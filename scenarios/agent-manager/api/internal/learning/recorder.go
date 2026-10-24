// Package learning adapts Agent Manager's completed investigation result to
// Vrooli Memory's outcome-linked Attempt contract. Memory is a delivery sink;
// Agent Manager remains authoritative for the diagnosis and its evidence.
package learning

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/investigation"
	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	memoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/learning"
	memoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/learning/learningv1connect"
)

const scope = "agent-manager-usage"

type Recorder interface {
	RecordInvestigation(context.Context, *investigation.Lifecycle) error
}

type Client struct {
	resolve func(context.Context) (string, error)
	http    *http.Client
}

func NewClient() *Client {
	return &Client{
		resolve: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "vrooli-memory")
		},
		http: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) RecordInvestigation(ctx context.Context, item *investigation.Lifecycle) error {
	if item == nil || item.Result == nil {
		return fmt.Errorf("completed investigation result is required")
	}
	base, err := c.resolve(ctx)
	if err != nil {
		return fmt.Errorf("discover vrooli-memory: %w", err)
	}
	attempt, started, finished := investigation.StableLearningAttempt(item)
	if attempt.AttemptID == "" {
		return fmt.Errorf("stable learning attempt identity is required")
	}
	provenance := strings.TrimSpace(item.Request.Provenance.Kind)
	if provenance == "" {
		provenance = "operator"
	}
	if provenance == "validation" {
		provenance = "test"
	}
	attemptProto := &memoryv1.Attempt{
		AttemptId:     attempt.AttemptID,
		TaskId:        item.Request.Subject.Ref,
		Operation:     "agent-investigation",
		ContextKey:    item.Request.Subject.Owner + "/" + item.Request.Subject.Kind,
		StartedAt:     started.Format(time.RFC3339Nano),
		FinishedAt:    finished.Format(time.RFC3339Nano),
		Outcome:       "unknown",
		EvidenceRefs:  investigationEvidenceRefs(item),
		RecallStatus:  "unavailable",
		Provenance:    provenance,
		Trigger:       strings.TrimSpace(item.Request.Provenance.TriggerOccurrenceRef),
		Approach:      strings.TrimSpace(item.Request.Question),
		TaskStartedAt: started.Format(time.RFC3339Nano),
		AttemptNumber: 1,
	}
	if attemptProto.Trigger == "" {
		attemptProto.Trigger = "finite investigation"
	}
	client := memoryconnect.NewLearningServiceClient(c.http, strings.TrimRight(base, "/"))
	_, err = client.RecordAttempt(ctx, connect.NewRequest(&memoryv1.RecordAttemptRequest{Scope: scope, Attempt: attemptProto}))
	if err != nil {
		return fmt.Errorf("record investigation learning attempt: %w", err)
	}
	return nil
}

func investigationEvidenceRefs(item *investigation.Lifecycle) []string {
	refs := make([]string, 0, len(item.Result.EvidenceRefs)+len(item.Result.Coverage))
	for _, ref := range item.Result.EvidenceRefs {
		refs = append(refs, ref.Owner+":"+ref.Kind+":"+ref.Ref)
	}
	for _, plane := range item.Result.Coverage {
		if plane.ThroughRef != "" {
			refs = append(refs, "coverage:"+plane.Plane+":"+plane.ThroughRef)
		}
	}
	return refs
}
