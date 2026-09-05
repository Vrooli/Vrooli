package platform

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// coreDefinitions builds every core unit for a target with fixture inputs
// rooted at home, the way setup does on a host. Executables are placed under
// home so systemd-analyze, which stats ExecStart=, can find them.
func coreDefinitions(t *testing.T, target, home string) []ServiceDefinition {
	t.Helper()
	join := filepath.Join
	if target == "windows" {
		join = windowsJoin
	}
	root := join(home, "Vrooli")
	loop := join(root, "scenarios", "vrooli-autoheal", "cli", "vrooli-autoheal-loop")
	vrooli := join(home, ".vrooli", "bin", "vrooli")
	watchdog := join(home, ".vrooli", "libexec", "vrooli-watchdog")
	if target == "windows" {
		loop, vrooli, watchdog = loop+".exe", vrooli+".exe", watchdog+".exe"
	}
	autoheal, err := WatchdogDefinition(target, WatchdogDefinitionOptions{Root: root, Home: home, LoopBinary: loop, VrooliBinary: vrooli, Username: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	supervisor, err := RuntimeSupervisorDefinition(target, RuntimeSupervisorOptions{Home: home, Executable: vrooli, SourceRoot: root, LogPath: join(home, ".vrooli", "logs", "runtime-supervisor.log")})
	if err != nil {
		t.Fatal(err)
	}
	supervisor.Username = "alice"
	emergency, err := EmergencyWatchdogDefinition(target, EmergencyWatchdogOptions{Home: home, Binary: watchdog, SetpointPath: join(root, "scenarios", "infrastructure-manager", "setpoint", "reliability-setpoint.json"), Interval: 5 * time.Minute, Username: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	return []ServiceDefinition{autoheal, supervisor, emergency}
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// renderCoreSystemd renders the four core units into one directory with real
// executables and returns the artifacts and their paths.
func renderCoreSystemd(t *testing.T, home string) ([]RenderedArtifact, []string) {
	t.Helper()
	dir := t.TempDir()
	var artifacts []RenderedArtifact
	var paths []string
	for _, d := range coreDefinitions(t, "linux", home) {
		writeExecutable(t, d.Executable)
		if d.WorkingDirectory != "" {
			if err := os.MkdirAll(d.WorkingDirectory, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		artifact, err := RenderSystemd(d)
		if err != nil {
			t.Fatal(err)
		}
		artifacts = append(artifacts, artifact)
		for _, file := range artifact.Files {
			path := filepath.Join(dir, file.Name)
			if err := os.WriteFile(path, []byte(file.Content), 0o644); err != nil {
				t.Fatal(err)
			}
			paths = append(paths, path)
		}
	}
	return artifacts, paths
}

// TestSystemdRenderPassesSystemdAnalyze is the contract that matters: systemd
// itself must accept every rendered core unit. Skipped where systemd-analyze
// is unavailable so the suite stays portable; on any host with systemd it
// fails loudly on a directive systemd will not load. Paths with spaces are
// the case most likely to regress the per-directive quoting.
func TestSystemdRenderPassesSystemdAnalyze(t *testing.T) {
	analyze, err := exec.LookPath("systemd-analyze")
	if err != nil {
		t.Skip("systemd-analyze is unavailable; skipping real-systemd unit verification")
	}
	for name, dirName := range map[string]string{"plain paths": "alice", "paths with spaces": "alice smith"} {
		t.Run(name, func(t *testing.T) {
			home := filepath.Join(t.TempDir(), dirName)
			_, paths := renderCoreSystemd(t, home)
			output, _ := exec.Command(analyze, append([]string{"--user", "verify"}, paths...)...).CombinedOutput()
			if findings := systemdFindings(string(output), paths); len(findings) > 0 {
				t.Fatalf("systemd rejected a rendered core unit:\n%s", strings.Join(findings, "\n"))
			}
			// The agent slice renders through the same validator.
			slice, err := RenderSystemd(agentSliceDefinition())
			if err != nil {
				t.Fatal(err)
			}
			if verdict := ValidateSystemd(slice, ScopeUser); verdict.State != VerdictAccepted {
				t.Fatalf("ValidateSystemd(vrooli-agents.slice) = %+v, want accepted", verdict)
			}
			// The validator must agree with the raw run above.
			for _, d := range coreDefinitions(t, "linux", home) {
				artifact, _ := RenderSystemd(d)
				if verdict := ValidateSystemd(artifact, ScopeUser); verdict.State != VerdictAccepted {
					t.Fatalf("ValidateSystemd(%s) = %+v, want accepted", d.Name, verdict)
				}
			}
		})
	}
}

func TestEveryCoreUnitDocumentationIsURL(t *testing.T) {
	for _, unit := range CoreUnits() {
		if _, err := os.Stat(filepath.Join("..", "..", unit.OwnerPath)); err != nil {
			t.Errorf("core unit %s owner path %s does not exist in the repository", unit.ID, unit.OwnerPath)
		}
	}
	artifacts, _ := renderCoreSystemd(t, filepath.Join(t.TempDir(), "alice"))
	for _, artifact := range artifacts {
		for _, file := range artifact.Files {
			var documented bool
			for _, line := range strings.Split(file.Content, "\n") {
				if value, ok := strings.CutPrefix(line, "Documentation="); ok {
					documented = true
					if !strings.HasPrefix(value, "https://github.com/Vrooli/Vrooli/blob/master/") {
						t.Errorf("%s: Documentation=%s is not a repository URL", file.Name, value)
					}
				}
			}
			if !documented {
				t.Errorf("%s has no Documentation= line", file.Name)
			}
		}
	}
}

// A user manager never reaches network-online.target, docker.service or
// multi-user.target; a user unit that orders after them waits forever, and
// one installed into multi-user.target is inactive after every boot. That
// was the bridge agent's state on 2026-09-02.
func TestUserScopeUnitsHaveNoSystemDependencies(t *testing.T) {
	artifacts, _ := renderCoreSystemd(t, filepath.Join(t.TempDir(), "alice"))
	for _, artifact := range artifacts {
		for _, file := range artifact.Files {
			for _, line := range strings.Split(file.Content, "\n") {
				for _, forbidden := range []string{"network-online.target", "docker.service", "docker.socket", "multi-user.target"} {
					if strings.Contains(line, forbidden) {
						t.Errorf("%s: user-scope unit references %s: %s", file.Name, forbidden, line)
					}
				}
				if strings.HasPrefix(line, "After=") {
					t.Errorf("%s: user-scope core unit must not order after anything: %s", file.Name, line)
				}
			}
			if strings.HasSuffix(file.Name, ".service") && strings.Contains(file.Content, "[Install]") && !strings.Contains(file.Content, "WantedBy=default.target\n") {
				t.Errorf("%s: user-scope daemon must be wanted by default.target:\n%s", file.Name, file.Content)
			}
		}
	}
}

func TestEveryCoreUnitCarriesTheGoToolchainOnPath(t *testing.T) {
	for _, target := range []string{"linux", "darwin"} {
		for _, d := range coreDefinitions(t, target, "/home/alice") {
			if !strings.Contains(d.Env["PATH"], "/usr/local/go/bin") {
				t.Errorf("%s/%s PATH %q lacks the Go toolchain; the recovery floor's `go mod download` cannot run", target, d.Name, d.Env["PATH"])
			}
		}
	}
	if !strings.Contains(DefaultPath("windows", `C:\Users\alice`), `WinGet\Links`) {
		t.Error("windows DefaultPath lacks the WinGet links directory")
	}
}

// The healer must survive the saturation it exists to report on. Without
// these the autoheal service is scheduled and reclaimed like any other
// process, and on 2026-08-19 it stopped responding at exactly the moment it
// was needed.
func TestAutohealLoopUnitSupervisionContract(t *testing.T) {
	d, err := WatchdogDefinition("linux", WatchdogDefinitionOptions{LoopBinary: "/home/u/loop", VrooliBinary: "/home/u/vrooli", Home: "/home/u", Root: "/home/u/Vrooli"})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := RenderSystemd(d)
	if err != nil {
		t.Fatal(err)
	}
	unit := artifact.Primary().Content
	service := unit[strings.Index(unit, "[Service]"):strings.Index(unit, "[Install]")]
	for _, directive := range []string{"CPUWeight=400", "MemoryMin=128M", "OOMScoreAdjust=-500", "Restart=on-failure", "RestartSec=15s", "TimeoutStopSec=30s", `ExecStart="/home/u/loop"`} {
		if !strings.Contains(service, directive) {
			t.Errorf("[Service] is missing %s:\n%s", directive, unit)
		}
	}
	for _, directive := range []string{"OnFailure=vrooli-emergency-watchdog.service", "StartLimitIntervalSec=300s", "StartLimitBurst=5"} {
		if !strings.Contains(unit[:strings.Index(unit, "[Service]")], directive) {
			t.Errorf("[Unit] is missing %s:\n%s", directive, unit)
		}
	}
}

func TestSystemdRenderQuotesPerDirective(t *testing.T) {
	d, err := RuntimeSupervisorDefinition("linux", RuntimeSupervisorOptions{Home: "/home/my tester", Executable: "/opt/vrooli bin/vrooli", SourceRoot: "/srv/my vrooli", LogPath: "/home/my tester/.vrooli/logs/runtime-supervisor.log"})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := RenderSystemd(d)
	if err != nil {
		t.Fatal(err)
	}
	content := artifact.Primary().Content
	for _, want := range []string{
		`Environment=HOME="/home/my tester"`,
		`Environment=VROOLI_SOURCE_ROOT="/srv/my vrooli"`,
		`ExecStart="/opt/vrooli bin/vrooli" runtime supervisor run`,
		"WorkingDirectory=/srv/my vrooli\n",
		"StandardOutput=append:/home/my tester/.vrooli/logs/runtime-supervisor.log\n",
		"StandardError=append:/home/my tester/.vrooli/logs/runtime-supervisor.log\n",
		"Restart=always\n",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("unit content missing %q:\n%s", want, content)
		}
	}
	if strings.Contains(content, `WorkingDirectory="`) || strings.Contains(content, `append:"`) {
		t.Fatalf("verbatim directives must not be quoted; systemd rejects the unit:\n%s", content)
	}
	if !strings.Contains(content, "\n[Install]\nWantedBy=default.target\n") {
		t.Fatalf("supervisor must be a default.target user unit:\n%s", content)
	}
}

func TestSystemdRenderOmitsDirectivesForEmptyFields(t *testing.T) {
	d, err := RuntimeSupervisorDefinition("linux", RuntimeSupervisorOptions{Home: "/home/tester", Executable: "/opt/vrooli/bin/vrooli", SourceRoot: "  ", LogPath: ""})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := RenderSystemd(d)
	if err != nil {
		t.Fatal(err)
	}
	content := artifact.Primary().Content
	for _, absent := range []string{"StandardOutput=", "StandardError=", "WorkingDirectory=", "VROOLI_SOURCE_ROOT"} {
		if strings.Contains(content, absent) {
			t.Errorf("expected no %s directive when its input is empty:\n%s", absent, content)
		}
	}
}

func TestTimerRendersServiceAndTimerUnit(t *testing.T) {
	d, err := EmergencyWatchdogDefinition("linux", EmergencyWatchdogOptions{Home: "/home/u", Binary: "/home/u/.vrooli/libexec/vrooli-watchdog", Interval: 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := RenderSystemd(d)
	if err != nil {
		t.Fatal(err)
	}
	service, ok := artifact.File("vrooli-emergency-watchdog.service")
	if !ok || !strings.Contains(service.Content, "Type=oneshot\n") || !strings.Contains(service.Content, `ExecStart="/home/u/.vrooli/libexec/vrooli-watchdog" --report-only --request-pressure`) {
		t.Fatalf("service unit wrong:\n%s", service.Content)
	}
	if strings.Contains(service.Content, "[Install]") {
		t.Fatalf("a timer-driven oneshot must not install itself:\n%s", service.Content)
	}
	timer, ok := artifact.File("vrooli-emergency-watchdog.timer")
	if !ok {
		t.Fatal("timer unit missing")
	}
	for _, want := range []string{"OnBootSec=120s", "OnUnitActiveSec=300s", "Persistent=true", "Unit=vrooli-emergency-watchdog.service", "WantedBy=timers.target"} {
		if !strings.Contains(timer.Content, want) {
			t.Errorf("timer unit missing %s:\n%s", want, timer.Content)
		}
	}
}

func TestValidateRejectsUnrenderableDefinitions(t *testing.T) {
	base, _ := WatchdogDefinition("linux", WatchdogDefinitionOptions{LoopBinary: "/loop", VrooliBinary: "/vrooli", Home: "/home/u", Root: "/home/u/Vrooli"})
	cases := map[string]func(d *ServiceDefinition){
		"empty name":          func(d *ServiceDefinition) { d.Name = "" },
		"relative executable": func(d *ServiceDefinition) { d.Executable = "bin/loop" },
		"newline in env":      func(d *ServiceDefinition) { d.Env["HOME"] = "/home/u\nExecStart=/bin/sh" },
		"newline in arg":      func(d *ServiceDefinition) { d.Args = []string{"--x\n"} },
		"non-url docs":        func(d *ServiceDefinition) { d.DocumentationURL = "internal/handler.go" },
		"timer without schedule": func(d *ServiceDefinition) {
			d.Kind = KindTimer
			d.Schedule = nil
		},
		"timer under a minute": func(d *ServiceDefinition) {
			d.Kind = KindTimer
			d.Schedule = &Schedule{Every: 30 * time.Second}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			d := base
			d.Env = map[string]string{}
			for key, value := range base.Env {
				d.Env[key] = value
			}
			mutate(&d)
			if err := d.Validate(); err == nil {
				t.Fatal("Validate accepted an unrenderable definition")
			}
		})
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("the base definition must validate: %v", err)
	}
	if _, err := RenderWindowsTaskXML(base); err == nil {
		t.Fatal("windows renderer accepted a definition without a principal")
	}
}

func TestSystemScopeRendersSystemTargets(t *testing.T) {
	d, err := WatchdogDefinition("linux", WatchdogDefinitionOptions{LoopBinary: "/loop", VrooliBinary: "/vrooli", Home: "/root", Root: "/opt/Vrooli", SystemService: true})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := RenderSystemd(d)
	if err != nil {
		t.Fatal(err)
	}
	content := artifact.Primary().Content
	for _, want := range []string{"WantedBy=multi-user.target", "\nUser=root\n", "After=network-online.target"} {
		if !strings.Contains(content, want) {
			t.Errorf("system unit missing %s:\n%s", want, content)
		}
	}
}

// Regression: the launchd plist must log where the darwin log reader
// (service_darwin.go nativeServiceLogs) looks, or `vrooli` reports an empty
// log for a running agent.
func TestLaunchdWatchdogLogPathMatchesLogReader(t *testing.T) {
	home := "/Users/alice"
	artifact, err := RenderWatchdogArtifact("darwin", WatchdogDefinitionOptions{Root: home + "/Vrooli", Home: home, LoopBinary: home + "/Vrooli/scenarios/vrooli-autoheal/cli/vrooli-autoheal-loop", VrooliBinary: home + "/.vrooli/bin/vrooli"})
	if err != nil {
		t.Fatal(err)
	}
	plistPath := LaunchAgentPath(home, "com.vrooli.autoheal")
	if filepath.Base(plistPath) != artifact.Primary().Name {
		t.Fatalf("plist file %q is not where the agent path says (%q)", artifact.Primary().Name, plistPath)
	}
	reader := LaunchdLogPath(plistPath, "com.vrooli.autoheal")
	want := "<key>StandardOutPath</key>\n  <string>" + reader + "</string>"
	if !strings.Contains(artifact.Primary().Content, want) {
		t.Fatalf("plist logs somewhere the log reader does not look; want %q in:\n%s", want, artifact.Primary().Content)
	}
}

// Fixture provenance: every darwin and windows fixture under
// testdata/servicedef was derived 2026-09-02 on swarminator (Linux, systemd
// 255) from platformgo.RenderLaunchd / RenderWindowsTaskXML through this
// test's coreDefinitions inputs. None is host-verified on macOS or Windows;
// they pin the rendered shape so a renderer change is a visible diff, not a
// proof that launchd or Task Scheduler accepts them. Regenerate with
// PLATFORM_GO_UPDATE_FIXTURES=1 and review the diff.
func TestDarwinAndWindowsRenderersMatchFixtures(t *testing.T) {
	for _, target := range []string{"darwin", "windows"} {
		home := "/Users/alice"
		if target == "windows" {
			home = `C:\Users\alice`
		}
		for _, d := range coreDefinitions(t, target, home) {
			if target == "windows" && d.Name == "vrooli-runtime-supervisor" {
				// The supervisor is an SCM service on Windows; its command line
				// is covered by TestWindowsServiceCommandLine.
				continue
			}
			artifact, err := RenderDefinition(d, target)
			if err != nil {
				t.Fatalf("%s/%s: %v", target, d.Name, err)
			}
			if verdict := ValidateArtifact(artifact, d.Scope); verdict.Rejected() {
				t.Fatalf("%s/%s: validator rejected the render: %s", target, d.Name, verdict.Output)
			}
			for _, file := range artifact.Files {
				fixture := filepath.Join("testdata", "servicedef", target+"."+file.Name)
				if os.Getenv("PLATFORM_GO_UPDATE_FIXTURES") == "1" {
					if err := os.WriteFile(fixture, []byte(fixtureHeader(target)+file.Content), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				raw, err := os.ReadFile(fixture)
				if err != nil {
					t.Fatalf("%s: %v (set PLATFORM_GO_UPDATE_FIXTURES=1 to regenerate)", fixture, err)
				}
				want := strings.TrimPrefix(string(raw), fixtureHeader(target))
				if want != file.Content {
					t.Errorf("%s drifted from its fixture; rendered:\n%s\nfixture:\n%s", file.Name, file.Content, want)
				}
			}
		}
	}
}

// fixtureHeader is the provenance line each fixture carries. It sits before
// the XML declaration, so the fixture is not itself a loadable document; the
// test strips it before comparing.
func fixtureHeader(target string) string {
	return "<!-- Fixture derived 2026-09-02 on swarminator (Linux) from platform-go RenderDefinition for target " + target + "; not host-verified on macOS or Windows. Regenerate: PLATFORM_GO_UPDATE_FIXTURES=1 go test -run TestDarwinAndWindowsRenderersMatchFixtures -->\n"
}

func TestWindowsServiceCommandLine(t *testing.T) {
	d, err := RuntimeSupervisorDefinition("windows", RuntimeSupervisorOptions{Home: `C:\Users\alice`, Executable: `C:\Users\alice\.vrooli\bin\vrooli.exe`, SourceRoot: `C:\Users\alice\Vrooli`})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := WindowsServiceCommandLine(d), `C:\Users\alice\.vrooli\bin\vrooli.exe runtime supervisor run`; got != want {
		t.Fatalf("command line = %q, want %q", got, want)
	}
	d.Executable = `C:\Program Files\Vrooli\vrooli.exe`
	if got := WindowsServiceCommandLine(d); !strings.HasPrefix(got, `"C:\Program Files\Vrooli\vrooli.exe" `) {
		t.Fatalf("executable with spaces must be quoted: %q", got)
	}
}

func TestValidateWindowsTaskRejectsEmptyPrincipal(t *testing.T) {
	d, _ := WatchdogDefinition("windows", WatchdogDefinitionOptions{LoopBinary: `C:\loop.exe`, VrooliBinary: `C:\vrooli.exe`, Home: `C:\Users\alice`, Root: `C:\Users\alice\Vrooli`, Username: "alice"})
	artifact, err := RenderWindowsTaskXML(d)
	if err != nil {
		t.Fatal(err)
	}
	if verdict := ValidateWindowsTask(artifact); verdict.State != VerdictAccepted {
		t.Fatalf("a rendered task must validate: %+v", verdict)
	}
	blank := RenderedArtifact{Target: "windows", Files: []RenderedFile{{Name: "x.xml", Content: strings.Replace(artifact.Primary().Content, "<UserId>alice</UserId>", "<UserId></UserId>", 1)}}}
	if verdict := ValidateWindowsTask(blank); !verdict.Rejected() || !strings.Contains(verdict.Output, "UserId") {
		t.Fatalf("empty principal must be rejected: %+v", verdict)
	}
	foreign := RenderedArtifact{Target: "windows", Files: []RenderedFile{{Name: "x.xml", Content: strings.Replace(artifact.Primary().Content, "<Hidden>false</Hidden>", "<Bogus>1</Bogus>", 1)}}}
	if verdict := ValidateWindowsTask(foreign); !verdict.Rejected() || !strings.Contains(verdict.Output, "Bogus") {
		t.Fatalf("an element outside the schema must be rejected: %+v", verdict)
	}
}

func TestValidateLaunchdRejectsMalformedPlist(t *testing.T) {
	broken := RenderedArtifact{Target: "darwin", Files: []RenderedFile{{Name: "x.plist", Content: "<plist><dict><key>Label</key></dict>"}}}
	if verdict := ValidateLaunchd(broken); !verdict.Rejected() {
		t.Fatalf("malformed plist must be rejected everywhere: %+v", verdict)
	}
	d, _ := WatchdogDefinition("darwin", WatchdogDefinitionOptions{LoopBinary: "/loop", VrooliBinary: "/vrooli", Home: "/Users/alice", Root: "/Users/alice/Vrooli"})
	artifact, err := RenderLaunchd(d)
	if err != nil {
		t.Fatal(err)
	}
	verdict := ValidateLaunchd(artifact)
	if _, err := exec.LookPath("plutil"); err != nil {
		if verdict.State != VerdictUnavailable {
			t.Fatalf("without plutil a well-formed plist is unproven, not accepted: %+v", verdict)
		}
	} else if verdict.State != VerdictAccepted {
		t.Fatalf("plutil rejected the rendered plist: %+v", verdict)
	}
}

func TestValidateSystemdRejectsUnknownDirective(t *testing.T) {
	if _, err := exec.LookPath("systemd-analyze"); err != nil {
		t.Skip("systemd-analyze is unavailable")
	}
	bad := RenderedArtifact{Target: "linux", Files: []RenderedFile{{Name: "vrooli-servicedef-probe.service", Content: "[Unit]\nDescription=probe\n\n[Service]\nBogusDirective=1\nExecStart=/bin/true\n"}}}
	verdict := ValidateSystemd(bad, ScopeUser)
	if !verdict.Rejected() || !strings.Contains(verdict.Output, "Bogus") {
		t.Fatalf("unknown directive must be rejected: %+v", verdict)
	}
	missing := RenderedArtifact{Target: "linux", Files: []RenderedFile{{Name: "vrooli-servicedef-probe.service", Content: "[Unit]\nDescription=probe\n\n[Service]\nExecStart=/nonexistent/vrooli-servicedef-probe\n"}}}
	if verdict := ValidateSystemd(missing, ScopeUser); !verdict.Rejected() {
		t.Fatalf("missing executable must be rejected: %+v", verdict)
	}
}

func TestVerdictIsJSONEvidence(t *testing.T) {
	raw, err := json.Marshal(Verdict{State: VerdictRejected, Validator: "systemd-analyze verify", Output: "x.service:3: Unknown key"})
	if err != nil {
		t.Fatal(err)
	}
	var back Verdict
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.State != VerdictRejected || !back.Rejected() || back.Validator == "" {
		t.Fatalf("verdict did not round-trip: %s", raw)
	}
}

func TestCoreUnitsAreTheOnlyUnitList(t *testing.T) {
	units := CoreUnits()
	if len(units) != 4 {
		t.Fatalf("expected four core units, got %d", len(units))
	}
	if got := CoreDaemonUnits(); strings.Join(got, ",") != "vrooli-autoheal.service,vrooli-runtime-supervisor.service" {
		t.Fatalf("daemon units = %v", got)
	}
	if got := CoreSystemdUnits(); len(got) != 4 || got[3] != "vrooli-emergency-watchdog.timer" {
		t.Fatalf("systemd units = %v", got)
	}
	if unit, ok := CoreUnitByID(CoreUnitAutohealLoop); !ok || unit.NativeName("darwin") != "com.vrooli.autoheal" || unit.NativeName("windows") != "VrooliAutoheal" {
		t.Fatalf("autoheal identity = %+v", unit)
	}
}

func TestNormalizeTargetAcceptsProductVocabulary(t *testing.T) {
	for token, want := range map[string]string{"linux": "linux", "macos": "darwin", "darwin": "darwin", "Windows": "windows"} {
		if got, err := NormalizeTarget(token); err != nil || got != want {
			t.Errorf("NormalizeTarget(%q) = %q, %v", token, got, err)
		}
	}
	if _, err := NormalizeTarget("plan9"); err == nil {
		t.Error("unknown target accepted")
	}
}
