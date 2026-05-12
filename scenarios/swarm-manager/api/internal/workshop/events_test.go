package operatingmode

import "testing"

func TestPhasePayloadIncludesRequiredMetadata(t *testing.T) {
	round := eventPayloadTestRound()
	round.Payload = map[string]any{
		payloadFinishedAt:   "2026-04-30T12:05:30Z",
		payloadReplanNeeded: true,
		payloadVerdict:      "accepted",
	}

	payload := phasePayload(round, "completed", "")

	if payload.Mode != string(ModeHolisticLoop) ||
		payload.ScopeKind != "initiative" ||
		payload.ScopeID != "init-a" ||
		payload.InitiativeName != "init-a" ||
		payload.Phase != "review" ||
		payload.RunStrategy != string(RunStrategyOperatorGatedLoop) ||
		payload.AgentProfileKey != "swarm-manager/deep-work" ||
		payload.RoundNumber != 4 ||
		payload.RunID != "run-4" ||
		payload.Status != "completed" {
		t.Fatalf("phase payload missing required metadata: %+v", payload)
	}
	if payload.Verdict != "accepted" || !payload.ReplanNeeded {
		t.Fatalf("phase payload verdict/replan = %q/%v", payload.Verdict, payload.ReplanNeeded)
	}
	if payload.DurationSeconds != 30 {
		t.Fatalf("duration = %v, want 30", payload.DurationSeconds)
	}
	if len(payload.ArtifactPaths) != 1 || payload.ArtifactPaths[0] != "modes/holistic-loop/review.md" {
		t.Fatalf("artifact paths = %+v", payload.ArtifactPaths)
	}
}

func TestBacklogSyncPayloadIncludesRequiredMetadataAndSource(t *testing.T) {
	round := eventPayloadTestRound()
	source := BacklogMutationSource{
		Entrypoint:     "initiative.operating_mode.backlog_sync",
		InitiativeName: "init-a",
		Mode:           string(ModeHolisticLoop),
		Phase:          "review",
		Round:          4,
		RunID:          "run-4",
		RequestedBy:    "tester",
	}

	payload := backlogSyncPayload(round, 1, 2, 3, source, []string{"execute/do-thing"})

	if payload.Mode != string(ModeHolisticLoop) ||
		payload.ScopeKind != "initiative" ||
		payload.ScopeID != "init-a" ||
		payload.InitiativeName != "init-a" ||
		payload.Phase != "review" ||
		payload.RunStrategy != string(RunStrategyOperatorGatedLoop) ||
		payload.AgentProfileKey != "swarm-manager/deep-work" ||
		payload.RoundNumber != 4 ||
		payload.RunID != "run-4" ||
		payload.Status != string(RoundStatusCompleted) {
		t.Fatalf("backlog sync payload missing required metadata: %+v", payload)
	}
	if payload.BacklogItemsCompleted != 1 || payload.BacklogItemsCreated != 2 || payload.BacklogItemsUpdated != 3 {
		t.Fatalf("backlog sync counts = %d/%d/%d", payload.BacklogItemsCompleted, payload.BacklogItemsCreated, payload.BacklogItemsUpdated)
	}
	if payload.Source == nil ||
		payload.Source.Entrypoint != source.Entrypoint ||
		payload.Source.InitiativeName != "init-a" ||
		payload.Source.Mode != string(ModeHolisticLoop) ||
		payload.Source.Phase != "review" ||
		payload.Source.Round != 4 ||
		payload.Source.RunID != "run-4" ||
		payload.Source.RequestedBy != "tester" {
		t.Fatalf("source = %+v", payload.Source)
	}
	if len(payload.ItemRefs) != 1 || payload.ItemRefs[0] != "execute/do-thing" {
		t.Fatalf("item refs = %+v", payload.ItemRefs)
	}
	if len(payload.ArtifactPaths) != 1 || payload.ArtifactPaths[0] != "modes/holistic-loop/review.md" {
		t.Fatalf("artifact paths = %+v", payload.ArtifactPaths)
	}
}

func eventPayloadTestRound() RoundEnvelope {
	return RoundEnvelope{
		Round:           4,
		Mode:            string(ModeHolisticLoop),
		ScopeKind:       "initiative",
		ScopeID:         "init-a",
		InitiativeName:  "init-a",
		Phase:           "review",
		RunStrategy:     string(RunStrategyOperatorGatedLoop),
		AgentProfileKey: "swarm-manager/deep-work",
		GeneratedAt:     "2026-04-30T12:05:00Z",
		RunID:           "run-4",
		Status:          RoundStatusCompleted,
		ArtifactUpdates: []ArtifactUpdate{{Path: "modes/holistic-loop/review.md"}},
	}
}
