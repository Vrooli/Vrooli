package audit_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"tunnel-manager/internal/audit"
	internalroutes "tunnel-manager/internal/routes"

	"github.com/stretchr/testify/require"
)

// fakeRoutesReader is a minimal audit.RoutesReader for service tests. It
// returns a fixed set of routes and records the tier filter it was passed so
// tests can assert the audit service reads the whole manifest.
type fakeRoutesReader struct {
	routes   []internalroutes.Route
	listErr  error
	listTier internalroutes.Tier
}

func (f *fakeRoutesReader) List(_ context.Context, tier internalroutes.Tier) ([]internalroutes.Route, error) {
	f.listTier = tier
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.routes, nil
}

// writeServiceJSON creates <root>/<scenario>/.vrooli/service.json with the
// given UI port. A uiPort of 0 writes a ports map with no fixed ui.port (a
// ranged-only entry), matching the real "ports": {"ui": {"range": ...}} shape.
func writeServiceJSON(t *testing.T, root, scenario string, uiPort int) {
	t.Helper()
	dir := filepath.Join(root, scenario, ".vrooli")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	var body string
	if uiPort == 0 {
		body = `{"ports":{"ui":{"env_var":"UI_PORT","range":"20000-24999"}}}`
	} else {
		body = `{"ports":{"ui":{"env_var":"UI_PORT","port":` + itoa(uiPort) + `}}}`
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "service.json"), []byte(body), 0o644))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func enabledRoute(subdomain, scenario string, port int) internalroutes.Route {
	return internalroutes.Route{
		Subdomain: subdomain,
		Scenario:  scenario,
		LocalPort: port,
		Enabled:   true,
	}
}

func TestRunAudit_ClassifiesAllStatuses(t *testing.T) {
	root := t.TempDir()
	// compliant: service.json ui.port matches the manifest local_port.
	writeServiceJSON(t, root, "web-console", 21233)
	// mismatch: service.json declares a different ui.port.
	writeServiceJSON(t, root, "agent-manager", 21238)
	// missing_port: service.json has a ranged-only ui entry (no fixed port).
	writeServiceJSON(t, root, "ranged-scenario", 0)
	// missing_scenario: no service.json written for "ghost-scenario".

	reader := &fakeRoutesReader{routes: []internalroutes.Route{
		enabledRoute("web-console", "web-console", 21233),
		enabledRoute("agent-manager", "agent-manager", 99999),
		enabledRoute("ranged", "ranged-scenario", 21000),
		enabledRoute("ghost", "ghost-scenario", 21001),
	}}
	svc := audit.NewService(reader, root)

	results, err := svc.RunAudit(context.Background())
	require.NoError(t, err)
	require.Equal(t, internalroutes.Tier(""), reader.listTier, "audit reads the whole manifest")
	require.Len(t, results, 4)

	byScenario := map[string]audit.PortAuditResult{}
	for _, r := range results {
		byScenario[r.Scenario] = r
	}

	// compliant
	compliant := byScenario["web-console"]
	require.Equal(t, audit.StatusCompliant, compliant.Status)
	require.Equal(t, 21233, compliant.ExpectedPort)
	require.Equal(t, 21233, compliant.ActualPort)
	require.Empty(t, compliant.Detail)

	// mismatch
	mismatch := byScenario["agent-manager"]
	require.Equal(t, audit.StatusMismatch, mismatch.Status)
	require.Equal(t, 99999, mismatch.ExpectedPort)
	require.Equal(t, 21238, mismatch.ActualPort)
	require.Contains(t, mismatch.Detail, "manifest expects port 99999")
	require.Contains(t, mismatch.Detail, "service.json has 21238")

	// missing_port
	missingPort := byScenario["ranged-scenario"]
	require.Equal(t, audit.StatusMissingPort, missingPort.Status)
	require.Equal(t, 0, missingPort.ActualPort)
	require.Contains(t, missingPort.Detail, "no fixed UI port")

	// missing_scenario
	missingScenario := byScenario["ghost-scenario"]
	require.Equal(t, audit.StatusMissingScenario, missingScenario.Status)
	require.Equal(t, 0, missingScenario.ActualPort)
	require.Contains(t, missingScenario.Detail, "service.json not found")
}

func TestRunAudit_SkipsDisabledRoutes(t *testing.T) {
	root := t.TempDir()
	writeServiceJSON(t, root, "web-console", 21233)

	reader := &fakeRoutesReader{routes: []internalroutes.Route{
		enabledRoute("web-console", "web-console", 21233),
		{Subdomain: "disabled", Scenario: "web-console", LocalPort: 21233, Enabled: false},
	}}
	svc := audit.NewService(reader, root)

	results, err := svc.RunAudit(context.Background())
	require.NoError(t, err)
	require.Len(t, results, 1, "disabled routes are not audited")
	require.Equal(t, "web-console", results[0].Subdomain)
}

func TestRunAudit_ListErrorPropagates(t *testing.T) {
	reader := &fakeRoutesReader{listErr: context.DeadlineExceeded}
	svc := audit.NewService(reader, t.TempDir())

	_, err := svc.RunAudit(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestViolationCount(t *testing.T) {
	results := []audit.PortAuditResult{
		{Status: audit.StatusCompliant},
		{Status: audit.StatusMismatch},
		{Status: audit.StatusMissingScenario},
		{Status: audit.StatusCompliant},
		{Status: audit.StatusMissingPort},
	}
	require.Equal(t, 3, audit.ViolationCount(results))
}
