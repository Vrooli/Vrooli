package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/operatorstate"
	sessiondomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/session"
)

func TestProfileSessionReconciliationHelpersAreStableAndRetainRemovedAnswers(t *testing.T) {
	answers := map[string]json.RawMessage{
		"removed-question": json.RawMessage(`"retained"`),
		"enabled":          json.RawMessage(`true`),
	}
	values := rawAnswerValues(answers)
	if values["removed-question"] != "retained" || values["enabled"] != true {
		t.Fatalf("raw answers = %#v", values)
	}
	changes := diffTargetContext(
		map[string]string{"environment": "staging", "region": "us-east"},
		map[string]string{"environment": "production", "region": "us-east", "account": "customer"},
	)
	if len(changes) != 2 || changes[0].Kind != "target-context" || changes[0].Field != "account" || changes[1].Field != "environment" {
		t.Fatalf("target context changes = %#v", changes)
	}
	change := sessiondomain.ReconciliationChange{Field: "removed-question", Before: compactRaw(answers["removed-question"])}
	if change.Before != `"retained"` {
		t.Fatalf("compact answer = %q", change.Before)
	}
}

func TestReconcileProfileSessionProducesDigestReviewAndNextQuestion(t *testing.T) {
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_ROOT", repoRoot)
	t.Setenv("BUNDLE_ROOT", "")
	previousPath := operatorStatePath
	statePath := filepath.Join(t.TempDir(), "operator-state.json")
	operatorStatePath = func() (string, error) { return statePath, nil }
	t.Cleanup(func() { operatorStatePath = previousPath })

	value := &sessiondomain.ProfileSession{
		Target: "local", Actor: "operator", Mode: "guided", ProfileID: "local-use", ProfileVersion: "0.9.0",
		ConsequenceDigest: "old-digest", Answers: map[string]json.RawMessage{
			"purposes":         json.RawMessage(`["use-local-apps"]`),
			"removed-question": json.RawMessage(`"retain-me"`),
		},
		TargetContext: map[string]string{"environment": "local"},
	}
	result := (&Server{}).reconcileProfileSession(context.Background(), value)
	if result.ConsequenceDigest == "" || result.ReconciliationState != "review_required" {
		t.Fatalf("reconciled session = %#v", result)
	}
	if result.NextAction != "review-profile" || len(result.ReconciliationChanges) < 3 {
		t.Fatalf("reconciliation metadata = %#v", result)
	}
	var removed, digest, version bool
	for _, change := range result.ReconciliationChanges {
		switch change.Kind {
		case "question-removed":
			removed = change.Field == "removed-question" && change.Before == `"retain-me"`
		case "consequence-digest":
			digest = change.Before == "old-digest" && change.After == result.ConsequenceDigest
		case "profile-version":
			version = change.Before == "0.9.0" && change.After == "1.0.0"
		}
	}
	if !removed || !digest || !version {
		t.Fatalf("missing reconciliation changes: %#v", result.ReconciliationChanges)
	}

	incomplete := &sessiondomain.ProfileSession{Target: "local", Actor: "operator", Mode: "guided", ProfileID: "local-use", ProfileVersion: "1.0.0"}
	incompleteResult := (&Server{}).reconcileProfileSession(context.Background(), incomplete)
	if incompleteResult.NextQuestionID != "purposes" || incompleteResult.NextAction != "answer-question" {
		t.Fatalf("next applicable question = %#v", incompleteResult)
	}
}

func TestSaveProfileSessionReturnsDurableUpdatedAt(t *testing.T) {
	previousPath, previousNow := operatorStatePath, operatorStateNow
	statePath := filepath.Join(t.TempDir(), "operator-state.json")
	fixedNow := time.Date(2026, 9, 11, 1, 2, 3, 4, time.UTC)
	operatorStatePath = func() (string, error) { return statePath, nil }
	operatorStateNow = func() time.Time { return fixedNow }
	t.Cleanup(func() { operatorStatePath, operatorStateNow = previousPath, previousNow })

	initial, err := operatorStateService().Apply(context.Background(), []byte(`{}`))
	if err != nil {
		t.Fatalf("initialize operator state: %v", err)
	}
	revision := operatorstate.Revision(initial)
	result, err := (&Server{}).saveProfileSession(context.Background(), sessiondomain.ProfileSession{
		Target: "local", Actor: "operator", Mode: "manual", BaseRevision: revision,
	}, revision)
	if err != nil {
		t.Fatalf("save profile session: %v", err)
	}
	want := fixedNow.Format(time.RFC3339Nano)
	if result.UpdatedAt != want {
		t.Fatalf("save response updated_at = %q, want durable timestamp %q", result.UpdatedAt, want)
	}
}
