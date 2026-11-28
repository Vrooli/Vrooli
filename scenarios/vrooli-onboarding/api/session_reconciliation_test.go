package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/operatorstate"
	sessiondomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/session"
)

type profileCatalogEnvironment map[string]string

func (e profileCatalogEnvironment) Getenv(key string) string { return e[key] }

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

func TestProfileSessionSurvivesCheckoutBundleCatalogLossAndUnsupportedProfile(t *testing.T) {
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	storageRoot := filepath.Join(t.TempDir(), "operator-storage")
	statePath := filepath.Join(storageRoot, "onboarding-state.json")
	previousPath, previousRoots, previousEnvironment := operatorStatePath, operatorStateRoots, rootEnvironment
	operatorStatePath = func() (string, error) { return statePath, nil }
	operatorStateRoots = nil
	rootEnvironment = profileCatalogEnvironment{"VROOLI_ROOT": repoRoot, "VROOLI_STORAGE_ROOT": storageRoot}
	t.Cleanup(func() {
		operatorStatePath, operatorStateRoots, rootEnvironment = previousPath, previousRoots, previousEnvironment
	})

	ctx := context.Background()
	initial, err := operatorStateService().Apply(ctx, []byte(`{}`))
	if err != nil {
		t.Fatalf("initialize operator state: %v", err)
	}
	revision := operatorstate.Revision(initial)
	saved, err := (&Server{}).saveProfileSession(ctx, sessiondomain.ProfileSession{
		Target: "local", Actor: "operator", Mode: "guided", ProfileID: "local-use", ProfileVersion: "1.0.0",
		CatalogRevision: "source-catalog-r1", BaseRevision: revision,
		Answers: map[string]json.RawMessage{"purposes": json.RawMessage(`["use-local-apps"]`)},
	}, revision)
	if err != nil {
		t.Fatalf("save profile work from checkout: %v", err)
	}
	if saved.Revision == "" || saved.ConsequenceDigest == "" {
		t.Fatalf("saved profile session lacks durable revision/evaluation: %+v", saved)
	}
	before, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read saved profile state: %v", err)
	}

	bundleRoot := filepath.Join(t.TempDir(), "bundle")
	rootEnvironment = profileCatalogEnvironment{"VROOLI_STORAGE_ROOT": storageRoot, "BUNDLE_ROOT": bundleRoot}
	unavailable, err := (&Server{}).getProfileSession(ctx, "local", "operator")
	if err != nil || unavailable == nil || unavailable.ReconciliationState != "unavailable" || unavailable.NextAction != "retry-profile-check" {
		t.Fatalf("missing bundle catalog session=%+v err=%v", unavailable, err)
	}
	if string(unavailable.Answers["purposes"]) != `["use-local-apps"]` {
		t.Fatalf("catalog loss changed saved answers: %#v", unavailable.Answers)
	}

	profileDir := filepath.Join(bundleRoot, "catalog", "scenarios", "vrooli-onboarding", "profiles")
	if err := os.MkdirAll(filepath.Join(bundleRoot, "catalog", "scenarios"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	generalProfile, err := os.ReadFile(filepath.Join(repoRoot, "scenarios", "vrooli-onboarding", "profiles", "general-purpose.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "general-purpose.json"), generalProfile, 0o600); err != nil {
		t.Fatal(err)
	}
	unsupported, err := (&Server{}).getProfileSession(ctx, "local", "operator")
	if err != nil || unsupported == nil || unsupported.ReconciliationState != "profile_revoked" || unsupported.NextAction != "choose-profile" {
		t.Fatalf("bundle without selected profile session=%+v err=%v", unsupported, err)
	}
	if len(unsupported.ReconciliationChanges) != 1 || unsupported.ReconciliationChanges[0].Kind != "profile-revoked" || string(unsupported.Answers["purposes"]) != `["use-local-apps"]` {
		t.Fatalf("unsupported bundle did not retain explicit profile and answers: %+v", unsupported)
	}

	localProfile, err := os.ReadFile(filepath.Join(repoRoot, "scenarios", "vrooli-onboarding", "profiles", "local-use.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "local-use.json"), localProfile, 0o600); err != nil {
		t.Fatal(err)
	}
	recovered, err := (&Server{}).getProfileSession(ctx, "local", "operator")
	if err != nil || recovered == nil || recovered.CurrentProfileVersion != "1.0.0" || recovered.ProfileID != "local-use" {
		t.Fatalf("restored bundle session=%+v err=%v", recovered, err)
	}
	if string(recovered.Answers["purposes"]) != `["use-local-apps"]` || recovered.ReconciliationState == "profile_revoked" || recovered.NextAction == "choose-profile" {
		t.Fatalf("restored profile lost retained work: %+v", recovered)
	}
	after, err := os.ReadFile(statePath)
	if err != nil || string(after) != string(before) {
		t.Fatalf("reconciliation wrote or lost saved work: err=%v", err)
	}
}
