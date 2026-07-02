package authoring

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	planmodel "plan-manager/internal/planmodel"
	"plan-manager/internal/probes"
)

func TestCurateContextBatchPartitionsDeterministically(t *testing.T) {
	candidates := []ContextCandidate{
		candidateForCuration("topic-filler", planmodel.RelevantContextSkill, "topic-only", "", 0.5508, "topic", 1),
		candidateForCuration("strong-skill", planmodel.RelevantContextSkill, "implementation-plan-authoring", "", 0.694, "search", 1),
		candidateForCuration("weak-search", planmodel.RelevantContextDoc, "docs/weak.md", "", 0.42, "search", 1),
		candidateForCuration("corroborated-topic", planmodel.RelevantContextSkill, "ecosystem-fit", "", 0.55, "topic", 2),
		candidateForCuration("record", planmodel.RelevantContextCommand, "", "swarm-manager records get --id rec-123", 0.81, "swarm-manager.records", 1),
	}

	curated, stats := curateContextBatch(candidates)

	byID := map[string]ContextCandidate{}
	for _, candidate := range curated {
		byID[candidate.ID] = candidate
	}
	require.Equal(t, contextTierLonglist, byID["topic-filler"].Tier)
	require.False(t, byID["topic-filler"].HighConfidence)
	require.Equal(t, contextTierShortlist, byID["strong-skill"].Tier)
	require.Equal(t, contextTierLonglist, byID["weak-search"].Tier)
	require.Equal(t, contextTierShortlist, byID["corroborated-topic"].Tier)
	require.True(t, byID["corroborated-topic"].HighConfidence)
	require.Equal(t, contextTierShortlist, byID["record"].Tier)
	require.True(t, byID["record"].HighConfidence)
	require.Equal(t, 1, stats.OmittedTopicFiller)
	require.Equal(t, 1, stats.OmittedBelowThreshold)
}

func TestCurateContextBatchAppliesKindCaps(t *testing.T) {
	var candidates []ContextCandidate
	for i, id := range []string{"a", "b", "c", "d"} {
		candidates = append(candidates, candidateForCuration(id, planmodel.RelevantContextSkill, id, "", 0.9-float64(i)*0.01, "search", 1))
	}

	curated, stats := curateContextBatch(candidates)

	shortlisted := 0
	for _, candidate := range curated {
		if candidate.Tier == contextTierShortlist {
			shortlisted++
		}
	}
	require.Equal(t, contextSkillShortlistCap, shortlisted)
	require.Equal(t, 1, stats.OmittedByCap)
}

func TestContextProposalNoDowngradeFromProbeFixtures(t *testing.T) {
	runner := func(_ context.Context, name string, args ...string) ([]byte, error) {
		argv := name + " " + strings.Join(args, " ")
		switch {
		case name == "prompt-manager" && strings.Contains(argv, "--type skill"):
			return os.ReadFile(filepath.Join("..", "probes", "testdata", "prompt-manager-discover-skill.json"))
		case name == "prompt-manager" && strings.Contains(argv, "--type all"):
			return os.ReadFile(filepath.Join("..", "probes", "testdata", "prompt-manager-discover-all.json"))
		case name == "search-hub":
			return os.ReadFile(filepath.Join("..", "probes", "testdata", "search-hub-query.json"))
		default:
			return nil, errors.New("unexpected probe " + argv)
		}
	}

	outcomes := probes.Discover(context.Background(), runner, []string{"plan authoring"}, "architectural", probes.Options{})
	result := candidatesFromProbeOutcomes(outcomes)
	sess := Session{Title: "Plan authoring"}
	batch := mergeContextDiscoveryBatch(&sess, []string{"plan authoring"}, "architectural", result)
	shortlist := contextCandidatesForBatch(sess.ContextCandidates, batch.ID)

	require.NotEmpty(t, shortlist)
	seenKinds := map[string]bool{}
	for _, candidate := range shortlist {
		seenKinds[string(candidate.Item.Kind)] = true
		require.Equal(t, batch.ID, candidate.BatchID)
		require.Equal(t, contextTierShortlist, candidate.Tier)
		require.NotEmpty(t, candidate.Handle)
		require.True(t, strings.HasPrefix(candidate.Handle, "c"), "shortlist handle = %q", candidate.Handle)
		require.NotZero(t, candidate.Score, "score must survive to proposal candidate")
		require.NotEmpty(t, candidate.Origin, "origin must survive to proposal candidate")
		require.NotEmpty(t, candidate.Title, "title must survive to proposal candidate")
		require.NotEmpty(t, candidate.Snippet, "snippet must survive to proposal candidate")
		require.NotEmpty(t, candidate.Corroboration, "corroboration must survive to proposal candidate")
		require.NotEmpty(t, candidate.Corroboration[0].Concept, "corroboration concept must survive")
		require.NotEmpty(t, candidate.SetupLine, "final-form setup line must be projected")
		if candidate.SizeChars > 0 {
			require.NotEmpty(t, candidate.Tags, "sized prompt-manager candidates must keep tags")
		}
	}

	require.True(t, seenKinds["skill"], "fixture shortlist must include a skill candidate")
	require.True(t, seenKinds["doc"], "fixture shortlist must include a doc candidate")
	require.True(t, seenKinds["command"], "fixture shortlist must include a command candidate")
}

func candidateForCuration(id string, kind planmodel.RelevantContextKind, target string, command string, score float64, origin string, hits int) ContextCandidate {
	candidate := ContextCandidate{
		ID:     id,
		Score:  score,
		Origin: origin,
		Item: planmodel.RelevantContextItem{
			Kind:    kind,
			Label:   id,
			Target:  target,
			Command: command,
		},
	}
	for i := 0; i < hits; i++ {
		candidate.Corroboration = append(candidate.Corroboration, ProbeHit{
			Probe:   "probe",
			Concept: id + string(rune('a'+i)),
			Score:   score,
		})
	}
	return candidate
}
