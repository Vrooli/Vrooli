package impact

import (
	"os"
	"path/filepath"
	"testing"

	"proto-health/internal/protosurface"

	"github.com/stretchr/testify/require"
	impactv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/impact"
)

func TestParseBreakingOutputClassifiesJSONFindings(t *testing.T) {
	raw := []byte(`[
		{"path":"schemas/demo/v1/orders/orders.proto","start_line":12,"message":"Field number changed from 1 to 2."},
		{"path":"demo/v1/orders/orders.proto","message":"Previously present field \"name\" was deleted."},
		{"path":"demo/v1/orders/orders.proto","message":"Field type changed from string to int64."}
	]`)

	changes := parseBreakingOutput(raw, map[string]string{"demo/v1/orders/orders.proto": "stable"})

	require.Len(t, changes, 3)
	require.Equal(t, "demo/v1/orders/orders.proto", changes[0].File)
	require.Equal(t, "demo/v1/orders/orders.proto:12", changes[0].Path)
	require.Equal(t, impactv1.ImpactChangeKind_IMPACT_CHANGE_KIND_RENUMBER, changes[0].Kind)
	require.True(t, changes[0].WireBreaking)
	require.False(t, changes[0].JsonBreaking)
	require.Equal(t, "stable", changes[0].Stability)
	require.Equal(t, impactv1.ImpactChangeKind_IMPACT_CHANGE_KIND_REMOVE, changes[1].Kind)
	require.True(t, changes[1].JsonBreaking)
	require.Equal(t, impactv1.ImpactChangeKind_IMPACT_CHANGE_KIND_RETYPE, changes[2].Kind)
}

func TestParseBreakingOutputHandlesTextFallback(t *testing.T) {
	changes := parseBreakingOutput([]byte("demo/v1/orders/orders.proto:4:1: Field type changed from string to int64."), nil)

	require.Len(t, changes, 1)
	require.Equal(t, "demo/v1/orders/orders.proto", changes[0].File)
	require.Equal(t, impactv1.ImpactChangeKind_IMPACT_CHANGE_KIND_RETYPE, changes[0].Kind)
}

func TestReportFromBreakingOutputAggregatesCounts(t *testing.T) {
	report := reportFromBreakingOutput("demo", resolvedScope{input: "HEAD", kind: "head", sha: "abc123"}, []byte(`[
		{"path":"schemas/demo/v1/orders/orders.proto","start_line":12,"message":"Field number changed from 1 to 2."},
		{"path":"schemas/demo/v1/orders/orders.proto","message":"Field type changed from string to int64."},
		{"path":"schemas/other/v1/api/api.proto","message":"Previously present field was deleted."}
	]`), protosurface.Surface{
		Files: []protosurface.File{{Path: "demo/v1/orders/orders.proto", Stability: "stable"}},
	})

	require.Equal(t, "demo", report.Scenario)
	require.Equal(t, "HEAD", report.Scope)
	require.Equal(t, "head", report.ScopeKind)
	require.Equal(t, "abc123", report.BaselineSha)
	require.Len(t, report.Changes, 2)
	require.EqualValues(t, 2, report.WireBreakingCount)
	require.EqualValues(t, 1, report.JsonBreakingCount)
	require.Equal(t, "stable", report.Changes[0].Stability)
}

func TestReportFromBreakingOutputAttachesUnreconciledConsumers(t *testing.T) {
	report := reportFromBreakingOutput("channel", resolvedScope{input: "baseline:main", kind: "baseline", sha: "abc123", baselineName: "main", commitsSince: 2, likelyStale: true}, []byte(`[
		{"path":"schemas/channel/v1/shared/events.proto","message":"Previously present field was deleted."}
	]`), protosurface.Surface{
		Files: []protosurface.File{{Path: "channel/v1/shared/events.proto", Stability: "stable"}},
		CrossScenarioImports: []protosurface.Import{
			{FromScenario: "presence", FromFile: "presence/v1/api/service.proto", ToFile: "channel/v1/shared/events.proto"},
			{FromScenario: "runs", FromFile: "runs/v1/domain/run.proto", ToFile: "channel/v1/shared/events.proto"},
		},
	})

	require.Equal(t, "baseline", report.ScopeKind)
	require.Equal(t, "main", report.BaselineName)
	require.EqualValues(t, 2, report.CommitsSinceBaseline)
	require.True(t, report.LikelyStale)
	require.EqualValues(t, 2, report.UnreconciledConsumerCount)
	require.EqualValues(t, 1, report.StableUnreconciledBreakingCount)
	require.Len(t, report.Changes[0].UnreconciledConsumers, 2)
	require.Equal(t, "presence", report.UnreconciledConsumers[0].Scenario)
	require.Equal(t, "runs", report.UnreconciledConsumers[1].Scenario)
}

func TestReportFromBreakingOutputSkipsExperimentalConsumers(t *testing.T) {
	report := reportFromBreakingOutput("channel", resolvedScope{input: "HEAD", kind: "head", sha: "abc123"}, []byte(`[
		{"path":"schemas/channel/v1/shared/events.proto","message":"Previously present field was deleted."}
	]`), protosurface.Surface{
		Files: []protosurface.File{{Path: "channel/v1/shared/events.proto", Stability: "experimental"}},
		CrossScenarioImports: []protosurface.Import{
			{FromScenario: "presence", FromFile: "presence/v1/api/service.proto", ToFile: "channel/v1/shared/events.proto"},
		},
	})

	require.Zero(t, report.UnreconciledConsumerCount)
	require.Empty(t, report.Changes[0].UnreconciledConsumers)
}

func TestCopyCurrentBufInputsCopiesConfigAndVendor(t *testing.T) {
	root := t.TempDir()
	currentProto := filepath.Join(root, "packages", "proto")
	require.NoError(t, os.MkdirAll(filepath.Join(currentProto, "vendor", "protovalidate", "buf", "validate"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(currentProto, "buf.yaml"), []byte("version: v2\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(currentProto, "buf.lock"), []byte("version: v2\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(currentProto, "vendor", "protovalidate", "buf", "validate", "validate.proto"), []byte("syntax = \"proto3\";\n"), 0o644))

	dest := filepath.Join(t.TempDir(), "packages", "proto")
	require.NoError(t, copyCurrentBufInputs(root, dest))

	require.FileExists(t, filepath.Join(dest, "buf.yaml"))
	require.FileExists(t, filepath.Join(dest, "buf.lock"))
	require.FileExists(t, filepath.Join(dest, "vendor", "protovalidate", "buf", "validate", "validate.proto"))
}

func TestCommandRunnerInterfaceShape(t *testing.T) {
	var _ BreakingRunner = commandRunner{}
	var _ SurfaceLoader = staticLoader{}
}

type staticLoader struct {
	surface protosurface.Surface
}

func (s staticLoader) LoadScenario(string) (protosurface.Surface, error) {
	return s.surface, nil
}
