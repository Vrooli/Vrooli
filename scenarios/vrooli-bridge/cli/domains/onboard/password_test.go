package onboard

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// neverPrompt fails the test if the masked prompt is ever reached — the
// no-auto-prompt invariant most of these tests defend.
func neverPrompt(t *testing.T) func() ([]byte, error) {
	t.Helper()
	return func() ([]byte, error) {
		t.Fatal("the masked prompt must not run unless --prompt-password was given")
		return nil, nil
	}
}

func TestPasswordResolve_NeverAutoPromptsEvenOnTTY(t *testing.T) {
	// The core contract: a TTY with no credential flags and no env var resolves
	// to empty ("host is already key-trusted") WITHOUT prompting.
	src := passwordSource{
		lookupEnv:  func(string) (string, bool) { return "", false },
		isTerminal: func() bool { return true },
		readSecret: neverPrompt(t),
		prompt:     &bytes.Buffer{},
	}
	got, source, err := src.resolve("deploy", "host", false, false)
	require.NoError(t, err)
	require.Equal(t, "", got)
	require.Equal(t, credentialNone, source)
}

func TestPasswordResolve_StdinReadsPipeAndStripsOneNewline(t *testing.T) {
	cases := []struct{ in, want string }{
		{"piped-secret\n", "piped-secret"},
		{"piped-secret\r\n", "piped-secret"},
		{"piped-secret", "piped-secret"},
		{"piped-secret\r", "piped-secret\r"},     // bare CR is password material, not a pipe newline
		{"trailing-space \n", "trailing-space "}, // inner whitespace is part of the password
		{"multi\nline\n", "multi\nline"},         // only the final pipe newline is stripped
		{"", ""},                                 // empty pipe = key-trusted, still explicit
	}
	for _, tc := range cases {
		src := passwordSource{
			lookupEnv: func(string) (string, bool) {
				t.Fatal("env must not be consulted when --password-stdin is given")
				return "", false
			},
			isTerminal: func() bool { return true },
			readSecret: neverPrompt(t),
			stdin:      bytes.NewBufferString(tc.in),
			prompt:     &bytes.Buffer{},
		}
		got, source, err := src.resolve("root", "host", true, false)
		require.NoError(t, err)
		require.Equal(t, tc.want, got, "input %q", tc.in)
		require.Equal(t, credentialFromStdin, source)
	}
}

func TestPasswordResolve_StdinAndPromptAreMutuallyExclusive(t *testing.T) {
	src := passwordSource{
		lookupEnv:  func(string) (string, bool) { return "", false },
		isTerminal: func() bool { return true },
		readSecret: neverPrompt(t),
		stdin:      bytes.NewBufferString("x"),
		prompt:     &bytes.Buffer{},
	}
	_, _, err := src.resolve("root", "host", true, true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mutually exclusive")
}

func TestPasswordResolve_PromptRequiresTTY(t *testing.T) {
	src := passwordSource{
		lookupEnv:  func(string) (string, bool) { return "", false },
		isTerminal: func() bool { return false },
		readSecret: neverPrompt(t),
		prompt:     &bytes.Buffer{},
	}
	_, _, err := src.resolve("root", "host", false, true)
	require.Error(t, err)
	// The error must teach the working alternatives, not just refuse.
	require.Contains(t, err.Error(), "--password-stdin")
	require.Contains(t, err.Error(), sshPasswordEnvVar)
}

func TestPasswordResolve_ExplicitPromptAsksOnTTY(t *testing.T) {
	var prompt bytes.Buffer
	src := passwordSource{
		lookupEnv:  func(string) (string, bool) { return "env-must-lose", true },
		isTerminal: func() bool { return true },
		readSecret: func() ([]byte, error) { return []byte("typed-secret"), nil },
		prompt:     &prompt,
	}
	got, source, err := src.resolve("deploy", "10.0.0.9", false, true)
	require.NoError(t, err)
	require.Equal(t, "typed-secret", got)
	require.Equal(t, credentialFromPrompt, source)
	require.Contains(t, prompt.String(), "deploy@10.0.0.9")
	require.Contains(t, prompt.String(), "leave blank")
}

func TestPasswordResolve_PromptReadErrorSurfaces(t *testing.T) {
	src := passwordSource{
		lookupEnv:  func(string) (string, bool) { return "", false },
		isTerminal: func() bool { return true },
		readSecret: func() ([]byte, error) { return nil, errors.New("tty closed") },
		prompt:     &bytes.Buffer{},
	}
	_, _, err := src.resolve("root", "host", false, true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "read SSH password")
}

func TestPasswordResolve_EnvIsTheAmbientFallback(t *testing.T) {
	src := passwordSource{
		lookupEnv:  func(k string) (string, bool) { return "from-env", k == sshPasswordEnvVar },
		isTerminal: func() bool { return true },
		readSecret: neverPrompt(t),
		prompt:     &bytes.Buffer{},
	}
	got, source, err := src.resolve("deploy", "host", false, false)
	require.NoError(t, err)
	require.Equal(t, "from-env", got)
	require.Equal(t, credentialFromEnv, source)
}

func TestPasswordResolve_EmptyEnvIsHonoredAsKeyTrusted(t *testing.T) {
	// An explicitly-set-but-empty env var means "the host already trusts the
	// bridge key" — honored as an env-sourced credential, never a prompt.
	src := passwordSource{
		lookupEnv:  func(string) (string, bool) { return "", true },
		isTerminal: func() bool { return true },
		readSecret: neverPrompt(t),
		prompt:     &bytes.Buffer{},
	}
	got, source, err := src.resolve("root", "host", false, false)
	require.NoError(t, err)
	require.Equal(t, "", got)
	require.Equal(t, credentialFromEnv, source)
}

func TestPasswordResolve_StdinFlagBeatsEnv(t *testing.T) {
	// An explicit flag is stronger intent than ambient environment.
	src := passwordSource{
		lookupEnv:  func(string) (string, bool) { return "env-must-lose", true },
		isTerminal: func() bool { return false },
		readSecret: neverPrompt(t),
		stdin:      bytes.NewBufferString("stdin-wins\n"),
		prompt:     &bytes.Buffer{},
	}
	got, source, err := src.resolve("root", "host", true, false)
	require.NoError(t, err)
	require.Equal(t, "stdin-wins", got)
	require.Equal(t, credentialFromStdin, source)
}
