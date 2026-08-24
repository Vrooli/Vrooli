// Package content implements the `resource-postgres content` subcommand group.
package content

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/envkit-go"
	"time"
)

// Runner executes one psql invocation against a running PostgreSQL instance.
// It is an interface so handlers can be unit-tested with an injected fake.
//
// The connection is always TCP on the loopback address rather than a
// Unix-domain socket, because Windows PostgreSQL has no Unix-domain sockets and
// a socket path would make this code path Unix-only.
type Runner interface {
	Run(ctx context.Context, endpoint Endpoint, args []string, stdin io.Reader, env []string) ([]byte, []byte, error)
}

// Endpoint is where the managed instance listens.
type Endpoint struct {
	Host string
	Port string
}

// processRunner is the production Runner: it executes the psql client that the
// managed-service driver staged, talking to the process that driver supervises.
// The resource declares a native managed service, so its CLI drives that
// process directly instead of a container runtime.
type processRunner struct {
	timeout time.Duration
	// lookPath resolves the psql client; injected in tests.
	lookPath func() (string, error)
}

// NewProcessRunner returns the default production runner with a sane timeout.
func NewProcessRunner() Runner {
	return &processRunner{timeout: 60 * time.Second, lookPath: LookupPSQL}
}

func (p *processRunner) Run(ctx context.Context, endpoint Endpoint, args []string, stdin io.Reader, env []string) ([]byte, []byte, error) {
	executable, err := p.lookPath()
	if err != nil {
		return nil, nil, err
	}
	if p.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	}
	// The endpoint is prepended so a caller cannot accidentally omit it; the
	// caller's own -U/-d/-c arguments follow and win where psql allows it.
	invocation := append([]string{"--host", endpoint.Host, "--port", endpoint.Port}, args...)
	// #nosec G204 -- executing the psql client is this runner's whole purpose.
	// The path comes from LookupPSQL, which accepts only an operator-set
	// override or a file inside the staged artifact tree, and validates that it
	// is a regular executable file before it is used.
	command := exec.CommandContext(ctx, executable, invocation...)
	command.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.Resource, envkit.Env(env))
	if stdin != nil {
		command.Stdin = stdin
	}
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	runErr := command.Run()
	return stdout.Bytes(), stderr.Bytes(), runErr
}

// psqlExecutableEnv names an explicit psql client, overriding discovery.
const psqlExecutableEnv = "POSTGRES_PSQL"

// psqlName is the client basename for this platform.
func psqlName() string {
	if runtime.GOOS == "windows" {
		return "psql.exe"
	}
	return "psql"
}

// LookupPSQL resolves the psql client. It prefers the copy inside the staged
// artifact tree, which is the exact build the supervised server came from, and
// falls back to PATH so an operator with a host client is not blocked.
//
// The staged tree is searched rather than assumed: the archive layout differs
// per platform (a Debian usr/lib path on Linux, a version-named directory on
// macOS and Windows), and that variation is declared per target in
// resource.json rather than duplicated here.
func LookupPSQL() (string, error) {
	if override := strings.TrimSpace(os.Getenv(psqlExecutableEnv)); override != "" {
		if err := executableAt(override); err != nil {
			return "", fmt.Errorf("%s=%q is not usable: %w", psqlExecutableEnv, override, err)
		}
		return override, nil
	}
	if root := strings.TrimSpace(os.Getenv("RESOURCE_ARTIFACT_DIR")); root != "" {
		if found, err := findPSQL(root); err == nil {
			return found, nil
		}
	}
	if found, err := exec.LookPath(psqlName()); err == nil {
		return found, nil
	}
	return "", fmt.Errorf("psql client not found: set %s, or start the postgres resource so its artifact tree is staged", psqlExecutableEnv)
}

// findPSQL locates the client beside the server executable inside the staged
// tree, which is the same entry the manifest declares per target.
//
// Taking the first psql in the tree is wrong, and measurably so: Debian's
// PostgreSQL packaging installs a Perl wrapper named psql at usr/bin/psql that
// fails with "Can't locate PgCommon.pm" outside a Debian host, while the real
// client sits at usr/lib/postgresql/16/bin/psql. Requiring the server binary as
// a sibling selects the actual client directory on every layout this resource
// stages, without a per-platform path table here that would drift away from the
// manifest.
func findPSQL(root string) (string, error) {
	want := psqlName()
	server := serverName()
	var fallback string
	var found string
	// #nosec G703 -- root is operator-supplied configuration (the staged
	// artifact directory), and every path below is a walk result under it.
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || entry.Name() != want {
			return nil
		}
		if executableAt(path) != nil {
			return nil
		}
		if executableAt(filepath.Join(filepath.Dir(path), server)) == nil {
			found = path
			return filepath.SkipAll
		}
		if fallback == "" {
			fallback = path
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		found = fallback
	}
	if found == "" {
		return "", fmt.Errorf("no %s under %s", want, root)
	}
	return found, nil
}

// serverName is the server executable basename for this platform. It is the
// marker that identifies the real client directory inside a staged tree.
func serverName() string {
	if runtime.GOOS == "windows" {
		return "postgres.exe"
	}
	return "postgres"
}

func executableAt(path string) error {
	// #nosec G703 -- the path is operator-supplied configuration (an explicit
	// override) or a walk result under the staged artifact root, not untrusted
	// input. This function only stats it; it never opens or executes it.
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("path is a directory")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		return errors.New("file is not executable")
	}
	return nil
}
