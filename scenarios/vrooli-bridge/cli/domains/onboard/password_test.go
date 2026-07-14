package onboard

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPasswordResolve_EnvWinsAndSkipsPrompt(t *testing.T) {
	src := passwordSource{
		lookupEnv:  func(k string) (string, bool) { return "from-env", k == sshPasswordEnvVar },
		isTerminal: func() bool { return true }, // even on a TTY, env takes precedence
		readSecret: func() ([]byte, error) { t.Fatal("prompt must not run when env is set"); return nil, nil },
		prompt:     &bytes.Buffer{},
	}
	got, err := src.resolve("deploy", "host")
	require.NoError(t, err)
	require.Equal(t, "from-env", got)
}

func TestPasswordResolve_EmptyEnvIsHonoredAsKeyTrusted(t *testing.T) {
	// An explicitly-set-but-empty env var means "the host already trusts the
	// bridge key" — it must be honored, not fall through to a prompt.
	src := passwordSource{
		lookupEnv:  func(k string) (string, bool) { return "", true },
		isTerminal: func() bool { return true },
		readSecret: func() ([]byte, error) { t.Fatal("prompt must not run"); return nil, nil },
		prompt:     &bytes.Buffer{},
	}
	got, err := src.resolve("root", "host")
	require.NoError(t, err)
	require.Equal(t, "", got)
}

func TestPasswordResolve_NonTTYWithoutEnvReturnsEmpty(t *testing.T) {
	// Non-interactive and no env var: assume key-trusted rather than blocking a
	// scripted run on a prompt that can never be answered.
	src := passwordSource{
		lookupEnv:  func(string) (string, bool) { return "", false },
		isTerminal: func() bool { return false },
		readSecret: func() ([]byte, error) { t.Fatal("prompt must not run without a TTY"); return nil, nil },
		prompt:     &bytes.Buffer{},
	}
	got, err := src.resolve("root", "host")
	require.NoError(t, err)
	require.Equal(t, "", got)
}

func TestPasswordResolve_TTYPrompts(t *testing.T) {
	var prompt bytes.Buffer
	src := passwordSource{
		lookupEnv:  func(string) (string, bool) { return "", false },
		isTerminal: func() bool { return true },
		readSecret: func() ([]byte, error) { return []byte("typed-secret"), nil },
		prompt:     &prompt,
	}
	got, err := src.resolve("deploy", "10.0.0.9")
	require.NoError(t, err)
	require.Equal(t, "typed-secret", got)
	require.Contains(t, prompt.String(), "deploy@10.0.0.9")
	require.Contains(t, prompt.String(), "leave blank")
}

func TestPasswordResolve_TTYReadErrorSurfaces(t *testing.T) {
	src := passwordSource{
		lookupEnv:  func(string) (string, bool) { return "", false },
		isTerminal: func() bool { return true },
		readSecret: func() ([]byte, error) { return nil, errors.New("tty closed") },
		prompt:     &bytes.Buffer{},
	}
	_, err := src.resolve("root", "host")
	require.Error(t, err)
	require.Contains(t, err.Error(), "read SSH password")
}
