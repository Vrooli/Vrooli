package catalogcoverage

import (
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCorpusReportHelpersClassifyDeclarationsAndAllowlist(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "gates.json")
	require.NoError(t, os.WriteFile(config, []byte(`{"gates":[{"id":"unit","blocking":true},{"id":"visual","blocking":true},{"id":"other","blocking":false}]}`), 0o600))
	runnerless, blocking := blockingGateCounts(config)
	require.Equal(t, 2, runnerless)
	require.Equal(t, 2, blocking)
	missingRunnerless, missingBlocking := blockingGateCounts(filepath.Join(root, "missing"))
	require.Zero(t, missingRunnerless)
	require.Zero(t, missingBlocking)
	require.NoError(t, os.WriteFile(config, []byte(`{`), 0o600))
	invalidRunnerless, invalidBlocking := blockingGateCounts(config)
	require.Zero(t, invalidRunnerless)
	require.Zero(t, invalidBlocking)

	allowlist := filepath.Join(root, "allowlist.json")
	require.NoError(t, os.WriteFile(allowlist, []byte(`{"entries":[{},{}]}`), 0o600))
	require.Equal(t, 2, countAllowlistEntries(allowlist))
	require.Equal(t, 0, countAllowlistEntries(filepath.Join(root, "missing")))
	require.NoError(t, os.WriteFile(allowlist, []byte(`{`), 0o600))
	require.Equal(t, 0, countAllowlistEntries(allowlist))
}

func TestExperienceClaimCountsSkipReferencesAndSeparateClaimTiers(t *testing.T) {
	root := t.TempDir()
	canonical := filepath.Join(root, "canonical")
	library := filepath.Join(root, "library")
	require.NoError(t, os.MkdirAll(canonical, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(library, "components", "Panel"), 0o755))
	write := func(path, body string) {
		t.Helper()
		require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	}
	write(filepath.Join(canonical, "claims.json"), `{"claims":[{"tier":"machine"},{"tier":"MANUAL"},{"tier":""}]}`)
	write(filepath.Join(canonical, "reference.json"), `{"kind":"experience-reference","claims":[{"tier":"machine"}]}`)
	write(filepath.Join(library, "components", "Panel", "experience-contract.json"), `{"claims":[{"tier":"machine"},{"tier":"human"}]}`)
	manual, machine := experienceClaimCounts(canonical, library)
	require.Equal(t, 3, manual)
	require.Equal(t, 2, machine)
}

func TestReleaseAndRetirementMeasurementsIgnoreInvalidEntries(t *testing.T) {
	root := t.TempDir()
	versions := filepath.Join(root, "components", "Panel", "versions")
	require.NoError(t, os.MkdirAll(versions, 0o755))
	for _, name := range []string{"1.0.0", "1.1.0", "bad", "2.0.0-rc.1"} {
		require.NoError(t, os.MkdirAll(filepath.Join(versions, name), 0o755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, "release-provenance.json"), []byte(`{"entries":[{"LibraryID":"react-component-library:Panel","Version":"1.0.0"}]}`), 0o600))
	require.Equal(t, 1, countUngovernedReleases(root))
	require.Equal(t, 0, countUngovernedReleases(filepath.Join(root, "absent")))

	retired := filepath.Join(root, ".retired", "expired")
	require.NoError(t, os.MkdirAll(retired, 0o755))
	old := time.Now().Add(-30 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(retired, old, old))
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".retired", "recent"), 0o755))
	require.Equal(t, 1, overdueRetiredTrees(root, 7))
	require.Equal(t, 0, overdueRetiredTrees(filepath.Join(root, "absent"), 7))
}

func TestLiveBuildProbeDistinguishesMissingSuccessfulAndFailingGenerator(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node unavailable")
	}
	root := t.TempDir()
	script := filepath.Join(root, "packages", "react-component-library", "tooling", "catalog-build.mjs")
	require.NoError(t, os.MkdirAll(filepath.Dir(script), 0o755))
	require.NoError(t, os.WriteFile(script, []byte("process.exit(0)\n"), 0o600))
	failures, measured := liveBuildFailures(root)
	require.True(t, measured)
	require.Zero(t, failures)
	require.NoError(t, os.WriteFile(script, []byte("process.exit(1)\n"), 0o600))
	failures, measured = liveBuildFailures(root)
	require.True(t, measured)
	require.Equal(t, 1, failures)
}

func TestCountMatrixFailuresReportsUnavailableInputs(t *testing.T) {
	root := t.TempDir()
	_, err := CountMatrixFailures(context.Background(), root, nil)
	require.Error(t, err, "missing gate config must be surfaced")
	config := filepath.Join(root, "scenarios", "react-component-library", "catalog", "config.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(config), 0o755))
	require.NoError(t, os.WriteFile(config, []byte(`{"gateTiers":[{"id":"release-integrity"},{"id":"asset-quality"},{"id":"consumer-use"}],"gates":[]}`), 0o600))
	_, err = CountMatrixFailures(context.Background(), root, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "needs a catalog database")
	count, err := CountMatrixFailures(context.Background(), root, &sql.DB{})
	require.NoError(t, err)
	require.Zero(t, count)
}
