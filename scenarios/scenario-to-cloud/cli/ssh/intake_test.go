package ssh

import (
	"strings"
	"testing"
)

func TestResolveSSHPasswordPrefersExplicitStdin(t *testing.T) {
	value, err := resolveSSHPasswordFrom(true, strings.NewReader("from-stdin\r\n"), func(string) string { return "from-env" })
	if err != nil {
		t.Fatal(err)
	}
	if value != "from-stdin" {
		t.Fatalf("password = %q, want stdin value", value)
	}
}

func TestResolveSSHPasswordUsesEnvironmentWithoutReadingInput(t *testing.T) {
	value, err := resolveSSHPasswordFrom(false, strings.NewReader("must-not-be-read"), func(name string) string {
		if name != scenarioToCloudSSHSecretEnv {
			t.Fatalf("environment name = %q", name)
		}
		return "from-env"
	})
	if err != nil {
		t.Fatal(err)
	}
	if value != "from-env" {
		t.Fatalf("password = %q, want environment value", value)
	}
}

func TestSSHCommandsRejectPasswordArguments(t *testing.T) {
	for _, run := range []struct {
		name string
		fn   func() error
	}{
		{name: "copy-key", fn: func() error { return runCopyKey(nil, []string{"host", "--password", "secret"}) }},
		{name: "bootstrap", fn: func() error { return runBootstrap(nil, []string{"host", "--password", "secret"}) }},
		{name: "generate", fn: func() error { return runGenerate(nil, []string{"key", "--password", "secret"}) }},
	} {
		t.Run(run.name, func(t *testing.T) {
			err := run.fn()
			if err == nil || !strings.Contains(err.Error(), "must not be placed in argv") {
				t.Fatalf("error = %v, want argv rejection", err)
			}
		})
	}
}

func TestNonInteractiveBootstrapMessageNamesSecretSafeChannels(t *testing.T) {
	message := nonInteractiveBootstrapMessage("host", "root", 22, "key")
	if strings.Contains(message, "when prompted") || strings.Contains(message, "--password ") {
		t.Fatalf("message advertises an unsafe prompt or argv secret: %s", message)
	}
	for _, expected := range []string{"--password-stdin", "$SCENARIO_TO_CLOUD_SSH_PASSWORD"} {
		if !strings.Contains(message, expected) {
			t.Fatalf("message missing %q: %s", expected, message)
		}
	}
}
