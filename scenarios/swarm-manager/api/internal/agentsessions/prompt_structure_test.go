package agentsessions

import (
	"context"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"
)

// The session prompt exists to be cached. Two sessions of the same kind must
// share every stable band byte-for-byte, and the shared prefix must be long
// enough to be worth caching. The defect these tests guard against: the session
// ID was emitted third, above every stable instruction, so no two sessions
// shared a prefix beyond roughly forty bytes.

func promptForSession(id string, kind Kind, message string) string {
	return buildInitialPrompt(
		Session{ID: id, Kind: kind, SkillID: skillIDForKind(kind)},
		Message{Content: message},
		nil,
	)
}

func sharedPrefixLen(a, b string) int {
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	for i := 0; i < limit; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return limit
}

func TestInitialPromptSharesStablePrefixAcrossSessionsOfOneKind(t *testing.T) {
	for _, kind := range []Kind{KindMetaOrchestration, KindSwarmOperations, KindWorkflowAuthoring} {
		t.Run(string(kind), func(t *testing.T) {
			first := promptForSession("sess_aaaaaaaaaaaaaaaa", kind, "What should I do?")
			second := promptForSession("sess_bbbbbbbbbbbbbbbb", kind, "Something else entirely.")

			shared := sharedPrefixLen(first, second)
			// The universal doctrine plus the kind band are both stable, so the
			// shared prefix must cover them. A regression that promotes any
			// volatile value above them shows up here as a collapsed prefix.
			wantAtLeast := len(sessionDoctrine)
			if shared < wantAtLeast {
				t.Fatalf("shared prefix = %d bytes, want >= %d (the universal doctrine must be identical and first)\n--- first ---\n%s", shared, wantAtLeast, first)
			}
			if !strings.Contains(first[:shared], subjectForKind(kind)) {
				t.Fatalf("kind band fell outside the shared prefix for %s; shared prefix:\n%s", kind, first[:shared])
			}
		})
	}
}

func TestInitialPromptEmitsVolatileIdentityBelowStableBands(t *testing.T) {
	prompt := promptForSession("sess_cafebabecafebabe", KindMetaOrchestration, "Plan this.")

	doctrineAt := strings.Index(prompt, "<session-doctrine>")
	kindAt := strings.Index(prompt, "<session-kind")
	identityAt := strings.Index(prompt, "<session-identity>")
	messageAt := strings.Index(prompt, "<operator-message>")

	if doctrineAt < 0 || kindAt < 0 || identityAt < 0 || messageAt < 0 {
		t.Fatalf("prompt is missing a required section:\n%s", prompt)
	}
	if !(doctrineAt < kindAt && kindAt < identityAt && identityAt < messageAt) {
		t.Fatalf("sections are out of volatility order (doctrine=%d kind=%d identity=%d message=%d):\n%s", doctrineAt, kindAt, identityAt, messageAt, prompt)
	}
	if strings.Index(prompt, "sess_cafebabecafebabe") < doctrineAt {
		t.Fatalf("the session ID appeared above the universal doctrine, which collapses the cacheable prefix:\n%s", prompt)
	}
}

func TestInitialPromptKeepsTheOperatorMessageOutsideTheContextBlock(t *testing.T) {
	prompt := promptForSession("sess_0123456789abcdef", KindSwarmOperations, "Which item matters most?")

	closeAt := strings.Index(prompt, "</context>")
	messageAt := strings.Index(prompt, "<operator-message>")
	if closeAt < 0 || messageAt < 0 {
		t.Fatalf("prompt is missing the context block or the operator message:\n%s", prompt)
	}
	if closeAt > messageAt {
		t.Fatalf("the operator message must sit outside the context block:\n%s", prompt)
	}
}

func TestEverySessionPromptSectionIsRegistered(t *testing.T) {
	// A section the registry does not name is an untracked block that tests and
	// clients cannot interpret. newPromptSection panics on an unregistered kind;
	// this asserts the registry stays complete for the kinds the builder emits.
	emitted := []string{
		promptSectionKindDoctrine,
		promptSectionKindSubject,
		promptSectionKindProposalTarget,
		promptSectionKindIdentity,
		promptSectionKindStartupBrief,
		promptSectionKindContext,
		promptSectionKindImages,
		promptSectionKindOperatorMsg,
	}
	for _, kind := range emitted {
		spec, ok := promptSectionSpecs[kind]
		if !ok {
			t.Fatalf("section kind %q is emitted by the builder but not registered", kind)
		}
		if strings.TrimSpace(spec.Element) == "" {
			t.Fatalf("section kind %q has no XML element name", kind)
		}
	}
	if len(promptSectionSpecs) != len(emitted) {
		t.Fatalf("registry has %d entries but the builder emits %d; an unemitted section is dead weight", len(promptSectionSpecs), len(emitted))
	}
}

func TestContinuationPromptStaysPlainWhenNothingIsAttached(t *testing.T) {
	// A follow-up with no context is the common case. Wrapping it in XML would
	// add tokens and break the prefix the conversation already established.
	got := buildContinuationPrompt(Message{Content: "  Keep going.  "}, nil)
	if got != "Keep going." {
		t.Fatalf("continuation prompt = %q, want the trimmed message alone", got)
	}
}

// The preview exists so the operator can judge the prompt before sending it.
// If it were assembled anywhere but the real builders it would drift, and a
// preview that disagrees with what is sent is worse than no preview.
func TestPreviewPromptMatchesWhatStartWouldSend(t *testing.T) {
	spawner := &fakeSessionSpawner{runState: agentmanager.RunState{Status: "running"}}
	svc := newTestService(t, spawner)
	ctx := context.Background()

	draft, err := svc.Create(ctx, CreateRequest{Kind: KindMetaOrchestration, Title: "Plan quality gates"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	const message = "Turn this into goals."
	preview, err := svc.PreviewPrompt(ctx, ContinueRequest{SessionID: draft.ID, Message: message})
	if err != nil {
		t.Fatalf("PreviewPrompt() error = %v", err)
	}
	if !preview.Initial {
		t.Error("a draft session must preview the initial-prompt builder")
	}

	if _, err := svc.Start(ctx, ContinueRequest{SessionID: draft.ID, Message: message}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if preview.Prompt != spawner.spawnReq.Prompt {
		t.Fatalf("preview did not match the spawned prompt\n--- preview ---\n%s\n--- sent ---\n%s", preview.Prompt, spawner.spawnReq.Prompt)
	}
}

func TestPreviewPromptDoesNotMutateTheSession(t *testing.T) {
	spawner := &fakeSessionSpawner{runState: agentmanager.RunState{Status: "running"}}
	svc := newTestService(t, spawner)
	ctx := context.Background()

	draft, err := svc.Create(ctx, CreateRequest{Kind: KindSwarmOperations, Title: "Check status"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := svc.PreviewPrompt(ctx, ContinueRequest{SessionID: draft.ID, Message: "What matters most?"}); err != nil {
		t.Fatalf("PreviewPrompt() error = %v", err)
	}

	after, err := svc.Get(ctx, draft.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(after.Messages) != 0 {
		t.Errorf("preview appended %d message(s); it must be read-only", len(after.Messages))
	}
	if after.Status != StatusDraft {
		t.Errorf("preview moved the session to %q; it must stay draft", after.Status)
	}
	if spawner.spawnCalls != 0 {
		t.Errorf("preview spawned %d run(s); it must not spawn", spawner.spawnCalls)
	}
}
