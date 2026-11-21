//go:build journey

package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/cli-core/upstreamcheck"
	harnesslifecycle "github.com/vrooli/vrooli/packages/harness-lifecycle"
)

// These journeys serve the production UI against the production API and its
// SQLite stores. No browser routes or product services are replaced.
func launcherJourney(t *testing.T, check string) {
	t.Helper()
	evidence := os.Getenv("LAUNCHER_JOURNEY_EVIDENCE")
	if evidence == "" {
		t.Fatal("LAUNCHER_JOURNEY_EVIDENCE must name the isolated served-build directory")
	}
	dist := filepath.Join(evidence, "ui-dist")
	if _, err := os.Stat(filepath.Join(dist, "index.html")); err != nil {
		t.Fatalf("build the production UI into %s first: %v", dist, err)
	}
	pythonSite, err := exec.Command("python3", "-c", "import site; print(site.getusersitepackages())").Output()
	if err != nil {
		t.Fatal(err)
	}
	goBinary, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	goCaches, err := exec.Command(goBinary, "env", "GOCACHE", "GOMODCACHE").Output()
	if err != nil {
		t.Fatal(err)
	}
	fullStackJourney := check == "TestJ8_FullDeploymentRouteOperated"
	allHarnessJourney := fullStackJourney || check == "TestJ8_InstallUpdateUninstallDoctorPerHarness"
	desktopJourney := !fullStackJourney && (allHarnessJourney || check == "TestJ8_DesktopUserOwnedInstallOperated")
	updateJourney := check == "TestJ7_ManageUpdateWorksOnIsolatedHome"
	cleanupJourney := check == "TestJ8_FoundCopyStaysUntouched" || check == "TestJ8_CleanupPreviewEqualsRemoval"
	usageJourney := check == "TestJ7_UsageWindowsComeFromDataWithResetTimes"
	manageJourney := fullStackJourney || desktopJourney || cleanupJourney || updateJourney || check == "TestJ7_ManageShowsBubbleWhenActionWaits" || check == "TestJ7_EligibilityReasonShownAndFixable"
	var governanceCLI string
	if fullStackJourney {
		governanceCLI, err = exec.LookPath("scenario-dependency-analyzer")
		if err != nil {
			t.Fatal(err)
		}
	}
	oldSources := map[string]string{}
	if allHarnessJourney {
		realHome, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		for resource, version := range map[string]string{"claude-code": "2.1.312", "codex": "0.165.0", "opencode": "1.19.4"} {
			oldSources[resource] = filepath.Join(realHome, ".local/share/vrooli/resources", resource, "cli", version+"-linux-amd64")
			if _, err := os.Stat(filepath.Join(oldSources[resource], "installation.json")); err != nil {
				t.Fatal(err)
			}
		}
	}
	var sourcePrefix string
	if (manageJourney || usageJourney) && !desktopJourney && !allHarnessJourney {
		sourcePrefix = installedClaudePrefix(t)
	}
	home, err := os.MkdirTemp(evidence, "home-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_HOME", filepath.Join(home, ".vrooli"))
	t.Setenv("WC_TMUX_SOCKET", filepath.Join(home, "tmux.sock"))
	t.Setenv("WC_SESSION_STATE_ROOT", filepath.Join(home, "sessions"))
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
	t.Setenv("GROK_HOME", filepath.Join(home, ".grok"))
	windowPath := filepath.Join(home, "claude-windows.json")
	t.Setenv("VROOLI_CLAUDE_MODEL_WINDOWS", windowPath)
	if check == "TestJ7_ContextChoicesShowRealNumbersPerModel" {
		data, err := os.ReadFile("../test/launcher_context_windows.json")
		if err != nil {
			t.Fatal(err)
		}
		var windows struct{ Codex, Claude json.RawMessage }
		if err := json.Unmarshal(data, &windows); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, ".codex", "models_cache.json"), windows.Codex, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(windowPath, windows.Claude, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, class := range []string{"config", "data", "cache", "logs", "state"} {
		t.Setenv("VROOLI_"+strings.ToUpper(class)+"_ROOT", filepath.Join(home, class))
	}
	for _, class := range []string{"CONFIG", "DATA", "STATE", "CACHE"} {
		t.Setenv("XDG_"+class+"_HOME", filepath.Join(home, "xdg-"+strings.ToLower(class)))
	}
	t.Setenv("AUDIO_TOOLS_URL", "http://127.0.0.1:1")
	var localToken string
	if manageJourney || usageJourney {
		prepareUnlinkedHarness(t, home, sourcePrefix)
		if desktopJourney {
			isolateDesktopTools(t, home)
		}
		if fullStackJourney {
			t.Setenv("VROOLI_DESKTOP_MODE", "false")
			t.Setenv("VROOLI_BIN", filepath.Join(evidence, "vrooli"))
			isolateDesktopTools(t, home)
			for _, name := range []string{"node", "npm", "bwrap", "systemd-run", "systemctl", "busctl", "prlimit"} {
				if err := os.Symlink("/usr/bin/"+name, filepath.Join(home, "tools", name)); err != nil {
					t.Fatal(err)
				}
			}
			bin := filepath.Join(home, ".vrooli", "bin")
			if err := os.MkdirAll(bin, 0o700); err != nil {
				t.Fatal(err)
			}
			if out, err := exec.Command("/bin/cp", "-L", governanceCLI, filepath.Join(bin, "scenario-dependency-analyzer")).CombinedOutput(); err != nil {
				t.Fatalf("stage real governance CLI: %v: %s", err, out)
			}
			root, err := filepath.Abs("../../..")
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("LAUNCHER_GOVERNANCE_ROOT", root)
			owner := launcherServiceOwner(t, home, strings.Fields(string(goCaches)), goBinary, "../../scenario-dependency-analyzer/api", "./internal/dependencygovernance", "TestLauncherGovernanceOwner", "LAUNCHER_GOVERNANCE_URL=")
			t.Setenv("SCENARIO_DEPENDENCY_ANALYZER_API_BASE", owner)
		}
		if updateJourney {
			prepareOldAntigravity(t, home)
		}
		if cleanupJourney {
			prepareFoundClaude(t, home, sourcePrefix)
		}
		if manageJourney {
			localToken = fmt.Sprintf("%x", makeLocalToken(t))
			t.Setenv("LAUNCHER_LOCAL_TOKEN", localToken)
		}
		if usageJourney {
			// Keep Claude available in New Session while giving Manage a stable
			// unlinked managed copy with its link action, independent of host PATH.
			prepareFoundClaude(t, home, sourcePrefix)
		}
	}
	db, err := sql.Open("sqlite", resolveSQLiteDSN())
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	srv := NewServer(database.NewFromPrimary(db))
	if check == "TestJ7_UsageWindowsComeFromDataWithResetTimes" {
		srv.agentUsageURL = launcherQuotaOwner(t, home, strings.Fields(string(goCaches)))
		t.Setenv("LAUNCHER_QUOTA_URL", srv.agentUsageURL)
	}
	t.Cleanup(func() {
		srv.sweeper.Stop()
		srv.sessions.StopReattachWatchdog()
		srv.sessions.Shutdown()
		if srv.codexTailer != nil {
			srv.codexTailer.Stop()
		}
		if srv.claudeTailer != nil {
			srv.claudeTailer.Stop()
		}
		if srv.grokTailer != nil {
			srv.grokTailer.Stop()
		}
		if srv.opencodeWatcher != nil {
			srv.opencodeWatcher.Stop()
		}
		_ = db.Close()
	})
	api := srv.Handler()
	if manageJourney {
		tokenFile := filepath.Join(home, "local-session-token")
		if err := os.WriteFile(tokenFile, []byte(localToken), 0o600); err != nil {
			t.Fatal(err)
		}
		api = authn.Middleware(authn.Config{Providers: []authn.Provider{authn.NewPersonalLocalProviderWithTokenFile(tokenFile)}})(api)
	}
	files := http.FileServer(http.Dir(dist))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if allHarnessJourney && r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/__journey/old/") {
			resource := strings.TrimPrefix(r.URL.Path, "/__journey/old/")
			if resource == "grok" {
				cmd := exec.Command(goBinary, "test", "./packages/harness-lifecycle", "-run", "^TestHarnessOldGrokJourneyFixture$", "-count=1", "-v", "-timeout=5m")
				cmd.Dir = "../../.."
				cmd.Env = append(os.Environ(), "GOWORK=off", "HARNESS_OLD_GROK_JOURNEY_ROOT="+filepath.Join(home, "bundle/catalog"))
				if out, err := cmd.CombinedOutput(); err != nil {
					http.Error(w, string(out), 500)
					return
				}
				srv.agentUpdates.invalidate("local", "grok")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			source, ok := oldSources[resource]
			if !ok {
				http.NotFound(w, r)
				return
			}
			data, err := os.ReadFile(filepath.Join(source, "installation.json"))
			var receipt struct{ Version string }
			if err == nil {
				err = json.Unmarshal(data, &receipt)
			}
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			prefix, err := harnesslifecycle.NPMInstallPrefix(home, resource, receipt.Version)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if out, err := exec.Command("/bin/cp", "-a", "--reflink=auto", source, prefix).CombinedOutput(); err != nil {
				http.Error(w, string(out), 500)
				return
			}
			h, err := harnesslifecycle.LoadHarness(filepath.Join(home, "bundle/catalog"), resource)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			oldExecutable := filepath.Join(prefix, h.Executable)
			if h.NativeArtifact != nil {
				native := filepath.Join(prefix, "node_modules", h.NativeArtifact.Dependency, h.NativeArtifact.Executable)
				if _, err := os.Stat(native); err == nil {
					oldExecutable = native
				}
			}
			if err := os.Symlink(oldExecutable, filepath.Join(home, ".local/bin", h.Binary)); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			id := resource
			if id == "claude-code" {
				id = "claude"
			}
			srv.agentUpdates.invalidate("local", id)
			w.WriteHeader(204)
			return
		}
		if cleanupJourney && r.Method == http.MethodPost && r.URL.Path == "/__journey/reset-managed" {
			data, err := os.ReadFile(filepath.Join(sourcePrefix, "installation.json"))
			var receipt struct{ Version string }
			if err == nil {
				err = json.Unmarshal(data, &receipt)
			}
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			prefix, err := harnesslifecycle.NPMInstallPrefix(home, "claude-code", receipt.Version)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if _, err = os.Stat(prefix); os.IsNotExist(err) {
				if out, err := exec.Command("/bin/cp", "-a", "--reflink=auto", sourcePrefix, prefix).CombinedOutput(); err != nil {
					http.Error(w, string(out), 500)
					return
				}
			}
			srv.agentUpdates.invalidate("local", "claude")
			w.WriteHeader(204)
			return
		}
		if (updateJourney || allHarnessJourney) && r.Method == http.MethodPost && r.URL.Path == "/__journey/old-release" {
			if allHarnessJourney || fullStackJourney {
				if err := seedOldAntigravity(r.Context(), home); err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
			}
			prefix, err := harnesslifecycle.NPMInstallPrefix(home, "antigravity", "1.1.22")
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			link := filepath.Join(home, ".local", "bin", "agy")
			if _, err := os.Readlink(link); err == nil {
				_ = os.Remove(link)
			}
			if err := os.Symlink(filepath.Join(prefix, "agy"), link); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			srv.agentUpdates.invalidate("local", "agy")
			w.WriteHeader(204)
			return
		}
		if manageJourney && r.Method == http.MethodPost && r.URL.Path == "/__journey/unlink" {
			path := filepath.Join(home, ".local", "bin", "claude")
			if _, err := os.Readlink(path); err == nil {
				if err := os.Remove(path); err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
			}
			srv.agentUpdates.invalidate("local", "claude")
			w.WriteHeader(204)
			return
		}
		// Delay actual owner responses to expose loading transitions without
		// substituting fixture JSON for any product service.
		if check == "TestJ7_CardsDoNotShiftOnLoad" && (strings.HasSuffix(r.URL.Path, "/AgentUsage") || strings.HasSuffix(r.URL.Path, "/ListHarnesses")) {
			select {
			case <-time.After(2 * time.Second):
			case <-r.Context().Done():
				return
			}
		}
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/vrooli.") || r.URL.Path == "/health" {
			api.ServeHTTP(w, r)
			return
		}
		files.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	script := "../test/launcher_journey.py"
	if strings.HasPrefix(check, "TestJ14_") {
		script = "../test/integrations_journey.py"
	}
	cmd := exec.CommandContext(t.Context(), "python3", script, server.URL, check, evidence)
	// Keep the driver's installed Python package discovery while its browser
	// profile and all product state remain inside this isolated home.
	cmd.Env = append(os.Environ(), "PYTHONPATH="+strings.TrimSpace(string(pythonSite)))
	output, err := cmd.CombinedOutput()
	t.Log(string(output))
	if err != nil {
		t.Fatalf("served browser journey: %v", err)
	}
}

func TestJ7_RawTerminalShowsNoAgentControls(t *testing.T) {
	launcherJourney(t, t.Name())
}

func TestJ7_CardOrderChosenOrRecentSurvivesReload(t *testing.T) {
	launcherJourney(t, t.Name())
}

func TestJ7_CardsDoNotShiftOnLoad(t *testing.T) {
	launcherJourney(t, t.Name())
}

func TestJ7_ContextChoicesShowRealNumbersPerModel(t *testing.T) {
	launcherJourney(t, t.Name())
}

func TestJ7_UsageWindowsComeFromDataWithResetTimes(t *testing.T) {
	launcherJourney(t, t.Name())
}

func TestJ7_ManageUpdateWorksOnIsolatedHome(t *testing.T) { launcherJourney(t, t.Name()) }

func TestJ7_ManageShowsBubbleWhenActionWaits(t *testing.T) { launcherJourney(t, t.Name()) }
func TestJ7_EligibilityReasonShownAndFixable(t *testing.T) { launcherJourney(t, t.Name()) }

func makeLocalToken(t *testing.T) []byte {
	t.Helper()
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}
	return data
}

func installedClaudePrefix(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(home, ".local", "bin", "claude")
	path, err := filepath.EvalSymlinks(binary)
	if err != nil {
		t.Fatal(err)
	}
	for dir := filepath.Dir(path); dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "installation.json")); err == nil {
			return dir
		}
	}
	t.Fatal("a receipt-owned real Claude installation is required for this journey")
	return ""
}

func prepareUnlinkedHarness(t *testing.T, home, source string) {
	t.Helper()
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(home, "bundle")
	for _, resource := range []string{"codex", "claude-code", "opencode", "grok", "antigravity"} {
		dir := filepath.Join(bundle, "catalog", "resources", resource)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"resource.json", "model-policy.json"} {
			data, err := os.ReadFile(filepath.Join(root, "resources", resource, name))
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	if source != "" {
		data, err := os.ReadFile(filepath.Join(source, "installation.json"))
		if err != nil {
			t.Fatal(err)
		}
		var receipt struct{ Version string }
		if err := json.Unmarshal(data, &receipt); err != nil {
			t.Fatal(err)
		}
		prefix, err := harnesslifecycle.NPMInstallPrefix(home, "claude-code", receipt.Version)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(prefix), 0o700); err != nil {
			t.Fatal(err)
		}
		if output, err := exec.Command("/bin/cp", "-a", "--reflink=auto", source, prefix).CombinedOutput(); err != nil {
			t.Fatalf("copy actual installed harness: %v: %s", err, output)
		}
	}
	bin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":/usr/bin:/bin")
	t.Setenv("BUNDLE_ROOT", bundle)
	t.Setenv("VROOLI_DESKTOP_MODE", "true")
}

func launcherQuotaOwner(t *testing.T, home string, caches []string) string {
	t.Helper()
	t.Setenv("LAUNCHER_QUOTA_HOME", home)
	return launcherServiceOwner(t, home, caches, "go", "../../agent-manager/api", "./internal/handlers", "TestLauncherQuotaOwner", "LAUNCHER_QUOTA_URL=")
}

func launcherServiceOwner(t *testing.T, home string, caches []string, goBinary, dir, pkg, test, readyPrefix string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 18*time.Minute)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, goBinary, "test", "-tags=journey", "-v", "-count=1", "-timeout=18m", "-run=^"+test+"$", pkg)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOCACHE="+caches[0], "GOMODCACHE="+caches[1], "GOWORK=off")
	log, err := os.Create(filepath.Join(home, test+".log"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	cmd.Stderr = log
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	ready := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			_, _ = fmt.Fprintln(log, line)
			if strings.HasPrefix(line, readyPrefix) {
				ready <- strings.TrimPrefix(line, readyPrefix)
			}
		}
		close(ready)
	}()
	select {
	case address := <-ready:
		if address == "" {
			data, _ := os.ReadFile(log.Name())
			t.Fatalf("%s startup: %s", test, data)
		}
		t.Cleanup(func() {
			response, _ := http.Get(address + "/__journey/stop")
			if response != nil {
				_ = response.Body.Close()
			}
			if err := cmd.Wait(); err != nil {
				t.Errorf("%s: %v", test, err)
			}
		})
		return address
	case <-time.After(2 * time.Minute):
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		data, _ := os.ReadFile(log.Name())
		t.Fatalf("%s startup timed out: %s", test, data)
	}
	return ""
}

// Seed an actual older upstream release, with the repository's independently
// pinned archive checksum. The UI must acquire and select the current release.
func prepareOldAntigravity(t *testing.T, home string) {
	t.Helper()
	if err := seedOldAntigravity(t.Context(), home); err != nil {
		t.Fatal(err)
	}
}

func seedOldAntigravity(ctx context.Context, home string) error {
	root := filepath.Join(home, "bundle", "catalog")
	h, err := harnesslifecycle.LoadHarness(root, "antigravity")
	if err != nil {
		return err
	}
	artifact, err := h.Acquisition.Resolve(binaryfetch.Facts{"os": "linux", "arch": "amd64"})
	if err != nil {
		return err
	}
	prefix, err := harnesslifecycle.NPMInstallPrefix(home, "antigravity", "1.1.22")
	if err != nil {
		return err
	}
	path, err := binaryfetch.Fetch(ctx, binaryfetch.Target{Name: "agy", URL: artifact.URL, SHA256: artifact.SHA256, Archive: artifact.Archive, BinPath: artifact.BinPath}, prefix, nil)
	if err != nil {
		return err
	}
	raw, err := harnesslifecycle.InstalledVersion(ctx, path)
	if err != nil || upstreamcheck.ExtractSemver(raw) != "1.1.22" {
		return fmt.Errorf("older release identity: %q %v", raw, err)
	}
	digest, err := binaryfetch.FileDigestContext(ctx, path)
	if err != nil {
		return err
	}
	record, _ := json.Marshal(map[string]string{"Resource": "antigravity", "Package": "", "Version": "1.1.22", "FileSHA256": digest})
	if err := os.WriteFile(filepath.Join(prefix, "installation.json"), record, 0o600); err != nil {
		return err
	}
	return nil
}

func TestJ8_FoundCopyStaysUntouched(t *testing.T)     { launcherJourney(t, t.Name()) }
func TestJ8_CleanupPreviewEqualsRemoval(t *testing.T) { launcherJourney(t, t.Name()) }

func prepareFoundClaude(t *testing.T, home, source string) {
	t.Helper()
	sourceBinary := filepath.Join(source, "node_modules", "@anthropic-ai", "claude-code-linux-x64", "claude")
	foundDir := filepath.Join(home, "found")
	if err := os.MkdirAll(foundDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("/bin/cp", "-a", "--reflink=auto", sourceBinary, filepath.Join(foundDir, "claude")).CombinedOutput(); err != nil {
		t.Fatalf("copy independent runner: %v: %s", err, out)
	}
	t.Setenv("PATH", foundDir+":"+os.Getenv("PATH"))
}

func TestJ8_SixthHarnessByData(t *testing.T) {
	TestSixthHarnessReachesTheLauncherFromItsResourceDirectory(t)
}

func TestJ8_HarnessListLintPasses(t *testing.T) {
	command := exec.CommandContext(t.Context(), "go", "test", "github.com/vrooli/cli-core/agentcatalog/harness", "-count=1", "-run=^Test(BuiltinSnapshotMatchesResources|EveryRunnableResourceDeclaresAHarness|LoadRejectsAnUnsafeOrAmbiguousDescriptor)$")
	output, err := command.CombinedOutput()
	t.Log(string(output))
	if err != nil {
		t.Fatalf("harness descriptor lint: %v", err)
	}
}

func TestJ8_DesktopUserOwnedInstallOperated(t *testing.T) { launcherJourney(t, t.Name()) }

func TestJ8_InstallUpdateUninstallDoctorPerHarness(t *testing.T) { launcherJourney(t, t.Name()) }

func isolateDesktopTools(t *testing.T, home string) {
	t.Helper()
	dir := filepath.Join(home, "tools")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sh", "bash", "python3", "tmux", "uname", "env", "readlink", "dirname", "cat", "mkdir", "rm", "grep", "sed", "find", "head", "date", "realpath", "pwd", "which", "ps"} {
		source := "/usr/bin/" + name
		if _, err := os.Stat(source); err != nil {
			continue
		}
		if err := os.Symlink(source, filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", filepath.Join(home, ".local/bin")+":"+dir)
}

func TestJ8_FullDeploymentRouteOperated(t *testing.T) { launcherJourney(t, t.Name()) }
