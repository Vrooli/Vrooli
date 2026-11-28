package resources

import (
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/values"
)

func TestClassifyActivityNetworkLaunchRequiresExactOwnerContract(t *testing.T) {
	t.Run("legacy direct artifact launch", func(t *testing.T) {
		kind, err := classifyActivityNetworkLaunch(activityNetworkLaunchFacts{
			PID: 1411181, Executable: "/opt/ollama/ollama", CommandLine: "/opt/ollama/ollama serve",
			RootlessKitPath: "/usr/bin/rootlesskit", RootlessKitArgs: []string{"--state-dir", "/tmp/vra-ollama"},
			ArtifactPath: "/opt/ollama/ollama", LegacyServiceArg: []string{"serve"},
		})
		if err != nil || kind != activityNetworkLaunchLegacyDirect {
			t.Fatalf("classification = %v, %v; want verified legacy direct launch", kind, err)
		}
	})

	t.Run("rootless owner and exact child", func(t *testing.T) {
		args := []string{"--state-dir", "/tmp/vra-ollama", "--net=slirp4netns", "--", "/opt/ollama/ollama", "serve"}
		kind, err := classifyActivityNetworkLaunch(activityNetworkLaunchFacts{
			PID: 1411181, Executable: "/usr/bin/rootlesskit", CommandLine: strings.Join(append([]string{"/usr/bin/rootlesskit"}, args...), " "),
			RootlessKitPath: "/usr/bin/rootlesskit", RootlessKitArgs: args, StateDir: "/tmp/vra-ollama",
			ArtifactPath: "/opt/ollama/ollama", ChildPID: 1411182, ChildParentPID: 1411181,
			ChildExecutable: "/opt/ollama/ollama", ChildCommandLine: "/opt/ollama/ollama serve", ChildArgs: []string{"serve"},
			LegacyServiceArg: []string{"serve"},
		})
		if err != nil || kind != activityNetworkLaunchRootlessKit {
			t.Fatalf("classification = %v, %v; want verified RootlessKit owner", kind, err)
		}
	})

	tests := []struct {
		name  string
		facts activityNetworkLaunchFacts
	}{
		{
			name:  "direct process argv drift",
			facts: activityNetworkLaunchFacts{PID: 1411181, Executable: "/opt/ollama/ollama", CommandLine: "/opt/ollama/ollama serve --host 0.0.0.0", RootlessKitPath: "/usr/bin/rootlesskit", ArtifactPath: "/opt/ollama/ollama", LegacyServiceArg: []string{"serve"}},
		},
		{
			name:  "rootless child parent mismatch",
			facts: activityNetworkLaunchFacts{PID: 1411181, Executable: "/usr/bin/rootlesskit", CommandLine: "/usr/bin/rootlesskit --state-dir /tmp/vra-ollama", RootlessKitPath: "/usr/bin/rootlesskit", RootlessKitArgs: []string{"--state-dir", "/tmp/vra-ollama"}, StateDir: "/tmp/vra-ollama", ChildPID: 1411182, ChildParentPID: 9000, ChildExecutable: "/opt/ollama/ollama", ChildCommandLine: "/opt/ollama/ollama serve", ChildArgs: []string{"serve"}, ArtifactPath: "/opt/ollama/ollama"},
		},
		{
			name:  "unknown executable",
			facts: activityNetworkLaunchFacts{PID: 1411181, Executable: "/tmp/other", CommandLine: "/tmp/other serve", RootlessKitPath: "/usr/bin/rootlesskit", ArtifactPath: "/opt/ollama/ollama", LegacyServiceArg: []string{"serve"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if kind, err := classifyActivityNetworkLaunch(tt.facts); err == nil || kind != activityNetworkLaunchUnknown {
				t.Fatalf("classification = %v, %v; want fail-closed unknown", kind, err)
			}
		})
	}
}

func TestProcessCommandLineMatchesRejectsAmbiguousArguments(t *testing.T) {
	if processCommandLineMatches("/bin/tool --value with space", "/bin/tool", []string{"--value", "with space"}) {
		t.Fatal("ambiguous process argv was accepted")
	}
	if !processCommandLineMatches("/bin/tool --flag", "/bin/tool", []string{"--flag"}) {
		t.Fatal("exact argv was rejected")
	}
}

func TestSameManagedServiceOwnerRequiresStableIdentityAndToken(t *testing.T) {
	expected := ManagedServiceState{PID: 100, Incarnation: "boot:1", InstanceID: "instance-a", OwnershipTokenHash: "token-a"}
	if !sameManagedServiceOwner(expected, expected) {
		t.Fatal("matching supervisor owner was rejected")
	}
	mutations := []func(*ManagedServiceState){
		func(s *ManagedServiceState) { s.PID++ },
		func(s *ManagedServiceState) { s.Incarnation = "boot:2" },
		func(s *ManagedServiceState) { s.InstanceID = "instance-b" },
		func(s *ManagedServiceState) { s.OwnershipTokenHash = "token-b" },
	}
	for i, mutate := range mutations {
		actual := expected
		mutate(&actual)
		if sameManagedServiceOwner(expected, actual) {
			t.Fatalf("identity mutation %d was accepted", i)
		}
	}
}

func TestOllamaOwnerCommandUsesVerifiedOwnerLoopbackEndpoint(t *testing.T) {
	ownerEnv := map[string]string{"OLLAMA_HOST": "http://127.0.0.1:21434"}
	ambient := []string{
		"PATH=/usr/bin",
		"OLLAMA_BASE_URL=https://remote.invalid",
		"OLLAMA_BASE_URL=http://attacker.invalid:9999",
	}
	got, err := activityNetworkOwnerCommandEnvironment("ollama", ownerEnv, ambient)
	if err != nil {
		t.Fatal(err)
	}
	if base := values.EnvValue(got, "OLLAMA_BASE_URL"); base != "http://127.0.0.1:21434" {
		t.Fatalf("owner-bound OLLAMA_BASE_URL=%q", base)
	}
	count := 0
	for _, entry := range got {
		if strings.HasPrefix(entry, "OLLAMA_BASE_URL=") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("owner command retained %d OLLAMA_BASE_URL entries", count)
	}
	if values.EnvValue(got, activityNetworkJoinedEnv) != "1" {
		t.Fatal("owner command did not mark the verified namespace join")
	}
}

func TestOllamaOwnerBaseURLRequiresLiteralLoopbackAndValidPort(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		valid       bool
	}{
		{input: "127.0.0.1:21434", want: "http://127.0.0.1:21434", valid: true},
		{input: "http://[::1]:21434", want: "http://[::1]:21434", valid: true},
		{input: "http://127.0.0.1:70000", valid: false},
		{input: "http://192.0.2.4:21434", valid: false},
		{input: "http://localhost:21434", valid: false},
		{input: "https://127.0.0.1:21434", valid: false},
		{input: "http://127.0.0.1:21434/api", valid: false},
		{input: "127.0.0.1", valid: false},
		{input: "", valid: false},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ollamaOwnerBaseURL(tc.input)
			if tc.valid && (err != nil || got != tc.want) {
				t.Fatalf("ollamaOwnerBaseURL(%q) = %q, %v; want %q", tc.input, got, err, tc.want)
			}
			if !tc.valid && err == nil {
				t.Fatalf("ollamaOwnerBaseURL(%q) = %q, want refusal", tc.input, got)
			}
		})
	}
}

func TestVerifiedActivityNetworkOwnerEnvironmentRequiresTokenAndIncarnation(t *testing.T) {
	state := ManagedServiceState{PID: 4242, Incarnation: "boot:original", OwnershipTokenHash: managedServiceTokenHash("owned")}
	withOwnershipFacts(t, func(pid int) (map[string]string, error) {
		if pid != state.PID {
			t.Fatalf("environment read PID=%d, want %d", pid, state.PID)
		}
		return map[string]string{managedServiceOwnershipTokenEnv: "owned", "OLLAMA_HOST": "127.0.0.1:21434"}, nil
	}, executableAt("/opt/ollama/ollama"))
	if _, err := verifiedActivityNetworkOwnerEnvironment(state); err != nil {
		t.Fatalf("verified owner environment rejected: %v", err)
	}
	wrongToken := state
	wrongToken.OwnershipTokenHash = managedServiceTokenHash("other")
	if _, err := verifiedActivityNetworkOwnerEnvironment(wrongToken); err == nil {
		t.Fatal("owner environment with a mismatched token was accepted")
	}
	wrongIncarnation := state
	wrongIncarnation.Incarnation = "boot:replacement"
	if _, err := verifiedActivityNetworkOwnerEnvironment(wrongIncarnation); err == nil {
		t.Fatal("owner environment with a mismatched incarnation was accepted")
	}
}

func TestOtherActivityNetworkCommandsDoNotAcquireOllamaEndpoint(t *testing.T) {
	got, err := activityNetworkOwnerCommandEnvironment("whisper", map[string]string{}, []string{"OLLAMA_BASE_URL=http://ambient.invalid:1"})
	if err != nil {
		t.Fatal(err)
	}
	if values.EnvValue(got, "OLLAMA_BASE_URL") != "http://ambient.invalid:1" {
		t.Fatal("non-Ollama owner endpoint was unexpectedly rewritten")
	}
}
