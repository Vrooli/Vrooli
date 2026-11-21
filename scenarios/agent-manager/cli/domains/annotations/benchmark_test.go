package annotations

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ptr(v float64) *float64 { return &v }
func boolPtr(v bool) *bool   { return &v }

func TestScoreGoldUsesStoredProbabilitiesAndNeverInferences(t *testing.T) {
	rows := []benchmarkGoldRow{
		{RunID: "run-1", Positive: boolPtr(true)},
		{RunID: "run-2", Positive: boolPtr(true)},
		{RunID: "run-3", Positive: boolPtr(false)},
		{RunID: "run-4", Positive: boolPtr(true)}, // no usable annotation
	}
	views := map[string]runView{
		"run-1": {Labels: []annotationView{{ID: "agent.spinning", State: "observed", Probability: ptr(0.9)}}},
		"run-2": {Labels: []annotationView{{ID: "agent.spinning", State: "observed", Probability: ptr(0.2)}}},
		"run-3": {Labels: []annotationView{{ID: "agent.spinning", State: "observed", Probability: ptr(0.7)}}},
		"run-4": {Labels: []annotationView{{ID: "agent.spinning", State: "not_observed"}}},
	}
	result := scoreGold(rows, views, "agent.spinning", 0.6)
	if result.Gold != 3 || result.Labelled != 4 || result.TruePositive != 1 || result.FalseNegative != 1 || result.FalsePositive != 1 || result.TrueNegative != 0 {
		t.Fatalf("counts = %+v", result)
	}
	if result.Unavailable != 1 {
		t.Fatalf("unavailable = %d, want 1", result.Unavailable)
	}
	if result.Precision != 0.5 || result.Recall != 0.5 {
		t.Fatalf("precision/recall = %v/%v", result.Precision, result.Recall)
	}
}

func TestScoreGoldReadsNoulAndExcludesOtherLabels(t *testing.T) {
	rows := []benchmarkGoldRow{
		{RunID: "run-1", Labels: map[string]bool{"operator.frustration": true}},
		{RunID: "run-2", Labels: map[string]bool{"operator.frustration": false}},
		{RunID: "run-3", Labels: map[string]bool{"agent.spinning": true}},
	}
	views := map[string]runView{
		"run-1": {Labels: []annotationView{{ID: "operator.frustration", State: "observed", Value: value{Noul: ptr(0.8)}}}},
		"run-2": {Labels: []annotationView{{ID: "operator.frustration", State: "observed", Value: value{Noul: ptr(0.2)}}}},
		"run-3": {Labels: []annotationView{{ID: "operator.frustration", State: "unavailable"}}},
	}
	result := scoreGold(rows, views, "operator.frustration", 0.6)
	if result.Gold != 2 || result.Labelled != 2 || result.TruePositive != 1 || result.TrueNegative != 1 || result.Unavailable != 0 {
		t.Fatalf("result = %+v", result)
	}
	if result.Precision != 1 || result.Recall != 1 {
		t.Fatalf("precision/recall = %v/%v", result.Precision, result.Recall)
	}
}

func TestSingleGoldLabelRejectsAmbiguity(t *testing.T) {
	rows := []benchmarkGoldRow{
		{RunID: "run-1", Labels: map[string]bool{"agent.spinning": true}},
		{RunID: "run-2", Labels: map[string]bool{"operator.valence": true}},
	}
	if _, err := singleGoldLabel(rows); err == nil {
		t.Fatal("ambiguous labels were accepted")
	}
}

func TestScoreGoldAppliesSignedScorePolarity(t *testing.T) {
	ceiling := -1.0
	rows := []benchmarkGoldRow{
		{RunID: "negative", Positive: boolPtr(true)},
		{RunID: "praise", Positive: boolPtr(false)},
		{RunID: "unsure", Positive: boolPtr(true)},
	}
	views := map[string]runView{
		"negative": {Labels: []annotationView{{
			ID: "operator.valence", State: "observed", Probability: ptr(0.9),
			Value: value{Kind: "score", Score: ptr(-1.5)}, Friction: friction{Enabled: true, MaxScore: &ceiling},
		}}},
		"praise": {Labels: []annotationView{{
			ID: "operator.valence", State: "observed", Probability: ptr(0.9),
			Value: value{Kind: "score", Score: ptr(1.5)}, Friction: friction{Enabled: true, MaxScore: &ceiling},
		}}},
		"unsure": {Labels: []annotationView{{
			ID: "operator.valence", State: "observed", Probability: ptr(0.4),
			Value: value{Kind: "score", Score: ptr(-1.5)}, Friction: friction{Enabled: true, MaxScore: &ceiling},
		}}},
	}
	result := scoreGold(rows, views, "operator.valence", 0.6)
	if result.TruePositive != 1 || result.FalsePositive != 0 || result.FalseNegative != 1 || result.TrueNegative != 1 {
		t.Fatalf("signed-score counts = %+v, want praise excluded and low confidence a miss", result)
	}
}

func TestScoreGoldWithoutPolarityStillUsesConfidence(t *testing.T) {
	rows := []benchmarkGoldRow{{RunID: "run-1", Positive: boolPtr(true)}}
	views := map[string]runView{
		"run-1": {Labels: []annotationView{{ID: "agent.spinning", State: "observed", Probability: ptr(0.8)}}},
	}
	result := scoreGold(rows, views, "agent.spinning", 0.6)
	if result.TruePositive != 1 || result.Precision != 1 {
		t.Fatalf("probability-only scoring regressed: %+v", result)
	}
}

func TestScoreGoldBindsTheGateToASingleDefinitionRevision(t *testing.T) {
	const hash = "definition-hash-current"
	rows := []benchmarkGoldRow{
		{RunID: "run-1", Positive: boolPtr(true)},
		{RunID: "run-2", Positive: boolPtr(true)},
	}
	views := map[string]runView{
		"run-1": {Labels: []annotationView{{ID: "agent.spinning", State: "observed", Probability: ptr(0.9), DefinitionHash: hash}}},
		"run-2": {Labels: []annotationView{{ID: "agent.spinning", State: "observed", Probability: ptr(0.8), DefinitionHash: hash}}},
	}
	result := scoreGold(rows, views, "agent.spinning", 0.6)
	if result.Revisions != 1 || result.DefinitionHash != hash {
		t.Fatalf("revision identity = %d/%q", result.Revisions, result.DefinitionHash)
	}
	var out bytes.Buffer
	renderBenchmark(&out, result)
	if !strings.Contains(out.String(), `"definition_hash":"`+hash+`"`) {
		t.Fatalf("gate block lacks the definition hash: %s", out.String())
	}
}

func TestScoreGoldRefusesAGateAcrossDefinitionRevisions(t *testing.T) {
	rows := []benchmarkGoldRow{
		{RunID: "run-1", Positive: boolPtr(true)},
		{RunID: "run-2", Positive: boolPtr(true)},
	}
	views := map[string]runView{
		"run-1": {Labels: []annotationView{{ID: "agent.spinning", State: "observed", Probability: ptr(0.9), DefinitionHash: "old"}}},
		"run-2": {Labels: []annotationView{{ID: "agent.spinning", State: "observed", Probability: ptr(0.8), DefinitionHash: "new"}}},
	}
	result := scoreGold(rows, views, "agent.spinning", 0.6)
	if result.Revisions != 2 || result.DefinitionHash != "" {
		t.Fatalf("mixed revisions = %d/%q", result.Revisions, result.DefinitionHash)
	}
	var out bytes.Buffer
	renderBenchmark(&out, result)
	if strings.Contains(out.String(), "gate.measured") || !strings.Contains(out.String(), "span 2 definition revisions") {
		t.Fatalf("mixed-revision gate was not refused: %s", out.String())
	}
}

func TestScoreGoldAppliesTheHighIsBadDirection(t *testing.T) {
	floor := 2.0
	rows := []benchmarkGoldRow{
		{RunID: "spinning", Positive: boolPtr(true)},
		{RunID: "mild", Positive: boolPtr(false)},
	}
	views := map[string]runView{
		"spinning": {Labels: []annotationView{{
			ID: "agent.spinning", State: "observed", Probability: ptr(0.25), FrictionProbability: ptr(0.7),
			Value: value{Kind: "score", Score: ptr(3.0)}, Friction: friction{Enabled: true, MinScore: &floor},
		}}},
		"mild": {Labels: []annotationView{{
			ID: "agent.spinning", State: "observed", Probability: ptr(0.8), FrictionProbability: ptr(0.2),
			Value: value{Kind: "score", Score: ptr(1.0)}, Friction: friction{Enabled: true, MinScore: &floor},
		}}},
	}
	result := scoreGold(rows, views, "agent.spinning", 0.6)
	if result.TruePositive != 1 || result.TrueNegative != 1 || result.FalsePositive != 0 || result.FalseNegative != 0 {
		t.Fatalf("high-is-bad counts = %+v, want region mass to decide both rows", result)
	}
}

func TestWriteGoldDraftEmitsUnfilledRowsAndTheBenchmarkRejectsThem(t *testing.T) {
	draft := goldDraft{
		Label: "agent.spinning", DefinitionHash: "hash-current", Window: "30d",
		Runs: []goldDraftRun{{RunID: "run-1", Digest: `{"run_id":"run-1"}`}},
	}
	var out bytes.Buffer
	if err := writeGoldDraft(&out, draft); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"positive":null`) || !strings.Contains(out.String(), `"run_id":"run-1"`) {
		t.Fatalf("draft row = %s", out.String())
	}
	path := filepath.Join(t.TempDir(), "gold.jsonl")
	if err := os.WriteFile(path, out.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readGoldRows(path); err == nil || !strings.Contains(err.Error(), "unfilled") {
		t.Fatalf("unfilled draft was accepted: %v", err)
	}
}

func TestRenderBenchmarkCarriesTheThresholdIntoTheGate(t *testing.T) {
	var out bytes.Buffer
	renderBenchmark(&out, benchmarkResult{
		Label: "agent.spinning", Threshold: 0.6, Gold: 40, Revisions: 1,
		DefinitionHash: "abc", Precision: 0.82, Recall: 0.75,
	})
	text := out.String()
	if !strings.Contains(text, `"threshold":0.60`) {
		t.Fatalf("gate block = %q, want the measured threshold", text)
	}
}
