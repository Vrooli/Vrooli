package modeltest_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"{{SCENARIO_ID}}/internal/testutil/modeltest"

	"github.com/stretchr/testify/require"
)

func TestValidateFormalArtifactFresh_AcceptsCurrentModelHash(t *testing.T) {
	modelPath := writeFormalModel(t, "module Example {}\n")
	artifact := validFormalArtifact(t, modelPath)

	errs := modeltest.ValidateFormalArtifactFresh(artifact, modelPath)
	require.Empty(t, errs)
}

func TestValidateFormalArtifactFresh_RejectsStaleHashAndMissingChecks(t *testing.T) {
	modelPath := writeFormalModel(t, "module Example {}\n")
	artifact := validFormalArtifact(t, modelPath)
	artifact.Source.ModelSHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
	artifact.Checks.Verified = false

	errs := modeltest.ValidateFormalArtifactFresh(artifact, modelPath)
	require.NotEmpty(t, errs)
	require.Contains(t, joined(errs), "formal artifact modelSha256=")
	require.Contains(t, joined(errs), "formal artifact was not verified")
}

func TestValidateFormalTransitionsReplay_AcceptsGeneratedMatrix(t *testing.T) {
	artifact := validFormalArtifact(t, writeFormalModel(t, "module Example {}\n"))

	errs := modeltest.ValidateFormalTransitionsReplay(
		artifact,
		[]status{statusIdle, statusBusy, statusDone},
		[]event{eventStart, eventFinish},
		transition,
	)
	require.Empty(t, errs)
}

func TestValidateFormalTransitionsReplay_RejectsUnknownAndDivergentTransition(t *testing.T) {
	artifact := validFormalArtifact(t, writeFormalModel(t, "module Example {}\n"))
	artifact.Transitions[0].To = "done"
	artifact.Transitions[1].Event = "ghost"

	errs := modeltest.ValidateFormalTransitionsReplay(
		artifact,
		[]status{statusIdle, statusBusy, statusDone},
		[]event{eventStart, eventFinish},
		transition,
	)
	require.NotEmpty(t, errs)
	require.Contains(t, joined(errs), "unknown event ghost")
}

func TestValidateFormalTracesReplay_RejectsUnknownAndDivergentTrace(t *testing.T) {
	artifact := validFormalArtifact(t, writeFormalModel(t, "module Example {}\n"))
	artifact.Traces[0].Steps[0].Want = "done"
	artifact.Traces[0].Steps[1].Event = "ghost"

	errs := modeltest.ValidateFormalTracesReplay(
		artifact,
		[]status{statusIdle, statusBusy, statusDone},
		[]event{eventStart, eventFinish},
		transition,
	)
	require.NotEmpty(t, errs)
	require.Contains(t, joined(errs), "unknown event ghost")
}

func validFormalArtifact(t *testing.T, modelPath string) modeltest.FormalArtifact {
	t.Helper()
	return modeltest.FormalArtifact{
		SchemaVersion: 1,
		FlowID:        "example.flow",
		Source: modeltest.FormalArtifactSource{
			ModelPath:    modelPath,
			ModelSHA256:  fileSHA256(t, modelPath),
			QuintVersion: "0.32.0",
		},
		Commands: map[string][]string{
			"typecheck": {"quint", "typecheck", modelPath},
			"test":      {"quint", "test", modelPath},
			"verify":    {"quint", "verify", modelPath},
			"run":       {"quint", "run", modelPath},
		},
		States: []string{"idle", "busy", "done"},
		Events: []string{"start", "finish"},
		Transitions: []modeltest.FormalArtifactTransition{
			{From: "idle", Event: "start", To: "busy", WantError: false},
			{From: "idle", Event: "finish", To: "idle", WantError: true},
			{From: "busy", Event: "start", To: "busy", WantError: true},
			{From: "busy", Event: "finish", To: "done", WantError: false},
			{From: "done", Event: "start", To: "done", WantError: true},
			{From: "done", Event: "finish", To: "done", WantError: true},
		},
		Traces: []modeltest.FormalArtifactTrace{
			{
				Name:    "generated",
				Initial: "idle",
				Steps: []modeltest.FormalArtifactTraceStep{
					{Event: "start", Want: "busy", WantError: false},
					{Event: "finish", Want: "done", WantError: false},
				},
			},
		},
		Checks: modeltest.FormalArtifactChecks{
			Typechecked:        true,
			Tested:             true,
			Verified:           true,
			GeneratedFromModel: true,
		},
	}
}

func writeFormalModel(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "model.qnt")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
