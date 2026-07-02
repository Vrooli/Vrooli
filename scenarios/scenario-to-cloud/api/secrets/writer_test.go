package secrets

import (
	"context"
	"strings"
	"testing"

	"scenario-to-cloud/ssh"
)

type fakeWriterRunner struct {
	result ssh.Result
	err    error
}

func (f fakeWriterRunner) Run(_ context.Context, _ ssh.Config, _ string, _ ssh.RunOptions) (ssh.Result, error) {
	return f.result, f.err
}

func TestReadFromVPSRejectsNonStringSecrets(t *testing.T) {
	runner := fakeWriterRunner{
		result: ssh.Result{
			Stdout:   `{"_metadata":{"generated_by":"scenario-to-cloud"},"POSTGRES_PASSWORD":42}`,
			ExitCode: 0,
		},
	}

	_, err := ReadFromVPS(context.Background(), runner, ssh.Config{}, "/root/Vrooli")
	if err == nil || !strings.Contains(err.Error(), "must be a JSON string") {
		t.Fatalf("ReadFromVPS error = %v, want string validation error", err)
	}
}

func TestReadAllFromVPSRejectsInvalidMetadata(t *testing.T) {
	runner := fakeWriterRunner{
		result: ssh.Result{
			Stdout:   `{"_metadata":"bad","API_KEY":"secret"}`,
			ExitCode: 0,
		},
	}

	_, err := ReadAllFromVPS(context.Background(), runner, ssh.Config{}, "/root/Vrooli")
	if err == nil || !strings.Contains(err.Error(), "invalid secrets metadata") {
		t.Fatalf("ReadAllFromVPS error = %v, want metadata validation error", err)
	}
}

func TestWriteToVPSFailsWhenExistingSecretsInvalid(t *testing.T) {
	runner := fakeWriterRunner{
		result: ssh.Result{
			Stdout:   `{"POSTGRES_PASSWORD":42}`,
			ExitCode: 0,
		},
	}

	err := WriteToVPS(
		context.Background(),
		runner,
		ssh.Config{},
		"/root/Vrooli",
		[]GeneratedSecret{{ID: "pg", Key: "POSTGRES_PASSWORD", Value: "generated"}},
		nil,
		"landing-page-business-suite",
	)
	if err == nil || !strings.Contains(err.Error(), "read existing secrets.json") {
		t.Fatalf("WriteToVPS error = %v, want existing secrets read failure", err)
	}
}
