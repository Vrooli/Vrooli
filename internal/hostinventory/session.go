package hostinventory

import (
	"context"
	"strings"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/shell"
)

type SessionCommandRunner = shell.Runner

// SessionBusEnv returns the two environment assignments a Secret Service
// client needs to reach the session bus of the user with the given uid, in
// `KEY=value` form for env(1) or exec.Cmd.Env.
//
// This is the single place the runtime-directory layout is spelled out. Three
// callers used to build these strings independently (the credential-store
// probe, the autoheal session probe, and the securestore repair); a layout
// change or a validation rule now has one home. The securestore repair keeps
// its own guarded planner because it repairs *this process's* environment and
// must refuse when ownership cannot be proven; probes of another user's
// session cannot make that check and take the layout on trust.
func SessionBusEnv(uid string) []string {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return nil
	}
	runtimeDir := "/run/user/" + uid
	return []string{
		"XDG_RUNTIME_DIR=" + runtimeDir,
		"DBUS_SESSION_BUS_ADDRESS=unix:path=" + runtimeDir + "/bus",
	}
}

// Secret Service D-Bus identifiers shared by every gdbus probe.
const (
	SecretServiceBusName        = "org.freedesktop.secrets"
	SecretServiceObjectPath     = "/org/freedesktop/secrets"
	SecretServiceLoginPath      = SecretServiceObjectPath + "/collection/login"
	secretServiceInterface      = "org.freedesktop.Secret.Service"
	secretCollectionInterface   = "org.freedesktop.Secret.Collection"
	dbusPropertiesGetMethod     = "org.freedesktop.DBus.Properties.Get"
	dbusBusName                 = "org.freedesktop.DBus"
	dbusObjectPath              = "/org/freedesktop/DBus"
	dbusNameHasOwnerMethod      = "org.freedesktop.DBus.NameHasOwner"
	secretServiceCollectionsKey = "Collections"
	secretCollectionLabelKey    = "Label"
)

// SecretServiceOwnerArgs are the gdbus arguments (after `call --session`) that
// ask the bus whether anything owns the Secret Service name.
func SecretServiceOwnerArgs() []string {
	return []string{"--dest", dbusBusName, "--object-path", dbusObjectPath, "--method", dbusNameHasOwnerMethod, SecretServiceBusName}
}

// SecretServiceCollectionsArgs are the gdbus arguments (after `call --session`)
// that read the Collections property of the Secret Service.
func SecretServiceCollectionsArgs() []string {
	return []string{"--dest", SecretServiceBusName, "--object-path", SecretServiceObjectPath, "--method", dbusPropertiesGetMethod, secretServiceInterface, secretServiceCollectionsKey}
}

// SecretServiceCollectionLabelArgs are the gdbus arguments (after `call
// --session`) that read one collection's Label, which fails when the
// collection is advertised but its object never loaded.
func SecretServiceCollectionLabelArgs(collectionPath string) []string {
	return []string{"--dest", SecretServiceBusName, "--object-path", collectionPath, "--method", dbusPropertiesGetMethod, secretCollectionInterface, secretCollectionLabelKey}
}

// ActiveSessionUser resolves the user owning seat0 through the shared host
// inventory authority. Callers that need a session bus may separately resolve
// that user's UID without reimplementing session selection.
func ActiveSessionUser(ctx context.Context, commands SessionCommandRunner) string {
	probeCtx, cancel := context.WithTimeout(ctx, tuning.ServiceHealthTimeout())
	defer cancel()
	activeSession, err := commands.Run(probeCtx, "loginctl", "show-seat", "seat0", "-p", "ActiveSession", "--value")
	if err != nil {
		return ""
	}
	sessionID := strings.TrimSpace(string(activeSession))
	if sessionID == "" {
		return ""
	}
	user, err := commands.Run(probeCtx, "loginctl", "show-session", sessionID, "-p", "Name", "--value")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(user))
}
