package hostinventory

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/tuning"
)

const (
	credentialStoreOwnerTimeout       = tuning.ShortOperationTimeout
	credentialStoreCollectionsTimeout = tuning.HealthCheckTimeout
	credentialStoreCollectionTimeout  = tuning.ShortOperationTimeout
)

// CredentialStoreStatus probes the current user's Secret Service without
// running the full host inventory. It is the read-only status seam used by the
// credentials keyring command.
func CredentialStoreStatus(ctx context.Context) CredentialStoreCapability {
	c := SystemCollector().withDefaults()
	return probeCredentialStore(ctx, c, ActiveSessionUser(ctx, c.Commands))
}

// probeCredentialStore performs a real Secret Service property read. Peer.Ping
// is intentionally not used: a wedged service can answer it successfully.
func probeCredentialStore(ctx context.Context, c Collector, user string) CredentialStoreCapability {
	if hostreqspec.PlatformFromGOOS(c.GOOS) != hostreqspec.PlatformLinux {
		return CredentialStoreCapability{State: "unsupported", Reason: "Secret Service probing is Linux-only"}
	}
	if !c.commandAvailable("gdbus") {
		return CredentialStoreCapability{State: "unsupported", Reason: "gdbus is unavailable"}
	}
	if strings.TrimSpace(user) == "" {
		return CredentialStoreCapability{Supported: true, State: "unavailable", Reason: "no active graphical session user"}
	}
	uid := c.commandValue(ctx, "id", "-u", user)
	if uid == "" {
		return CredentialStoreCapability{Supported: true, State: "unavailable", Reason: "active session user has no resolvable uid"}
	}

	runtimeDir := "/run/user/" + uid
	envArgs := []string{
		"XDG_RUNTIME_DIR=" + runtimeDir,
		"DBUS_SESSION_BUS_ADDRESS=unix:path=" + runtimeDir + "/bus",
	}

	owner := runCredentialStoreProbe(ctx, c, credentialStoreOwnerTimeout, envArgs,
		"--dest", "org.freedesktop.DBus",
		"--object-path", "/org/freedesktop/DBus",
		"--method", "org.freedesktop.DBus.NameHasOwner", "org.freedesktop.secrets")
	if owner.err != nil {
		if owner.timedOut {
			return CredentialStoreCapability{Supported: true, Observed: true, State: "unresponsive", Reason: "Secret Service owner probe did not answer before the deadline"}
		}
		return CredentialStoreCapability{Supported: true, State: "unavailable", Reason: "Secret Service owner probe failed"}
	}
	if !strings.Contains(strings.ToLower(string(owner.output)), "true") {
		return CredentialStoreCapability{Supported: true, State: "unavailable", Reason: "Secret Service has no session-bus owner"}
	}

	collections := runCredentialStoreProbe(ctx, c, credentialStoreCollectionsTimeout, envArgs,
		"--dest", "org.freedesktop.secrets",
		"--object-path", "/org/freedesktop/secrets",
		"--method", "org.freedesktop.DBus.Properties.Get",
		"org.freedesktop.Secret.Service", "Collections")
	if collections.err != nil {
		if collections.timedOut {
			return CredentialStoreCapability{Supported: true, Observed: true, State: "unresponsive", Reason: "Secret Service Collections did not answer before the deadline"}
		}

		loginCollection := runCredentialStoreProbe(ctx, c, credentialStoreCollectionTimeout, envArgs,
			"--dest", "org.freedesktop.secrets",
			"--object-path", "/org/freedesktop/secrets/collection/login",
			"--method", "org.freedesktop.DBus.Properties.Get",
			"org.freedesktop.Secret.Collection", "Label")
		if loginCollection.err == nil && strings.TrimSpace(string(loginCollection.output)) != "" {
			return CredentialStoreCapability{Supported: true, Observed: true, State: "ready", ProbeSucceeded: true}
		}
		if loginCollection.timedOut {
			return CredentialStoreCapability{Supported: true, Observed: true, State: "unresponsive", Reason: "Secret Service login collection did not answer before the deadline"}
		}
		if isMissingCredentialCollection(string(loginCollection.output)) {
			return CredentialStoreCapability{Supported: true, Observed: true, State: "empty", ProbeSucceeded: true, Reason: "Secret Service has no readable login collection"}
		}
		return CredentialStoreCapability{Supported: true, Observed: true, State: "locked", Reason: "Secret Service owns the session bus, but the login collection requires a passphrase"}
	}

	text := string(collections.output)
	if strings.Contains(text, "/org/freedesktop/secrets/collection/login") {
		return CredentialStoreCapability{Supported: true, Observed: true, State: "ready", ProbeSucceeded: true}
	}
	return CredentialStoreCapability{Supported: true, Observed: true, State: "empty", ProbeSucceeded: true, Reason: "Secret Service returned no login collection"}
}

type credentialStoreProbeResult struct {
	output   []byte
	err      error
	timedOut bool
}

func runCredentialStoreProbe(parent context.Context, c Collector, timeout time.Duration, envArgs []string, args ...string) credentialStoreProbeResult {
	probeCtx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	var output []byte
	var err error
	if runner, ok := c.Commands.(EnvironmentCommandRunner); ok {
		// Keep gdbus as the process-group leader. The old `env gdbus ...`
		// wrapper could let a wedged gdbus process escape cancellation after
		// the wrapper exited and retain CombinedOutput's pipe.
		commandArgs := append([]string{"call", "--session"}, args...)
		output, err = runner.RunWithEnv(probeCtx, envArgs, "gdbus", commandArgs...)
	} else {
		// Test doubles predating EnvironmentCommandRunner still observe the
		// exact command shape they were written to assert.
		commandArgs := append(append([]string{}, envArgs...), "gdbus", "call", "--session")
		commandArgs = append(commandArgs, args...)
		output, err = c.Commands.Run(probeCtx, "env", commandArgs...)
	}
	return credentialStoreProbeResult{
		output:   output,
		err:      err,
		timedOut: errors.Is(probeCtx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded),
	}
}

func isMissingCredentialCollection(message string) bool {
	message = strings.ToLower(message)
	for _, marker := range []string{"unknownobject", "unknown object", "no such object", "does not exist", "not found"} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}
