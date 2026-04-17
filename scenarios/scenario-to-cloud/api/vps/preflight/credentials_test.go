package preflight

import (
	"context"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/ssh"
	"strings"
	"testing"
)

type fakeCredentialRunner struct {
	result ssh.Result
	err    error
}

func (f fakeCredentialRunner) Run(_ context.Context, _ ssh.Config, _ string, _ ssh.RunOptions) (ssh.Result, error) {
	return f.result, f.err
}

func TestReadSecretsForValidationRejectsNonStringSecrets(t *testing.T) {
	runner := fakeCredentialRunner{
		result: ssh.Result{
			Stdout:   `{"_metadata":{"generated_by":"scenario-to-cloud"},"POSTGRES_PASSWORD":42}`,
			ExitCode: 0,
		},
	}

	_, err := readSecretsForValidation(context.Background(), ssh.Config{}, runner, "/root/Vrooli")
	if err == nil || !strings.Contains(err.Error(), "must be a JSON string") {
		t.Fatalf("readSecretsForValidation error = %v, want string validation error", err)
	}
}

func TestRunCredentialValidationWarnsWhenSecretsInvalid(t *testing.T) {
	runner := fakeCredentialRunner{
		result: ssh.Result{
			Stdout:   `{"POSTGRES_PASSWORD":42}`,
			ExitCode: 0,
		},
	}

	checks := RunCredentialValidation(
		context.Background(),
		ssh.Config{},
		runner,
		domain.CloudManifest{
			Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
			Dependencies: domain.ManifestDependencies{
				Resources: []string{"postgres"},
			},
		},
		"/root/Vrooli",
	)

	if len(checks) != 1 {
		t.Fatalf("len(checks) = %d, want 1", len(checks))
	}
	if checks[0].ID != "secrets_read" || checks[0].Status != domain.PreflightWarn {
		t.Fatalf("checks[0] = %#v, want secrets_read warning", checks[0])
	}
}
