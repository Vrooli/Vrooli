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

// A deploy must refuse rather than guess when it cannot tell whether the target
// already holds a credential. Treating an unreachable store as "nothing stored"
// would regenerate a database password that is still in use, which is exactly
// the "password authentication failed" class of redeploy failure.
func TestWriteToVPSRefusesWhenTheRemoteStoreCannotAnswer(t *testing.T) {
	runner := fakeWriterRunner{
		result: ssh.Result{
			Stdout:   `{"configured":false,"provider_state":"unavailable"}`,
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
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("WriteToVPS error = %v, want a refusal naming the unreachable remote store", err)
	}
}

// The identity is what makes a credential findable again. A deploy that wrote
// values under an empty namespace would store them where nothing looks.
func TestWriteToVPSRequiresAScenarioIdentity(t *testing.T) {
	runner := fakeWriterRunner{result: ssh.Result{Stdout: `{}`, ExitCode: 0}}
	err := WriteToVPS(
		context.Background(), runner, ssh.Config{}, "/root/Vrooli",
		[]GeneratedSecret{{ID: "pg", Key: "POSTGRES_PASSWORD", Value: "generated"}}, nil, "  ",
	)
	if err == nil || !strings.Contains(err.Error(), "scenario id is required") {
		t.Fatalf("WriteToVPS error = %v, want a refusal naming the missing scenario id", err)
	}
}

// A secret value must never enter a command string: that string is argv for the
// local ssh process and the argument to the remote shell, so anything embedded
// in it is readable in both process listings for the length of the deploy.
func TestWriteToVPSNeverPutsASecretValueInTheCommand(t *testing.T) {
	const value = "super-secret-generated-value"
	runner := &recordingWriterRunner{stdout: `{"configured":false,"provider_state":"available"}`}
	_ = WriteToVPS(
		context.Background(), runner, ssh.Config{}, "/root/Vrooli",
		[]GeneratedSecret{{ID: "pg", Key: "POSTGRES_PASSWORD", Value: value}}, nil, "demo",
	)
	for _, command := range runner.commands {
		if strings.Contains(command, value) {
			t.Fatalf("secret value appeared in an ssh command string: %q", command)
		}
	}
	if len(runner.stdins) == 0 {
		t.Fatal("no value was sent over stdin, so nothing was provisioned")
	}
	found := false
	for _, in := range runner.stdins {
		if string(in) == value {
			found = true
		}
	}
	if !found {
		t.Fatal("the secret value never reached the remote command's standard input")
	}
}

type recordingWriterRunner struct {
	stdout   string
	commands []string
	stdins   [][]byte
}

func (r *recordingWriterRunner) Run(_ context.Context, _ ssh.Config, command string, opts ssh.RunOptions) (ssh.Result, error) {
	r.commands = append(r.commands, command)
	if len(opts.Stdin) > 0 {
		r.stdins = append(r.stdins, append([]byte(nil), opts.Stdin...))
	}
	if strings.Contains(command, "doctor") {
		return ssh.Result{Stdout: `{"provider":{"condition":"available"}}`, ExitCode: 0}, nil
	}
	return ssh.Result{Stdout: r.stdout, ExitCode: 0}, nil
}
