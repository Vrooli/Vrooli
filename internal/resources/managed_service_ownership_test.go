package resources

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"testing"

	platform "github.com/vrooli/platform-go"
)

func withOwnershipFacts(t *testing.T, env func(int) (map[string]string, error), executable func(int) (string, error)) {
	t.Helper()
	previousEnv, previousExecutable, previousIncarnation := readManagedServiceEnvironment, managedServiceExecutablePath, managedServiceProcessIncarnation
	readManagedServiceEnvironment, managedServiceExecutablePath = env, executable
	managedServiceProcessIncarnation = func(int) (string, error) { return "boot:original", nil }
	t.Cleanup(func() {
		readManagedServiceEnvironment, managedServiceExecutablePath, managedServiceProcessIncarnation = previousEnv, previousExecutable, previousIncarnation
	})
}

func unsupportedEnvironment(int) (map[string]string, error) {
	return nil, fmt.Errorf("platform: process environment inspection is not supported: %w", platform.ErrUnsupported)
}

func executableAt(path string) func(int) (string, error) {
	return func(int) (string, error) { return path, nil }
}

func ownershipState() ManagedServiceState {
	return ManagedServiceState{PID: 4242, Incarnation: "boot:original", ArtifactPath: "/opt/postgres/bin/postgres", OwnershipTokenHash: managedServiceTokenHash("owned")}
}

func TestOwnershipFallsBackToTheExecutableWhenTheEnvironmentIsUnobservable(t *testing.T) {
	withOwnershipFacts(t, unsupportedEnvironment, executableAt("/opt/postgres/bin/postgres"))
	if err := verifyManagedServiceOwnership(ownershipState()); err != nil {
		t.Fatalf("verifyManagedServiceOwnership() = %v, want the executable proof to establish ownership", err)
	}
}

func TestOwnershipStillFailsWhenNeitherProofHolds(t *testing.T) {
	withOwnershipFacts(t, unsupportedEnvironment, executableAt("/usr/bin/impostor"))
	err := verifyManagedServiceOwnership(ownershipState())
	if err == nil || !errors.Is(err, platform.ErrUnsupported) || !strings.Contains(err.Error(), "verify managed-service process ownership") {
		t.Fatalf("verifyManagedServiceOwnership() = %v, want an ownership failure naming the unobservable environment", err)
	}
}

func TestOwnershipDoesNotFallBackOnARealEnvironmentReadFailure(t *testing.T) {
	denied := func(int) (map[string]string, error) { return nil, syscall.EPERM }
	withOwnershipFacts(t, denied, executableAt("/opt/postgres/bin/postgres"))
	if err := verifyManagedServiceOwnership(ownershipState()); !errors.Is(err, syscall.EPERM) {
		t.Fatalf("verifyManagedServiceOwnership() = %v, want the permission failure preserved", err)
	}
}

func TestOwnershipAcceptsTheTokenWithoutConsultingTheExecutable(t *testing.T) {
	env := func(int) (map[string]string, error) {
		return map[string]string{managedServiceOwnershipTokenEnv: "owned"}, nil
	}
	withOwnershipFacts(t, env, func(int) (string, error) {
		t.Fatal("executable consulted although the token proved ownership")
		return "", nil
	})
	if err := verifyManagedServiceOwnership(ownershipState()); err != nil {
		t.Fatalf("verifyManagedServiceOwnership() = %v", err)
	}
}

func TestOwnershipAcceptsAClearedEnvironmentWhenTheExecutableMatches(t *testing.T) {
	cleared := func(int) (map[string]string, error) { return map[string]string{}, nil }
	withOwnershipFacts(t, cleared, executableAt("/opt/postgres/bin/postgres"))
	if err := verifyManagedServiceOwnership(ownershipState()); err != nil {
		t.Fatalf("verifyManagedServiceOwnership() = %v", err)
	}
}

func TestExecutableMatchAcceptsOnlyTheArtifactOrItsBundle(t *testing.T) {
	cases := map[string]bool{
		"/opt/postgres/bin/postgres":           true,
		"/opt/postgres/bin/postgres/libexec/x": true,
		"/opt/postgres/bin/postgres-evil":      false,
		"/opt/postgres/bin":                    false,
		"relative/postgres":                    false,
	}
	for executable, want := range cases {
		withOwnershipFacts(t, unsupportedEnvironment, executableAt(executable))
		if got := managedServiceExecutableMatchesArtifact(ownershipState()); got != want {
			t.Errorf("executable %q match = %t, want %t", executable, got, want)
		}
	}
	withOwnershipFacts(t, unsupportedEnvironment, func(int) (string, error) { return "", platform.ErrUnsupported })
	if managedServiceExecutableMatchesArtifact(ownershipState()) {
		t.Error("a host that cannot name the executable must not match")
	}
}

// A replacement can run the exact same executable. A path match alone must
// never grant ownership of that replacement after the recorded process exits.
func TestOwnershipRejectsSameExecutableWithoutRecordedIncarnation(t *testing.T) {
	withOwnershipFacts(t, func(int) (map[string]string, error) { return map[string]string{}, nil }, executableAt("/opt/postgres/bin/postgres"))
	if err := verifyManagedServiceOwnership(func() ManagedServiceState { s := ownershipState(); s.Incarnation = ""; return s }()); err == nil {
		t.Fatal("same executable accepted without launch incarnation or ownership token")
	}
}

func TestOwnershipRejectsReusedPIDDespiteSameExecutableOrToken(t *testing.T) {
	for _, token := range []string{"", "owned"} {
		t.Run("token_"+token, func(t *testing.T) {
			withOwnershipFacts(t, func(int) (map[string]string, error) {
				return map[string]string{managedServiceOwnershipTokenEnv: token}, nil
			}, executableAt("/opt/postgres/bin/postgres"))
			managedServiceProcessIncarnation = func(int) (string, error) { return "boot:replacement", nil }
			if err := verifyManagedServiceOwnership(ownershipState()); err == nil {
				t.Fatal("replacement incarnation gained ownership")
			}
		})
	}
}

func TestOwnershipRejectsContradictoryTokenEvenWhenExecutableAndIncarnationMatch(t *testing.T) {
	withOwnershipFacts(t, func(int) (map[string]string, error) {
		return map[string]string{managedServiceOwnershipTokenEnv: "someone-else"}, nil
	}, executableAt("/opt/postgres/bin/postgres"))
	if err := verifyManagedServiceOwnership(ownershipState()); err == nil {
		t.Fatal("contradictory token overridden by executable")
	}
}

func TestOwnershipLegacyMatchingTokenRemainsUsable(t *testing.T) {
	withOwnershipFacts(t, func(int) (map[string]string, error) {
		return map[string]string{managedServiceOwnershipTokenEnv: "owned"}, nil
	}, executableAt("/opt/postgres/bin/postgres"))
	s := ownershipState()
	s.Incarnation = ""
	if err := verifyManagedServiceOwnership(s); err != nil {
		t.Fatalf("legacy explicit token proof rejected: %v", err)
	}
}
