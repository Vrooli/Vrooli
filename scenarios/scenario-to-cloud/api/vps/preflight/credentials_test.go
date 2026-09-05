package preflight

import (
	"context"
	"strings"
	"testing"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/ssh"
)

type fakeCredentialRunner struct {
	commands []string
}

func (f *fakeCredentialRunner) Run(_ context.Context, _ ssh.Config, command string, _ ssh.RunOptions) (ssh.Result, error) {
	f.commands = append(f.commands, command)
	switch {
	case strings.Contains(command, "credentials' 'doctor"):
		return ssh.Result{Stdout: `{"provider":{"condition":"available","available":true}}`, ExitCode: 0}, nil
	case strings.Contains(command, "credentials' 'status"):
		return ssh.Result{Stdout: `{"identity":"vrooli/landing-page-business-suite","field":"postgres-password","configured":true,"provider":"ssh","provider_state":"available"}`, ExitCode: 0}, nil
	case strings.HasPrefix(command, "pg_isready"):
		return ssh.Result{Stdout: "localhost:5433 - accepting connections", ExitCode: 0}, nil
	default:
		return ssh.Result{ExitCode: 0}, nil
	}
}

func TestRunCredentialValidationUsesAuthorityAndNeverReadsPlaintextSecrets(t *testing.T) {
	runner := &fakeCredentialRunner{}
	checks := RunCredentialValidation(
		context.Background(),
		ssh.Config{User: "root", Host: "target"},
		runner,
		domain.CloudManifest{
			Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
			Dependencies: domain.ManifestDependencies{
				Resources: []string{"postgres"},
			},
		},
		"/root/Vrooli",
	)

	if len(checks) != 1 || checks[0].ID != "postgres_credentials" || checks[0].Status != domain.PreflightPass {
		t.Fatalf("checks = %#v, want passing postgres credential check", checks)
	}
	for _, command := range runner.commands {
		if strings.Contains(command, "secrets.json") || strings.Contains(command, "PGPASSWORD") {
			t.Fatalf("credential validation used plaintext secret path/value: %s", command)
		}
	}
}
