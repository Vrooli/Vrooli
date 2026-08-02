//go:build linux

package securestore

import (
	"os"
	"strings"
	"testing"
)

// The repair reaches into the operator's keyring, so the refusal cases are the
// ones that matter most: a repair that fires when it should not is a path to
// the wrong user's secrets, and no error message would reveal it.

const (
	testUID    = 1000
	testDir    = "/run/user/1000"
	testBus    = "/run/user/1000/bus"
	foreignBus = "/run/user/0/bus"
)

// healthySession is a correct logind runtime directory for testUID: a private
// directory and a socket, both owned by us.
func healthySession() map[string]pathFact {
	return map[string]pathFact{
		testDir: {exists: true, uid: testUID, mode: os.ModeDir | 0o700},
		testBus: {exists: true, uid: testUID, mode: os.ModeSocket | 0o600},
	}
}

func factStat(facts map[string]pathFact) func(string) pathFact {
	return func(path string) pathFact { return facts[path] }
}

func envFrom(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}

// The reported incident: an ssh login as root followed by `su matthalloran8`
// leaves root's session variables in a shell running as uid 1000, and a
// perfectly healthy keyring reads as permission denied.
func TestPlanSessionRepairCorrectsAForeignSessionInheritedByASuShell(t *testing.T) {
	repair, ok := planSessionRepair(testUID, envFrom(map[string]string{
		"XDG_RUNTIME_DIR":          "/run/user/0",
		"DBUS_SESSION_BUS_ADDRESS": "unix:path=" + foreignBus,
	}), factStat(map[string]pathFact{
		testDir:       {exists: true, uid: testUID, mode: os.ModeDir | 0o700},
		testBus:       {exists: true, uid: testUID, mode: os.ModeSocket | 0o600},
		"/run/user/0": {exists: true, uid: 0, mode: os.ModeDir | 0o700},
		foreignBus:    {exists: true, uid: 0, mode: os.ModeSocket | 0o600},
	}))
	if !ok {
		t.Fatal("no repair planned; the operator is back to editing environment variables by hand")
	}
	if repair.runtimeDir != testDir {
		t.Fatalf("runtimeDir = %q, want this user's own session %q", repair.runtimeDir, testDir)
	}
	if repair.busAddress != "unix:path="+testBus {
		t.Fatalf("busAddress = %q, want this user's own bus", repair.busAddress)
	}
	for _, want := range []string{"owned by uid 0", "uid 1000", testDir} {
		if !strings.Contains(repair.note, want) {
			t.Fatalf("note = %q, want it to name %q so the operator can still see the real cause", repair.note, want)
		}
	}
}

// A repair must never be able to aim at a session that is not ours, whatever
// the environment says. This is the property that keeps the feature from
// becoming a way to read another user's secrets.
func TestPlanSessionRepairOnlyEverTargetsThisUsersOwnSession(t *testing.T) {
	for _, hostile := range []string{"/run/user/0", "/run/user/1001", "/tmp/attacker"} {
		repair, ok := planSessionRepair(testUID, envFrom(map[string]string{
			"XDG_RUNTIME_DIR":          hostile,
			"DBUS_SESSION_BUS_ADDRESS": "unix:path=" + hostile + "/bus",
		}), factStat(map[string]pathFact{
			testDir:          {exists: true, uid: testUID, mode: os.ModeDir | 0o700},
			testBus:          {exists: true, uid: testUID, mode: os.ModeSocket | 0o600},
			hostile:          {exists: true, uid: 0, mode: os.ModeDir | 0o700},
			hostile + "/bus": {exists: true, uid: 0, mode: os.ModeSocket | 0o600},
		}))
		if !ok {
			continue
		}
		if repair.runtimeDir != testDir || repair.busAddress != "unix:path="+testBus {
			t.Fatalf("XDG_RUNTIME_DIR=%s produced repair %+v, which points somewhere other than this user's own session",
				hostile, repair)
		}
	}
}

func TestPlanSessionRepairRefusesWhenOurOwnSessionIsNotTrustworthy(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		facts map[string]pathFact
		why   string
	}{
		{
			name:  "no runtime directory at all",
			facts: map[string]pathFact{testBus: {exists: true, uid: testUID, mode: os.ModeSocket | 0o600}},
			why:   "a headless host with no logind session has nothing to redirect to",
		},
		{
			name: "runtime directory owned by someone else",
			facts: map[string]pathFact{
				testDir: {exists: true, uid: 0, mode: os.ModeDir | 0o700},
				testBus: {exists: true, uid: testUID, mode: os.ModeSocket | 0o600},
			},
			why: "a directory at our path that we do not own is not our session",
		},
		{
			name: "runtime directory writable by others",
			facts: map[string]pathFact{
				testDir: {exists: true, uid: testUID, mode: os.ModeDir | 0o777},
				testBus: {exists: true, uid: testUID, mode: os.ModeSocket | 0o600},
			},
			why: "anyone could plant a bus socket inside a world-writable directory",
		},
		{
			name: "runtime path is a file rather than a directory",
			facts: map[string]pathFact{
				testDir: {exists: true, uid: testUID, mode: 0o700},
				testBus: {exists: true, uid: testUID, mode: os.ModeSocket | 0o600},
			},
			why: "a regular file is not a runtime directory",
		},
		{
			name:  "no bus in our session",
			facts: map[string]pathFact{testDir: {exists: true, uid: testUID, mode: os.ModeDir | 0o700}},
			why:   "there is no Secret Service to reach",
		},
		{
			name: "bus is not a socket",
			facts: map[string]pathFact{
				testDir: {exists: true, uid: testUID, mode: os.ModeDir | 0o700},
				testBus: {exists: true, uid: testUID, mode: 0o600},
			},
			why: "a regular file named bus is not a session bus",
		},
		{
			name: "bus owned by someone else",
			facts: map[string]pathFact{
				testDir: {exists: true, uid: testUID, mode: os.ModeDir | 0o700},
				testBus: {exists: true, uid: 0, mode: os.ModeSocket | 0o600},
			},
			why: "a socket we do not own is not our bus",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, ok := planSessionRepair(testUID, envFrom(map[string]string{
				"XDG_RUNTIME_DIR": "/run/user/0",
			}), factStat(testCase.facts))
			if ok {
				t.Fatalf("planned a repair anyway; %s", testCase.why)
			}
		})
	}
}

// A working session is left alone. Rewriting it would be a no-op at best and,
// on a host with a deliberately custom setup, a regression.
func TestPlanSessionRepairLeavesAWorkingSessionUntouched(t *testing.T) {
	for _, testCase := range []struct {
		name string
		env  map[string]string
	}{
		{
			name: "already pointing at our own session",
			env: map[string]string{
				"XDG_RUNTIME_DIR":          testDir,
				"DBUS_SESSION_BUS_ADDRESS": "unix:path=" + testBus,
			},
		},
		{
			name: "trailing-slash spelling of the same directory",
			env: map[string]string{
				"XDG_RUNTIME_DIR":          testDir + "/",
				"DBUS_SESSION_BUS_ADDRESS": "unix:path=" + testBus,
			},
		},
		{
			name: "a non-unix bus transport we cannot reason about",
			env: map[string]string{
				"XDG_RUNTIME_DIR":          testDir,
				"DBUS_SESSION_BUS_ADDRESS": "tcp:host=127.0.0.1,port=12345",
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if _, ok := planSessionRepair(testUID, envFrom(testCase.env), factStat(healthySession())); ok {
				t.Fatal("repaired a session that was already working")
			}
		})
	}
}

// A custom runtime directory that the operator owns is a deliberate choice, not
// a fault, so only the unset or foreign cases are corrected.
func TestPlanSessionRepairRespectsACustomRuntimeDirectoryWeOwn(t *testing.T) {
	const custom = "/opt/session"
	facts := healthySession()
	facts[custom] = pathFact{exists: true, uid: testUID, mode: os.ModeDir | 0o700}
	facts[custom+"/bus"] = pathFact{exists: true, uid: testUID, mode: os.ModeSocket | 0o600}

	if _, ok := planSessionRepair(testUID, envFrom(map[string]string{
		"XDG_RUNTIME_DIR":          custom,
		"DBUS_SESSION_BUS_ADDRESS": "unix:path=" + custom + "/bus",
	}), factStat(facts)); ok {
		t.Fatal("overrode a custom session the operator owns and presumably chose")
	}
}

// An ssh session with no session variables at all is the other common way to
// reach an unreachable keyring, and it is repairable whenever logind did give
// this user a session.
func TestPlanSessionRepairFillsInAnEmptySession(t *testing.T) {
	repair, ok := planSessionRepair(testUID, envFrom(nil), factStat(healthySession()))
	if !ok {
		t.Fatal("no repair planned for a session with no variables set")
	}
	if repair.runtimeDir != testDir || repair.busAddress != "unix:path="+testBus {
		t.Fatalf("repair = %+v, want this user's own session", repair)
	}
	if !strings.Contains(repair.note, "unset") {
		t.Fatalf("note = %q, want it to say the variables were unset", repair.note)
	}
}

// Both variables are corrected together even when only one was wrong, because
// libsecret derives a bus from the runtime directory when the address is unset.
func TestPlanSessionRepairCorrectsBothVariablesTogether(t *testing.T) {
	repair, ok := planSessionRepair(testUID, envFrom(map[string]string{
		"XDG_RUNTIME_DIR":          "/run/user/0",
		"DBUS_SESSION_BUS_ADDRESS": "unix:path=" + testBus,
	}), factStat(map[string]pathFact{
		testDir:       {exists: true, uid: testUID, mode: os.ModeDir | 0o700},
		testBus:       {exists: true, uid: testUID, mode: os.ModeSocket | 0o600},
		"/run/user/0": {exists: true, uid: 0, mode: os.ModeDir | 0o700},
	}))
	if !ok {
		t.Fatal("no repair planned for a half-wrong session")
	}
	if repair.runtimeDir != testDir || repair.busAddress != "unix:path="+testBus {
		t.Fatalf("repair = %+v, want both variables naming the same session", repair)
	}
}

// withEnv must replace rather than append: a duplicate assignment leaves the
// subprocess taking whichever one the platform happens to prefer.
func TestWithEnvReplacesAnExistingAssignment(t *testing.T) {
	environ := withEnv([]string{"PATH=/bin", "XDG_RUNTIME_DIR=/run/user/0", "HOME=/root"},
		"XDG_RUNTIME_DIR", testDir)

	var seen int
	for _, entry := range environ {
		if strings.HasPrefix(entry, "XDG_RUNTIME_DIR=") {
			seen++
			if entry != "XDG_RUNTIME_DIR="+testDir {
				t.Fatalf("entry = %q, want the corrected value", entry)
			}
		}
	}
	if seen != 1 {
		t.Fatalf("XDG_RUNTIME_DIR appears %d times, want exactly 1", seen)
	}
	if len(environ) != 3 {
		t.Fatalf("environ = %v, want the other variables preserved", environ)
	}
}

func TestWithEnvAddsAMissingAssignment(t *testing.T) {
	environ := withEnv([]string{"PATH=/bin"}, "DBUS_SESSION_BUS_ADDRESS", "unix:path="+testBus)
	if len(environ) != 2 || environ[1] != "DBUS_SESSION_BUS_ADDRESS=unix:path="+testBus {
		t.Fatalf("environ = %v, want the assignment appended", environ)
	}
}

// sessionDiagnosis must not send an operator to fix variables the credential
// path already stopped using. On a host where the repair applies, the session
// is no longer the explanation for a failure.
func TestSessionDiagnosisStaysSilentWhenTheSessionWasRepaired(t *testing.T) {
	if _, repaired := repairSession(); !repaired {
		t.Skip("this host's session needs no repair, so there is nothing to suppress")
	}
	if diagnosis := sessionDiagnosis(); diagnosis != "" {
		t.Fatalf("sessionDiagnosis() = %q, want silence: the repaired session is not the fault", diagnosis)
	}
}
