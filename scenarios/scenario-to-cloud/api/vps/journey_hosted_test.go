package vps

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/cloudrelease"
	"scenario-to-cloud/closure"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

// Only the transport is local. Every target invocation executes the source-built
// control plane, including release verification/staging/activation and lifecycle
// start. No command result or activation receipt is synthesized here.
type hostedSignInReach struct {
	cli, root string
	env       []string
}

func (r *hostedSignInReach) Exec(ctx context.Context, _ identity.TargetRef, command reach.Command) (reach.Result, error) {
	cmd := exec.CommandContext(ctx, r.cli, append(strings.Fields(command.Verb), command.Args...)...)
	cmd.Dir, cmd.Env = r.root, r.env
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	result := reach.Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if exit, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exit.ExitCode()
		return result, nil // a completed owner refusal, rather than transport loss
	}
	return result, err
}

func (*hostedSignInReach) Deliver(context.Context, identity.TargetRef, reach.Delivery) (reach.DeliveryReceipt, error) {
	return reach.DeliveryReceipt{}, fmt.Errorf("offline target is already local")
}
func (*hostedSignInReach) Negotiate(context.Context, identity.TargetRef) (reach.Capabilities, error) {
	return reach.Capabilities{}, fmt.Errorf("offline target must not contact a VPS")
}

func TestJ13_SignInSurvivesOptionalDependencyFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 9*time.Minute)
	t.Cleanup(cancel)
	repo, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	root, home := t.TempDir(), t.TempDir()
	cache := os.Getenv("GOCACHE")
	if cache == "" {
		var rawCache []byte
		rawCache, err = exec.Command("go", "env", "GOCACHE").Output()
		cache = string(rawCache)
		if err != nil {
			t.Fatal(err)
		}
	}
	env := append(os.Environ(), "HOME="+home, "VROOLI_HOME="+filepath.Join(home, ".vrooli"), "VROOLI_ROOT="+root,
		"VROOLI_SOURCE_ROOT="+root, "VROOLI_CREDENTIAL_BACKEND=encrypted-file", "XDG_RUNTIME_DIR="+filepath.Join(home, "runtime"), "GOCACHE="+strings.TrimSpace(cache),
		"VROOLI_CLI_ARTIFACT_MODE=1", "VROOLI_RUNTIME_SUPERVISOR=on", "LPBS_TEST_CREDENTIAL_FALLBACK=0")
	if err := os.MkdirAll(filepath.Join(home, "runtime"), 0700); err != nil {
		t.Fatal(err)
	}
	cli := filepath.Join(root, ".vrooli/bin/vrooli")
	copyHostedFile(t, filepath.Join(repo, ".vrooli/build/vrooli"), cli, 0755)
	copyHostedFile(t, filepath.Join(repo, ".vrooli/repo-contract.json"), filepath.Join(root, ".vrooli/repo-contract.json"), 0644)
	copyHostedFile(t, filepath.Join(repo, "go.mod"), filepath.Join(root, "go.mod"), 0644)
	for _, name := range []string{"templates", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0755); err != nil {
			t.Fatal(err)
		}
	}
	writeHostedJSON(t, filepath.Join(root, ".vrooli/service.json"), map[string]any{"service": map[string]string{"name": "vrooli"}})
	target := &hostedSignInReach{cli: cli, root: root, env: env}
	// Keep a real supervisor alive in the isolated HOME. A lifecycle lease
	// cannot outlive its start command without this production handoff owner.
	supervisorCtx, stopSupervisor := context.WithCancel(ctx)
	supervisor := exec.CommandContext(supervisorCtx, cli, "runtime", "supervisor", "run")
	supervisor.Dir, supervisor.Env = root, env
	var supervisorOutput bytes.Buffer
	supervisor.Stdout, supervisor.Stderr = &supervisorOutput, &supervisorOutput
	if err := supervisor.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = supervisor.Process.Signal(os.Interrupt)
		if err := supervisor.Wait(); err != nil {
			if !strings.Contains(supervisorOutput.String(), "context canceled") {
				t.Errorf("isolated runtime supervisor: %v: %s", err, supervisorOutput.String())
			}
		}
		stopSupervisor()
	})

	// The helper prepares only fixture storage and the actual production binary.
	// It cannot activate or serve the product and stays alive until lifecycle stop.
	fixture := startHostedFixture(t, ctx, repo, root, env)
	port := 0
	startPort := 15000 + int(time.Now().UnixNano()%5000)
	for offset := 0; offset < 5000; offset++ {
		candidate := 15000 + (startPort-15000+offset)%5000
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", candidate))
		if err == nil {
			port = candidate
			listener.Close()
			break
		}
	}
	if port == 0 {
		t.Fatal("no free canonical API port for isolated lifecycle")
	}
	servicePath := filepath.Join(repo, "scenarios/landing-page-business-suite/.vrooli/service.json")
	var service map[string]any
	raw, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &service); err != nil {
		t.Fatal(err)
	}
	// A prebuilt API-only target profile: storage is the fixture's real isolated
	// PostgreSQL. Dependency startup is exercised separately at its compiled
	// owner boundary; the application still uses its complete production server.
	service["dependencies"] = map[string]any{}
	service["cli"] = map[string]any{"enabled": false}
	service["credentials"] = map[string]any{}
	service["hostTools"] = []any{}
	service["ports"] = map[string]any{"api": map[string]any{"env_var": "API_PORT", "port": port}}
	service["components"] = map[string]any{"api": service["components"].(map[string]any)["api"]}
	service["lifecycle"] = map[string]any{"version": "2.0.0", "health": map[string]any{"checks": []any{map[string]any{
		"name": "api_endpoint", "type": "http", "target": "http://127.0.0.1:${API_PORT}/health", "critical": true, "timeout": 1000}}, "startup_grace_period": 30000}}
	settings := service["environment"].(map[string]any)
	settings["DATABASE_URL"], settings["POSTGRES_URL"], settings["LPBS_ENVIRONMENT"] = fixture["database_url"], fixture["database_url"], "production"
	settings["LPBS_TEST_CREDENTIAL_FALLBACK"] = "0"
	serviceDir := filepath.Join(root, "scenarios/landing-page-business-suite")
	writeHostedJSON(t, filepath.Join(serviceDir, ".vrooli/service.json"), service)
	copyHostedFile(t, fixture["binary"], filepath.Join(serviceDir, "api/landing-page-business-suite-api"), 0755)
	copyHostedTree(t, filepath.Join(repo, "scenarios/landing-page-business-suite/config"), filepath.Join(serviceDir, "config"))

	// Select a real declared optional resource for this failure variant. The
	// committed ordinary hosted selection still excludes local/router resources.
	catalog, err := closure.NewRepoCatalog(repo)
	if err != nil {
		t.Fatal(err)
	}
	c, err := closure.Resolve(ctx, closure.Inputs{ScenarioID: "landing-page-business-suite", Environment: "production", RepoRoot: repo,
		Platform: closure.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}, Scope: closure.ScopeBundle, Catalog: catalog,
		HostRequirements: closure.ControlPlaneHostRequirements{}, Overrides: closure.Overrides{DeselectOptional: []string{"openrouter", "ollaya", "orcarouter", "abliteration"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !c.Supported() {
		t.Fatalf("optional failure variant unsupported: %+v", c.Unsupported)
	}
	var manifest domain.CloudManifest
	raw, err = os.ReadFile(filepath.Join(repo, "scenarios/landing-page-business-suite/cloud-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.Target.VPS.Workdir, manifest.Target.VPS.Host = root, "127.0.0.1"
	manifest.Bundle.Scenarios = []string{"landing-page-business-suite"}
	manifest.Dependencies.DeselectOptional = []string{"openrouter", "ollaya", "orcarouter", "abliteration"}
	// No public edge or fixed cloud ports on the isolated local target. The
	// lifecycle allocates the profile's API port and checks its actual listener.
	manifest.Ports = nil
	manifest.Edge = domain.ManifestEdge{}
	configDigest, err := ConfigurationDigest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	archive := archiveHostedTarget(t, root)
	release := cloudrelease.Manifest{SchemaVersion: cloudrelease.ManifestSchemaVersion, BundleSHA256: hostedSHA(t, archive),
		NativeCLI:     cloudrelease.NativeCLI{SHA256: hostedSHA(t, cli), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH},
		ClosureDigest: strings.TrimPrefix(c.Digest, "sha256:"), ConfigurationDigest: strings.TrimPrefix(configDigest, "sha256:"),
		Provenance: cloudrelease.Provenance{Builder: "scenario-to-cloud", Policy: cloudrelease.PolicyDevelopmentLocalUnsigned}, Limits: cloudrelease.DefaultLimits()}
	release.ReleaseDigest, err = cloudrelease.ComputeReleaseDigest(release)
	if err != nil {
		t.Fatal(err)
	}
	bundleDir := filepath.Join(root, ".vrooli/cloud/bundles", release.ReleaseDigest)
	bundlePath := filepath.Join(bundleDir, "bundle.tar.gz")
	copyHostedFile(t, archive, bundlePath, 0600)
	manifestPath := filepath.Join(bundleDir, cloudrelease.ManifestFileName)
	writeHostedJSON(t, manifestPath, release)
	var satisfied []string
	for _, component := range c.Components {
		if component.Kind == domain.ClosureKindCredentialDescriptor {
			satisfied = append(satisfied, component.ID)
		}
	}
	for _, secret := range manifest.Secrets.BundleSecrets {
		satisfied = append(satisfied, secret.ID)
	}
	dep := identity.DeploymentRef{ID: "j13-offline", ScenarioID: manifest.Scenario.ID, Environment: "production", Target: targetRef()}
	plan, err := execplan.Compile(ctx, execplan.CompileInputs{Manifest: manifest, Closure: &c, Deployment: dep, DesiredRevision: 1,
		Release:          identity.ReleaseRef{Digest: release.ReleaseDigest, ClosureDigest: c.Digest, ConfigurationDigest: configDigest},
		ReleaseArtifacts: execplan.ReleaseArtifacts{ID: release.ReleaseDigest, ManifestPath: manifestPath, NativeCLI: cli},
		ArtifactPath:     bundlePath, Scope: execplan.ScopeFull, Observations: execplan.Observations{SatisfiedInputs: satisfied}})
	if err != nil {
		t.Fatal(err)
	}
	e := &executor{manifest: manifest, rt: Runtime{Target: dep.Target, Identity: Identity{OperationID: "j13-offline", Fence: 1}},
		reach: &attributedReach{inner: target}, deploymentID: dep.ID}
	stage := plan.Action(execplan.OpReleaseStage)
	e.reach.enter(stage.ID)
	if _, err := e.runCommands(ctx, *stage); err != nil {
		t.Fatal(err)
	}

	// Compile the real Gateway dependency-start subplan. PostgreSQL was
	// established by the isolated storage owner above; this action covers the
	// selected optional resource and executes the complete production loop.
	gatewayClosure, err := closure.Resolve(ctx, closure.Inputs{ScenarioID: "ai-gateway", Environment: "production", RepoRoot: repo,
		Platform: closure.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}, Scope: closure.ScopeBundle, Catalog: catalog,
		HostRequirements: closure.ControlPlaneHostRequirements{}, Overrides: closure.Overrides{DeselectOptional: manifest.Dependencies.DeselectOptional}})
	if err != nil {
		t.Fatal(err)
	}
	gatewayManifest := manifest
	gatewayManifest.Scenario.ID = "ai-gateway"
	gatewayManifest.Dependencies, gatewayManifest.Bundle = closure.ManifestDependencies(gatewayClosure)
	gatewayManifest.Bundle.Scenarios = []string{"ai-gateway"}
	gatewayDeployment := dep
	gatewayDeployment.ScenarioID = "ai-gateway"
	dependencyPlan, err := execplan.Compile(ctx, execplan.CompileInputs{Manifest: gatewayManifest, Closure: &gatewayClosure,
		Deployment: gatewayDeployment, DesiredRevision: 1, Release: identity.ReleaseRef{Digest: release.ReleaseDigest, ClosureDigest: gatewayClosure.Digest, ConfigurationDigest: configDigest},
		ReleaseArtifacts: execplan.ReleaseArtifacts{ID: release.ReleaseDigest, ManifestPath: manifestPath, NativeCLI: cli},
		ArtifactPath:     bundlePath, Scope: execplan.ScopeStart, Observations: execplan.Observations{SatisfiedInputs: satisfied}})
	if err != nil {
		t.Fatal(err)
	}
	deps := dependencyPlan.Action(execplan.OpRuntimeStartDeps)
	if deps == nil || deps.Inputs["optional_resources"] != "ollama" {
		t.Fatal("compiler did not admit the selected optional resource")
	}
	// Fault injection at the real resource owner boundary: this target has
	// deliberately not received Ollama. Its real CLI refuses startup, and
	// runCommands must record degradation and continue to primary activation.
	e.reach.enter(deps.ID)
	detail, err := e.runCommands(ctx, *deps)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(detail, "degraded:") || !strings.Contains(detail, "ollama") {
		t.Fatalf("optional refusal was not recorded: %s", detail)
	}
	t.Log("real optional owner refusal recorded as degraded")
	activate := plan.Action(execplan.OpReleaseActivate)
	t.Cleanup(func() {
		result, err := target.Exec(context.Background(), dep.Target, reach.Command{Verb: "scenario stop", Args: []string{"landing-page-business-suite", "--json"}})
		if err != nil || result.ExitCode != 0 {
			t.Errorf("stop isolated LPBS: %+v %v", result, err)
		}
	})
	t.Cleanup(func() {
		if t.Failed() {
			for _, diagnostic := range []reach.Command{
				{Verb: "scenario status", Args: []string{"landing-page-business-suite", "--json"}},
				{Verb: "scenario logs", Args: []string{"landing-page-business-suite", "--runtime", "--tail", "50"}},
				{Verb: "scenario logs", Args: []string{"landing-page-business-suite", "--lifecycle", "--tail", "50"}},
				{Verb: "runtime supervisor status", Args: []string{"--json"}},
			} {
				result, err := target.Exec(ctx, dep.Target, diagnostic)
				t.Logf("%s: %+v; %v", diagnostic.Verb, result, err)
			}
		}
	})
	e.reach.enter(activate.ID)
	if _, err := e.runCommands(ctx, *activate); err != nil {
		t.Fatal(err)
	}
	listing, err := target.Exec(ctx, dep.Target, reach.Command{Verb: "cloud-target release list", Args: []string{"--deployment", dep.ID}})
	if err != nil || listing.ExitCode != 0 {
		t.Fatalf("release list: %+v %v", listing, err)
	}
	var state struct {
		Active *struct {
			ActiveRelease string `json:"active_release"`
		} `json:"active"`
		Releases []struct {
			Digest string `json:"digest"`
			Path   string `json:"path"`
		} `json:"releases"`
	}
	if err := json.Unmarshal([]byte(listing.Stdout), &state); err != nil {
		t.Fatal(err)
	}
	if state.Active == nil || state.Active.ActiveRelease != release.ReleaseDigest {
		t.Fatalf("real owner did not commit active release: %s", listing.Stdout)
	}
	releaseDir := ""
	for _, staged := range state.Releases {
		if staged.Digest == release.ReleaseDigest {
			releaseDir = staged.Path
		}
	}
	if releaseDir == "" {
		t.Fatalf("active release directory absent: %s", listing.Stdout)
	}
	portResult, err := target.Exec(ctx, dep.Target, reach.Command{Verb: "scenario port", Args: []string{"landing-page-business-suite", "api", "--path", filepath.Join(releaseDir, "scenarios/landing-page-business-suite"), "--json"}})
	if err != nil || portResult.ExitCode != 0 {
		t.Fatalf("lifecycle port: %+v %v", portResult, err)
	}
	var assigned struct {
		Port int `json:"port"`
	}
	if err := json.Unmarshal([]byte(portResult.Stdout), &assigned); err != nil {
		t.Fatal(err)
	}
	if assigned.Port == 0 {
		t.Fatalf("lifecycle did not allocate API port: %s", portResult.Stdout)
	}
	base := fmt.Sprintf("http://127.0.0.1:%d", assigned.Port)
	response, err := http.Post(base+"/api/v1/auth/magic-link", "application/json", strings.NewReader(`{"email":"hosted-journey@example.invalid"}`))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("sign-in status %d: %s", response.StatusCode, body)
	}
	response, err = http.Get(fixture["observer_url"])
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var evidence map[string]int
	if err := json.NewDecoder(response.Body).Decode(&evidence); err != nil {
		t.Fatal(err)
	}
	if evidence["queued_tokens"] != 1 || evidence["outbox_messages"] != 1 {
		t.Fatalf("sign-in persistence: %v", evidence)
	}
}

func startHostedFixture(t *testing.T, ctx context.Context, repo, root string, env []string) map[string]string {
	t.Helper()
	cmd := exec.CommandContext(ctx, "go", "test", ".", "-run", "^TestJourneyHostedSignInServer$", "-v", "-count=1", "-timeout", "8m")
	cmd.Dir = filepath.Join(repo, "scenarios/landing-page-business-suite/api")
	cmd.Env = append(env, "GOWORK=off", "VROOLI_JOURNEY_HOSTED_SIGNIN=1", "VROOLI_JOURNEY_HOSTED_ROOT="+root)
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	observerURL := ""
	t.Cleanup(func() {
		if observerURL != "" {
			response, err := http.Post(observerURL+"/__fixture/stop", "text/plain", nil)
			if err != nil {
				t.Error(err)
			} else {
				response.Body.Close()
			}
		}
		if err := cmd.Wait(); err != nil {
			t.Errorf("storage fixture: %v: %s", err, stderr.String())
		}
	})
	scanner := bufio.NewScanner(output)
	var startup strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "HOSTED_FIXTURE=") {
			var fixture map[string]string
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "HOSTED_FIXTURE=")), &fixture); err != nil {
				t.Fatal(err)
			}
			observerURL = fixture["observer_url"]
			go io.Copy(io.Discard, output)
			return fixture
		}
		startup.WriteString(line + "\n")
	}
	t.Fatalf("storage fixture did not start: %s; %s", startup.String(), stderr.String())
	return nil
}

func writeHostedJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
}
func copyHostedFile(t *testing.T, src, dst string, mode os.FileMode) {
	t.Helper()
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, raw, mode); err != nil {
		t.Fatal(err)
	}
}
func copyHostedTree(t *testing.T, src, dst string) {
	t.Helper()
	if err := filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		copyHostedFile(t, path, filepath.Join(dst, rel), 0600)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
func hostedSHA(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func archiveHostedTarget(t *testing.T, root string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bundle.tar.gz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || (!strings.Contains(path, string(filepath.Separator)+"scenarios"+string(filepath.Separator)) && !strings.HasSuffix(path, "/.vrooli/bin/vrooli")) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name, err = filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = tw.Write(raw)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	for _, closer := range []io.Closer{tw, gz, file} {
		if err := closer.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return path
}
