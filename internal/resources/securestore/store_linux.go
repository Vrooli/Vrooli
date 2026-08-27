//go:build linux

package securestore

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tuning"
)

// nativeDefault uses libsecret's Secret Service command client. If no desktop
// keyring is available it fails closed, so a headless Linux target stays
// conditional instead of leaving Vault recovery material on disk.
func nativeDefault() Store {
	if _, err := exec.LookPath("secret-tool"); err != nil {
		return Absent("secret-tool (libsecret) is not installed")
	}
	return &secretToolStore{}
}

type secretToolStore struct {
	collectionsOnce sync.Once
	collectionsErr  error
}

func (*secretToolStore) AdapterName() string { return "libsecret" }

// secretToolTimeout bounds every Secret Service call.
//
// Without it, a host whose keyring will not load has no timeout at all:
// gnome-keyring answers a write by raising an unlock prompt, and on a machine
// being reached over SSH or driven by a supervisor there is nobody to dismiss
// it. The call then blocks forever. That is not a hypothetical — it is how the
// keyring corruption this package now repairs presented itself: not as a failed
// credential write, but as a `vrooli` process that never returned.
//
// A failure here is recoverable and an operator can see it. A hang is neither,
// and it takes autoheal down with it, since a supervisor that blocks on a
// credential read cannot repair the thing that is blocking it.
var secretToolTimeout = tuning.CredentialServiceTimeout()

var collectionPathPattern = regexp.MustCompile(`/org/freedesktop/secrets/collection/[A-Za-z0-9_.-]+`)

// runSecretServiceCommand is the command seam shared by the collection health
// probe and its tests. It uses gdbus instead of adding a D-Bus library.
var runSecretServiceCommand = func(ctx context.Context, args ...string) (string, string, error) {
	cmd := shell.NewCommandContext(ctx, "gdbus", args...)
	cmd.Env = sessionEnviron()
	var stderr strings.Builder
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	return string(stdout), stderr.String(), err
}

// secretToolCommand builds every Secret Service invocation, so all three
// operations reach the same bus and share one timeout. sessionEnviron repairs a
// session that names another user's bus; it returns nil when nothing needs
// repair, and a nil Env already means "inherit" to exec.
//
// The caller must invoke the returned cancel func once the command has
// finished.
func secretToolCommand(args ...string) (*exec.Cmd, context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), tuning.SecretToolTimeout())
	cmd := shell.NewCommandContext(ctx, "secret-tool", args...)
	cmd.Env = sessionEnviron()
	return cmd, ctx, cancel
}

// classifySecretToolTimeout turns the generic "signal: killed" that a context
// deadline produces into the one condition an operator can act on.
func classifySecretToolTimeout(stage string) error {
	return fmt.Errorf("%w: %s: the Secret Service did not answer within %s, which usually means it is waiting on an unlock prompt nobody can answer; run `secrets-manager keyring inspect`",
		ErrUnavailable, stage, secretToolTimeout)
}

func (*secretToolStore) Put(service, key, value string) error {
	cmd, ctx, cancel := secretToolCommand("store", "--label=Vrooli managed resource", "service", service, "key", key)
	defer cancel()
	// The value goes over stdin so it never appears in argv, /proc, or shell
	// history. Every adapter on every platform owes the same guarantee.
	cmd.Stdin = strings.NewReader(value)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return classifySecretToolTimeout("store secure resource material")
	}
	return classifySecretToolError("store secure resource material", err, string(output))
}

func (store *secretToolStore) Get(service, key string) (string, error) {
	cmd, ctx, cancel := secretToolCommand("lookup", "service", service, "key", key)
	defer cancel()
	var stderr strings.Builder
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	if err == nil {
		return strings.TrimSuffix(string(stdout), "\n"), nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "", classifySecretToolTimeout("read secure resource material")
	}
	if isSecretToolNotFound(err, len(stdout), stderr.String()) {
		if healthErr := store.collectionHealth(); healthErr != nil {
			return "", healthErr
		}
		return "", fmt.Errorf("%w: %s/%s", ErrNotFound, service, key)
	}
	return "", classifySecretToolError("read secure resource material", err, stderr.String())
}

func (store *secretToolStore) Delete(service, key string) error {
	cmd, ctx, cancel := secretToolCommand("clear", "service", service, "key", key)
	defer cancel()
	var stderr strings.Builder
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	if err == nil {
		return nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return classifySecretToolTimeout("delete secure resource material")
	}
	// Removing what is already gone is the state the caller asked for, and
	// every adapter must agree on that. The shared conformance suite caught
	// this adapter disagreeing.
	if isSecretToolNotFound(err, len(stdout), stderr.String()) {
		if healthErr := store.collectionHealth(); healthErr != nil {
			return healthErr
		}
		return nil
	}
	return classifySecretToolError("delete secure resource material", err, stderr.String())
}

// staleKeyringDaemonRemedy is the one action that clears a half-loaded
// collection. The daemon holds the keyring it parsed at login, so repairing the
// file underneath it changes nothing until a new session reloads it — which is
// precisely the state a host lands in right after `credentials keyring repair`.
const staleKeyringDaemonRemedy = "log out and back in so the keyring daemon reloads the keyring file; it is still serving the copy it parsed at login"

// collectionHealth checks advertised collections only after an ambiguous empty
// lookup. A collection can remain in Collections after its object failed to
// load; that state is unavailable, not an empty credential store.
//
// The verdict is computed once per store (collectionsOnce) and is the store's
// answer, not one read's answer. Reporting it only on the path that happened to
// ask is what let one credential read "configured" beside its siblings reading
// "provider_unavailable", from the same store, in the same table.
func (store *secretToolStore) collectionHealth() error {
	store.collectionsOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), tuning.SecretToolTimeout())
		defer cancel()
		stdout, stderr, err := runSecretServiceCommand(ctx,
			"call", "--session", "--dest", "org.freedesktop.secrets",
			"--object-path", "/org/freedesktop/secrets", "--method",
			"org.freedesktop.DBus.Properties.Get", "org.freedesktop.Secret.Service", "Collections")
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			store.collectionsErr = classifySecretToolTimeout("check Secret Service collections")
			return
		}
		if err != nil {
			store.collectionsErr = fmt.Errorf("%w: check Secret Service collections: %s", ErrUnavailable, conciseCommandError(err, stderr))
			return
		}
		collections := collectionPathPattern.FindAllString(stdout, -1)
		for _, collection := range collections {
			ctx, cancel := context.WithTimeout(context.Background(), tuning.SecretToolTimeout())
			_, collectionStderr, collectionErr := runSecretServiceCommand(ctx,
				"call", "--session", "--dest", "org.freedesktop.secrets",
				"--object-path", collection, "--method",
				"org.freedesktop.DBus.Properties.Get", "org.freedesktop.Secret.Collection", "Label")
			cancel()
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				store.collectionsErr = classifySecretToolTimeout("check Secret Service collection " + collection)
				return
			}
			if collectionErr != nil {
				store.collectionsErr = withRemediation(
					fmt.Errorf("%w: collection %s is advertised but its object is not available: %s",
						ErrUnavailable, collection, conciseCommandError(collectionErr, collectionStderr)),
					staleKeyringDaemonRemedy)
				return
			}
		}
	})
	return store.collectionsErr
}

func conciseCommandError(err error, stderr string) string {
	if detail := strings.TrimSpace(stderr); detail != "" {
		return detail
	}
	return err.Error()
}

// isSecretToolNotFound recognizes the one way libsecret says "the Secret
// Service answered and holds no such item": exit status 1 with no output on
// either stream. A connection or protocol failure always writes to stderr, so
// the two cannot be confused — and conflating them is what made an unset API
// key abort a scenario start.
func isSecretToolNotFound(err error, stdoutLen int, stderr string) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) &&
		exitErr.ExitCode() == 1 &&
		stdoutLen == 0 &&
		strings.TrimSpace(stderr) == "" &&
		len(exitErr.Stderr) == 0
}

// classifySecretToolError keeps Put, Get, and Delete answering identically for
// identical host conditions. Before this, an uninstalled secret-tool produced
// ErrUnavailable from Delete and a bare exec error from Put and Get.
func classifySecretToolError(stage string, err error, output string) error {
	detail := strings.TrimSpace(output)
	if errors.Is(err, exec.ErrNotFound) {
		return fmt.Errorf("%w: secret-tool (libsecret) is not installed", ErrAbsent)
	}
	message := stage + ": " + err.Error()
	if detail != "" {
		message += ": " + detail
	}
	if diagnosis := sessionDiagnosis(); diagnosis != "" {
		message += " (" + diagnosis + ")"
	}
	return fmt.Errorf("%w: %s", ErrUnavailable, message)
}
