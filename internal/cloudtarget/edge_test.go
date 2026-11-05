package cloudtarget

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

type recordingCaddy struct {
	calls       []string
	validateErr error
	reloadErr   error
	// failOnce makes the first reload fail and later ones succeed, so the
	// restore-then-reload path is observable.
	failReloadOnce bool
}

func (c *recordingCaddy) Validate(context.Context) error {
	c.calls = append(c.calls, "validate")
	return c.validateErr
}

func (c *recordingCaddy) Reload(context.Context) error {
	c.calls = append(c.calls, "reload")
	if c.failReloadOnce {
		c.failReloadOnce = false
		return c.reloadErr
	}
	if c.reloadErr != nil && !c.failReloadOnce {
		return c.reloadErr
	}
	return nil
}

func edgeSpec(deployment, host string, port int) EdgeRouteSpec {
	return EdgeRouteSpec{
		SchemaVersion:   EdgeSpecSchemaVersion,
		DeploymentID:    deployment,
		Domain:          host,
		Routes:          []EdgeRoute{{Host: host, UpstreamPort: port, ListenerID: "app/ui"}},
		Snippet:         host + " {\n  reverse_proxy 127.0.0.1:" + strconv.Itoa(port) + "\n}\n",
		ACMEEnvironment: "staging",
	}
}

func edgeFixture(t *testing.T) (*Store, CaddyPaths) {
	t.Helper()
	root := t.TempDir()
	paths := CaddyPaths{MainConfig: filepath.Join(root, "etc", "caddy", "Caddyfile"), ConfDir: filepath.Join(root, "etc", "caddy", "conf.d"), DataDir: filepath.Join(root, "data")}
	if err := os.MkdirAll(paths.ConfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// An operator-owned main Caddyfile with an unrelated domain and another
	// deployment's snippet: neither may change.
	if err := os.WriteFile(paths.MainConfig, []byte("{\n  email ops@example.test\n}\nunrelated.example.test {\n  respond \"hi\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(paths.ConfDir, "vrooli-other.caddy"), []byte("neighbour.example.test {\n  reverse_proxy 127.0.0.1:4000\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return NewStore(filepath.Join(root, "runtime", "cloud", "deployments")), paths
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

// [REQ:STC-P0-036] P14-A06: applying one deployment's route appends the
// import line once, writes only the per-deployment snippet, and leaves the
// main Caddyfile body and the neighbouring snippet byte-identical.
func TestEdgeRouteApplyWritesOnlyOwnSnippetAndPreservesNeighbours(t *testing.T) {
	store, paths := edgeFixture(t)
	mainBefore := readFile(t, paths.MainConfig)
	neighbourBefore := readFile(t, filepath.Join(paths.ConfDir, "vrooli-other.caddy"))
	caddy := &recordingCaddy{}
	spec := edgeSpec("dep-a", "app.example.test", 3000)
	report, result, err := store.EdgeRouteApply(context.Background(), EdgeRouteApplyRequest{Effect: EffectRequest{DeploymentID: "dep-a", OperationID: "op-1", Step: "edge", Fence: 1}, Spec: spec, Paths: paths, Controller: caddy})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if result.Receipt.Outcome != OutcomeSucceeded || !report.ImportLineAdded || report.RolledBack {
		t.Fatalf("report=%+v receipt=%+v", report, result.Receipt)
	}
	if strings.Join(caddy.calls, ",") != "validate,reload" {
		t.Fatalf("controller calls = %v", caddy.calls)
	}
	mainAfter := readFile(t, paths.MainConfig)
	if !strings.HasPrefix(mainAfter, mainBefore) || strings.TrimSpace(strings.TrimPrefix(mainAfter, mainBefore)) != "import conf.d/*.caddy" {
		t.Fatalf("main Caddyfile was rewritten:\n%s", mainAfter)
	}
	if readFile(t, filepath.Join(paths.ConfDir, "vrooli-other.caddy")) != neighbourBefore {
		t.Fatal("neighbouring deployment snippet changed")
	}
	if readFile(t, paths.SnippetPath("dep-a")) != spec.Snippet {
		t.Fatal("snippet content differs from the spec")
	}
	// Second apply of a different route for the same deployment: import
	// line is not appended again, neighbour still untouched, previous kept.
	updated := edgeSpec("dep-a", "app.example.test", 3100)
	report2, result2, err := store.EdgeRouteApply(context.Background(), EdgeRouteApplyRequest{Effect: EffectRequest{DeploymentID: "dep-a", OperationID: "op-2", Step: "edge", Fence: 2}, Spec: updated, Paths: paths, Controller: caddy})
	if err != nil || result2.Receipt.Outcome != OutcomeSucceeded || report2.ImportLineAdded {
		t.Fatalf("second apply: report=%+v err=%v", report2, err)
	}
	if strings.Count(readFile(t, paths.MainConfig), "import conf.d/*.caddy") != 1 {
		t.Fatal("import line duplicated")
	}
	if readFile(t, filepath.Join(paths.ConfDir, "vrooli-other.caddy")) != neighbourBefore {
		t.Fatal("neighbouring deployment snippet changed on update")
	}
	if report2.UnrelatedHash != report.UnrelatedHash {
		t.Fatalf("unrelated hash moved between applies: %s -> %s", report.UnrelatedHash, report2.UnrelatedHash)
	}
	status, err := store.EdgeRouteStatus("dep-a", paths)
	if err != nil {
		t.Fatal(err)
	}
	if !status.PreviousRetained || !status.ImportLinePresent || len(status.Hosts) != 1 || status.Hosts[0].UpstreamPort != 3100 || status.OtherSnippetCount != 1 || status.UnrelatedHash != report.UnrelatedHash {
		t.Fatalf("status = %+v", status)
	}
	// Replay of the same (operation, step) returns the receipt without
	// touching the proxy again.
	calls := len(caddy.calls)
	_, replay, err := store.EdgeRouteApply(context.Background(), EdgeRouteApplyRequest{Effect: EffectRequest{DeploymentID: "dep-a", OperationID: "op-2", Step: "edge", Fence: 2}, Spec: updated, Paths: paths, Controller: caddy})
	if err != nil || !replay.Replayed || len(caddy.calls) != calls {
		t.Fatalf("replay = %+v err=%v calls=%d", replay, err, len(caddy.calls)-calls)
	}
	// Same spec, new operation: unchanged, no reload.
	_, same, err := store.EdgeRouteApply(context.Background(), EdgeRouteApplyRequest{Effect: EffectRequest{DeploymentID: "dep-a", OperationID: "op-3", Step: "edge", Fence: 3}, Spec: updated, Paths: paths, Controller: caddy})
	if err != nil || same.Receipt.Outcome != OutcomeUnchanged || len(caddy.calls) != calls {
		t.Fatalf("unchanged apply = %+v err=%v", same.Receipt, err)
	}
}

// [REQ:STC-P0-036] P14-A03: a snippet that fails validation or reload is
// restored to the previous snippet, the receipt records the rollback, and
// the prior working route is what remains on disk.
func TestEdgeRouteApplyRestoresPreviousSnippetOnFailure(t *testing.T) {
	store, paths := edgeFixture(t)
	good := edgeSpec("dep-b", "shop.example.test", 3000)
	if _, _, err := store.EdgeRouteApply(context.Background(), EdgeRouteApplyRequest{Effect: EffectRequest{DeploymentID: "dep-b", OperationID: "op-1", Step: "edge", Fence: 1}, Spec: good, Paths: paths, Controller: &recordingCaddy{}}); err != nil {
		t.Fatal(err)
	}
	bad := edgeSpec("dep-b", "shop.example.test", 3001)
	caddy := &recordingCaddy{validateErr: fail(CodeEdgeConfigInvalid, "caddy validate: unrecognized directive")}
	report, result, err := store.EdgeRouteApply(context.Background(), EdgeRouteApplyRequest{Effect: EffectRequest{DeploymentID: "dep-b", OperationID: "op-2", Step: "edge", Fence: 2}, Spec: bad, Paths: paths, Controller: caddy})
	if code, _ := codeOf(t, err); code != CodeEdgeConfigInvalid {
		t.Fatalf("code = %s", code)
	}
	if !report.RolledBack || result.Receipt.Outcome != OutcomeFailed || result.Receipt.Details["rolled_back"] != true {
		t.Fatalf("report=%+v receipt=%+v", report, result.Receipt)
	}
	if readFile(t, paths.SnippetPath("dep-b")) != good.Snippet {
		t.Fatal("previous snippet was not restored after a failed validate")
	}
	if strings.Join(caddy.calls, ",") != "validate" {
		t.Fatalf("a failed validate must not reload: %v", caddy.calls)
	}
	reloadFail := &recordingCaddy{reloadErr: fail(CodeEdgeReloadFailed, "systemctl reload caddy: exit 1"), failReloadOnce: true}
	report, _, err = store.EdgeRouteApply(context.Background(), EdgeRouteApplyRequest{Effect: EffectRequest{DeploymentID: "dep-b", OperationID: "op-3", Step: "edge", Fence: 3}, Spec: bad, Paths: paths, Controller: reloadFail})
	if code, _ := codeOf(t, err); code != CodeEdgeReloadFailed || !report.RolledBack {
		t.Fatalf("reload failure code=%s report=%+v", code, report)
	}
	if readFile(t, paths.SnippetPath("dep-b")) != good.Snippet {
		t.Fatal("previous snippet was not restored after a failed reload")
	}
	if strings.Join(reloadFail.calls, ",") != "validate,reload,reload" {
		t.Fatalf("restore must reload the previous configuration: %v", reloadFail.calls)
	}
	// A first-ever apply that fails leaves no snippet behind.
	first := edgeSpec("dep-c", "new.example.test", 3000)
	_, _, err = store.EdgeRouteApply(context.Background(), EdgeRouteApplyRequest{Effect: EffectRequest{DeploymentID: "dep-c", OperationID: "op-1", Step: "edge", Fence: 1}, Spec: first, Paths: paths, Controller: &recordingCaddy{validateErr: errors.New("bad")}})
	if err == nil {
		t.Fatal("expected failure")
	}
	if _, statErr := os.Stat(paths.SnippetPath("dep-c")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatal("failed first apply left a snippet on disk")
	}
}

// [REQ:STC-P0-036] rollback restores the displaced snippet and is refused
// when nothing was displaced.
func TestEdgeRouteRollbackRestoresDisplacedSnippet(t *testing.T) {
	store, paths := edgeFixture(t)
	ctx := context.Background()
	if _, _, err := store.EdgeRouteRollback(ctx, EdgeRouteRollbackRequest{Effect: EffectRequest{DeploymentID: "dep-r", OperationID: "op-0", Step: "rollback", Fence: 1}, Paths: paths, Controller: &recordingCaddy{}}); err == nil {
		t.Fatal("rollback without a previous snippet must be refused")
	} else if code, exit := codeOf(t, err); code != CodeEdgeRollbackNotEligible || exit != ExitRefused {
		t.Fatalf("code/exit = %s/%d", code, exit)
	}
	v1 := edgeSpec("dep-r", "r.example.test", 3000)
	v2 := edgeSpec("dep-r", "r.example.test", 3001)
	for i, spec := range []EdgeRouteSpec{v1, v2} {
		if _, _, err := store.EdgeRouteApply(ctx, EdgeRouteApplyRequest{Effect: EffectRequest{DeploymentID: "dep-r", OperationID: "op-" + strconv.Itoa(i+1), Step: "edge", Fence: uint64(i + 1)}, Spec: spec, Paths: paths, Controller: &recordingCaddy{}}); err != nil {
			t.Fatal(err)
		}
	}
	caddy := &recordingCaddy{}
	report, result, err := store.EdgeRouteRollback(ctx, EdgeRouteRollbackRequest{Effect: EffectRequest{DeploymentID: "dep-r", OperationID: "op-3", Step: "rollback", Fence: 3}, Paths: paths, Controller: caddy})
	if err != nil || result.Receipt.Outcome != OutcomeSucceeded || !report.RolledBack {
		t.Fatalf("rollback report=%+v receipt=%+v err=%v", report, result.Receipt, err)
	}
	if readFile(t, paths.SnippetPath("dep-r")) != v1.Snippet {
		t.Fatal("rollback did not restore the displaced snippet")
	}
	if strings.Join(caddy.calls, ",") != "validate,reload" {
		t.Fatalf("calls = %v", caddy.calls)
	}
	// Stale fence is refused before any file changes.
	before := readFile(t, paths.SnippetPath("dep-r"))
	_, _, err = store.EdgeRouteApply(ctx, EdgeRouteApplyRequest{Effect: EffectRequest{DeploymentID: "dep-r", OperationID: "op-old", Step: "edge", Fence: 1}, Spec: v2, Paths: paths, Controller: caddy})
	if code, _ := codeOf(t, err); code != CodeFenceStale || readFile(t, paths.SnippetPath("dep-r")) != before {
		t.Fatalf("stale fence code=%s", code)
	}
}

// [REQ:STC-P0-036] P14-A02/P14-O04: the target refuses a spec that would
// route public traffic to a management listener, an undeclared host, a
// non-loopback upstream, or that smuggles imports or global options.
func TestEdgeSpecRefusesPrivateListenersAndOutOfScopeSnippets(t *testing.T) {
	base := edgeSpec("dep-x", "app.example.test", 3000)
	cases := map[string]struct {
		mutate func(*EdgeRouteSpec)
		code   string
	}{
		"ssh management port": {func(s *EdgeRouteSpec) {
			s.Routes[0].UpstreamPort = 22
			s.Snippet = "app.example.test {\n  reverse_proxy 127.0.0.1:22\n}\n"
		}, CodeEdgePrivateListener},
		"bridge management port": {func(s *EdgeRouteSpec) {
			s.Routes[0].UpstreamPort = 18767
			s.Snippet = "app.example.test {\n  reverse_proxy 127.0.0.1:18767\n}\n"
		}, CodeEdgePrivateListener},
		"undeclared host": {func(s *EdgeRouteSpec) { s.Snippet += "other.example.test {\n  reverse_proxy 127.0.0.1:3000\n}\n" }, CodeEdgeSnippetOutOfScope},
		"undeclared port": {func(s *EdgeRouteSpec) { s.Snippet = "app.example.test {\n  reverse_proxy 127.0.0.1:5432\n}\n" }, CodeEdgeSnippetOutOfScope},
		"remote upstream": {func(s *EdgeRouteSpec) { s.Snippet = "app.example.test {\n  reverse_proxy 10.0.0.9:3000\n}\n" }, CodeEdgeSnippetOutOfScope},
		"import smuggled": {func(s *EdgeRouteSpec) {
			s.Snippet = "app.example.test {\n  import /etc/caddy/other\n  reverse_proxy 127.0.0.1:3000\n}\n"
		}, CodeEdgeSnippetOutOfScope},
		"global options": {func(s *EdgeRouteSpec) { s.Snippet = "{\n  admin off\n}\n" + s.Snippet }, CodeEdgeSnippetOutOfScope},
		"bad acme env":   {func(s *EdgeRouteSpec) { s.ACMEEnvironment = "prod" }, CodeEdgeSpecInvalid},
		"invalid host":   {func(s *EdgeRouteSpec) { s.Routes[0].Host = "not a host" }, CodeEdgeSpecInvalid},
		"missing site block": {func(s *EdgeRouteSpec) {
			s.Routes = append(s.Routes, EdgeRoute{Host: "b.example.test", UpstreamPort: 3000, ListenerID: "app/api"})
		}, CodeEdgeSpecInvalid},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			spec := base
			spec.Routes = append([]EdgeRoute(nil), base.Routes...)
			tc.mutate(&spec)
			err := spec.Validate()
			if code, exit := codeOf(t, err); code != tc.code || exit != ExitRefused {
				t.Fatalf("code/exit = %s/%d (%v)", code, exit, err)
			}
		})
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("base spec refused: %v", err)
	}
	if _, err := ParseEdgeRouteSpec([]byte(`{"schema_version":1,"deployment_id":"d","domain":"a.example.test","routes":[],"snippet":"","acme_environment":"staging","command":"rm"}`)); err == nil {
		t.Fatal("unknown field accepted")
	}
}

// Certificate metadata is read from Caddy's storage; only public fields are
// reported.
func TestEdgeRouteStatusReportsCertificateExpiry(t *testing.T) {
	store, paths := edgeFixture(t)
	spec := edgeSpec("dep-s", "s.example.test", 3000)
	if _, _, err := store.EdgeRouteApply(context.Background(), EdgeRouteApplyRequest{Effect: EffectRequest{DeploymentID: "dep-s", OperationID: "op-1", Step: "edge", Fence: 1}, Spec: spec, Paths: paths, Controller: &recordingCaddy{}}); err != nil {
		t.Fatal(err)
	}
	certDir := filepath.Join(paths.DataDir, "certificates", "acme-staging-v02.api.letsencrypt.org-directory", "s.example.test")
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		t.Fatal(err)
	}
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	notAfter := time.Now().Add(20 * 24 * time.Hour)
	ca := &x509.Certificate{SerialNumber: big.NewInt(9), Subject: pkix.Name{CommonName: "(STAGING) Fake CA"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: notAfter.Add(24 * time.Hour), IsCA: true, BasicConstraintsValid: true}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "s.example.test"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: notAfter, DNSNames: []string{"s.example.test"}}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "s.example.test.crt"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "s.example.test.key"), []byte("canary-private-key-9f3a"), 0o600); err != nil {
		t.Fatal(err)
	}
	status, err := store.EdgeRouteStatus("dep-s", paths)
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Hosts) != 1 || !status.Hosts[0].CertificatePresent || status.Hosts[0].DaysLeft < 18 || status.Hosts[0].DaysLeft > 20 || status.Hosts[0].Issuer != "(STAGING) Fake CA" || status.ACMEEnvironment != "staging" {
		t.Fatalf("status = %+v", status)
	}
	if raw := mustJSON(t, status); strings.Contains(raw, "canary-private-key") {
		t.Fatal("status carries private key material")
	}
}
