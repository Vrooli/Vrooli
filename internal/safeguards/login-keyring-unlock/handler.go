// Package loginkeyringunlock makes an autologin user's GNOME login keyring
// passwordless. It is deliberately high-risk: any process running as that
// user can read the keyring's stored secrets once the master passphrase is
// removed. The safeguard never handles a remote-desktop credential.
package loginkeyringunlock

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vrooli/vrooli/internal/credentials"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

var (
	collectFactsFn = func() hostinventory.Snapshot { return hostinventory.CollectPlatformFacts(context.Background()) }
	keyringPathFn  = func(username string) (string, error) {
		account, err := user.Lookup(username)
		if err != nil {
			return "", err
		}
		return credentials.DefaultKeyringPath(filepath.Join(account.HomeDir, ".local", "share", "keyrings", "login.keyring"))
	}
	passwordlessFn  = credentials.IsPasswordless
	backupFn        = backupKeyring
	runUserFn       = hostreqkit.RunAsInvokingUserWithSession
	runUserOutputFn = hostreqkit.RunAsInvokingUserWithSessionOutput
)

var keyringPromptPathPattern = regexp.MustCompile(`['"](/org/gnome/keyring/Prompt/[A-Za-z0-9_./:@-]+)['"]`)

type handler struct{ manifest hostreqkit.SafeguardManifest }

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}
func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	if host.OS != "linux" {
		return hostreqkit.NotApplicableRequirementStatus(requirement, "the login-keyring unlock safeguard is Linux-only")
	}
	if requirement.OperatorChoice != hostreqspec.OperatorChoiceOptedIn {
		status.Notes = append(status.Notes, "inspection only: opt in to remove the login-keyring passphrase")
		return status
	}

	facts := collectFactsFn()
	if strings.TrimSpace(facts.AutoLoginUser) == "" {
		return hostreqkit.NotApplicableRequirementStatus(requirement, "no autologin user is configured; removing a login-keyring passphrase would buy nothing")
	}
	path, err := keyringPathFn(facts.AutoLoginUser)
	if err != nil {
		status.Notes = append(status.Notes, "cannot resolve the autologin user's login keyring: "+err.Error())
		return status
	}
	passwordless, err := passwordlessFn(path)
	if err != nil {
		if os.IsNotExist(err) {
			return hostreqkit.NotApplicableRequirementStatus(requirement, "the autologin user has no login keyring file to unlock")
		}
		status.Notes = append(status.Notes, "cannot inspect the login keyring: "+err.Error())
		return status
	}
	if passwordless {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "the autologin user's login keyring is already passwordless")
		return status
	}
	status.Notes = append(status.Notes, "login keyring is protected by a passphrase; apply will back it up before requesting a passwordless collection")
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if status.OperatorChoice != hostreqspec.OperatorChoiceOptedIn {
		status.ExecutionState = hostreqkit.ExecutionPending
		status.Notes = append(status.Notes, "inspection only: login-keyring unlock requires explicit operator opt-in")
		return status, nil
	}
	if status.SupportClass == hostreqkit.SupportNotApplicable || status.SupportClass == hostreqkit.SupportUnsupported {
		return status, nil
	}
	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}
	if host.OS != "linux" {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	}
	facts := collectFactsFn()
	if strings.TrimSpace(facts.AutoLoginUser) == "" {
		return hostreqkit.NotApplicableRequirementStatus(hostreqspec.ResolvedRequirement{Name: status.Name, Kind: status.Kind, Required: status.Required}, "no autologin user is configured"), nil
	}
	path, err := keyringPathFn(facts.AutoLoginUser)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "cannot resolve login keyring: "+err.Error())
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would back up the login keyring and request removal of its passphrase")
		return status, nil
	}
	backup, err := backupFn(path)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "refused to change the login keyring because its backup failed: "+err.Error())
		return status, nil
	}
	promptOutput, err := runUserOutputFn("gdbus", []string{
		"call", "--session", "--dest", "org.freedesktop.secrets",
		"--object-path", "/org/freedesktop/secrets",
		"--method", "org.gnome.keyring.InternalUnsupportedGuiltRiddenInterface.ChangeWithPrompt",
		"/org/freedesktop/secrets/collection/login",
	}, opts)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "login-keyring password-change prompt could not be opened: "+err.Error())
		return status, nil
	}
	promptPath := keyringPromptPathPattern.FindStringSubmatch(string(promptOutput))
	if len(promptPath) != 2 {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "login-keyring password-change prompt returned no valid prompt object")
		return status, nil
	}
	if err := runUserFn("gdbus", []string{
		"call", "--session", "--dest", "org.freedesktop.secrets",
		"--object-path", promptPath[1],
		"--method", "org.freedesktop.Secret.Prompt.Prompt", "",
	}, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "login-keyring password-change prompt could not be displayed: "+err.Error())
		return status, nil
	}
	status.ExecutionState = hostreqkit.ExecutionManualActionRequired
	status.BlockingReason = hostreqkit.BlockingManual
	status.Notes = append(status.Notes, "login-keyring password-change prompt opened; choose a blank new password and confirm, then rerun setup status to verify; backup saved at "+backup)
	return status, nil
}

func backupKeyring(path string) (string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read login keyring: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat login keyring: %w", err)
	}
	backup := path + ".vrooli-login-unlock-backup"
	file, err := os.OpenFile(backup, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return "", fmt.Errorf("create login-keyring backup: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(contents); err != nil {
		return "", fmt.Errorf("write login-keyring backup: %w", err)
	}
	if err := file.Sync(); err != nil {
		return "", fmt.Errorf("sync login-keyring backup: %w", err)
	}
	return backup, nil
}
