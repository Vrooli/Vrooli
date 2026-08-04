// Package infra provides infrastructure health checks
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001] [REQ:HEAL-ACTION-001]
package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/journal"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// RDPType identifies which RDP implementation is in use
type RDPType string

const (
	RDPTypeXrdp        RDPType = "xrdp"
	RDPTypeGnome       RDPType = "gnome-remote-desktop"
	RDPTypeTermService RDPType = "TermService"
	RDPTypeUnknown     RDPType = "unknown"
)

// RDPServiceInfo describes which RDP service to check on a given platform
type RDPServiceInfo struct {
	ServiceName string
	Type        RDPType
	Checkable   bool
	// IsUserSession indicates if the RDP runs as a user session daemon (not systemd)
	IsUserSession bool
}

// SelectRDPService is a static helper that determines which RDP service to check
// based on platform capabilities WITHOUT runtime detection.
// This is used for tests and static analysis. For actual runtime detection
// (which checks for running processes), use RDPCheck.detectRDPService().
//
// Decision logic:
//   - Linux with systemd → xrdp service (static fallback)
//   - Linux without systemd → not checkable
//   - Windows → TermService (built-in RDP)
//   - Other platforms → not checkable
func SelectRDPService(caps *platform.Capabilities) RDPServiceInfo {
	switch caps.Platform {
	case platform.Linux:
		if caps.SupportsSystemd {
			// Note: Runtime detection in detectRDPService prefers GNOME RDP if running
			return RDPServiceInfo{
				ServiceName:   "xrdp",
				Type:          RDPTypeXrdp,
				Checkable:     true,
				IsUserSession: false,
			}
		}
		return RDPServiceInfo{
			ServiceName: "xrdp",
			Type:        RDPTypeXrdp,
			Checkable:   false,
		}
	case platform.Windows:
		return RDPServiceInfo{
			ServiceName:   "TermService",
			Type:          RDPTypeTermService,
			Checkable:     true,
			IsUserSession: false,
		}
	default:
		return RDPServiceInfo{Checkable: false}
	}
}

// RDPCheck verifies RDP service (xrdp, GNOME Remote Desktop, or Windows TermService).
// Platform capabilities are injected to avoid hidden dependencies and enable testing.
type RDPCheck struct {
	caps     *platform.Capabilities
	executor checks.CommandExecutor
	// autoLoginUserProvider is the auto-login discovery seam. It reads host
	// configuration outside the command executor, so it needs its own seam.
	autoLoginUserProvider func() string
	// cachedServiceInfo stores detected RDP service info to avoid repeated detection
	cachedServiceInfo *RDPServiceInfo
}

// RDPCheckOption configures an RDPCheck.
type RDPCheckOption func(*RDPCheck)

// WithRDPExecutor sets the command executor (for testing).
// [REQ:TEST-SEAM-001]
func WithRDPExecutor(executor checks.CommandExecutor) RDPCheckOption {
	return func(c *RDPCheck) {
		c.executor = executor
	}
}

// WithRDPAutoLoginUserProvider sets the auto-login discovery seam for testing.
// [REQ:TEST-SEAM-001]
func WithRDPAutoLoginUserProvider(provider func() string) RDPCheckOption {
	return func(c *RDPCheck) {
		c.autoLoginUserProvider = provider
	}
}

// NewRDPCheck creates an RDP health check with injected platform capabilities.
func NewRDPCheck(caps *platform.Capabilities, opts ...RDPCheckOption) *RDPCheck {
	c := &RDPCheck{
		caps:                  caps,
		executor:              checks.DefaultExecutor,
		autoLoginUserProvider: autoLoginUser,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// detectRDPService determines which RDP implementation is available on this system.
// Detection order on Linux:
//  1. Check if GNOME Remote Desktop is CONFIGURED (via grdctl) - catches crashed daemons
//  2. Check for running gnome-remote-desktop-daemon process (GNOME 42+ native RDP)
//  3. Check for xrdp systemd service
//  4. Fall back to unknown if neither is found
func (c *RDPCheck) detectRDPService(ctx context.Context) RDPServiceInfo {
	switch c.caps.Platform {
	case platform.Linux:
		// First check if GNOME Remote Desktop is CONFIGURED (not just running)
		// This catches the case where RDP is enabled but daemon has crashed
		if c.isGnomeRDPConfigured(ctx) {
			return RDPServiceInfo{
				ServiceName:   "gnome-remote-desktop",
				Type:          RDPTypeGnome,
				Checkable:     true,
				IsUserSession: true,
			}
		}

		// Then check for xrdp (traditional systemd service)
		if c.caps.SupportsSystemd {
			// Check if xrdp is installed/available
			output, err := c.executor.Output(ctx, "systemctl", "list-unit-files", "xrdp.service")
			if err == nil && strings.Contains(string(output), "xrdp.service") {
				return RDPServiceInfo{
					ServiceName:   "xrdp",
					Type:          RDPTypeXrdp,
					Checkable:     true,
					IsUserSession: false,
				}
			}
		}

		// No RDP service detected - this is OK, not all systems need RDP
		return RDPServiceInfo{
			Type:      RDPTypeUnknown,
			Checkable: false,
		}

	case platform.Windows:
		return RDPServiceInfo{
			ServiceName:   "TermService",
			Type:          RDPTypeTermService,
			Checkable:     true,
			IsUserSession: false,
		}

	default:
		return RDPServiceInfo{Checkable: false}
	}
}

// CredentialState classifies whether GNOME Remote Desktop holds RDP credentials
// that a remote client can authenticate against.
//
// The three states are deliberately distinct. A daemon that is running and
// listening still denies every client when its credentials are empty, and the
// probe itself can fail in a way that looks like success if the credential
// lines are simply absent from the output. Absence is never read as presence.
type CredentialState string

const (
	// CredentialStatePresent means both a username and a password are set.
	CredentialStatePresent CredentialState = "present"
	// CredentialStateEmpty means the daemon holds no usable credentials and
	// will deny every client that reaches authentication.
	CredentialStateEmpty CredentialState = "empty"
	// CredentialStateUnreadable means the credential state could not be
	// determined from the calling context. This never reports OK.
	CredentialStateUnreadable CredentialState = "unreadable"
)

// credentialProbeTimeout bounds the grdctl probe. A prior incident on a Vrooli
// host recorded a command whose pipe was held open by a supervisor and never
// returned, so every probe in this check carries an explicit deadline.
const credentialProbeTimeout = 10 * time.Second

// readGnomeRDPCredentialState determines whether GNOME Remote Desktop can
// authenticate a remote client.
//
// grdctl prints the credential lines only when it can reach the session user's
// D-Bus bus. Without that environment it still prints "Status: enabled" while
// omitting the credential lines entirely, so a naive grep for "(empty)" yields
// a false negative from the daemon's own context. The probe therefore runs with
// an explicit session-bus environment and treats missing lines as unreadable.
//
// This never passes --show-credentials and never records a credential value.
// Only the classified state is returned.
func (c *RDPCheck) readGnomeRDPCredentialState(ctx context.Context) CredentialState {
	ctx, cancel := context.WithTimeout(ctx, credentialProbeTimeout)
	defer cancel()

	env := sessionBusEnv(ctx, c.executor)
	if len(env) == 0 {
		// Without a resolvable session bus the credential state is unknowable
		// from here. Report that honestly rather than guessing.
		return CredentialStateUnreadable
	}

	args := append(append([]string{}, env...), "grdctl", "status")
	output, err := c.executor.CombinedOutput(ctx, "env", args...)
	if err != nil && len(output) == 0 {
		return CredentialStateUnreadable
	}

	return classifyCredentialOutput(string(output))
}

// classifyCredentialOutput maps grdctl status output onto a CredentialState.
// It is split out from the probe so the classification is directly testable.
func classifyCredentialOutput(output string) CredentialState {
	if strings.Contains(output, "Failed to read credentials") {
		return CredentialStateUnreadable
	}

	username, hasUsername := credentialFieldValue(output, "Username:")
	password, hasPassword := credentialFieldValue(output, "Password:")

	// Missing lines mean the probe could not see the credential store. Never
	// read absence as presence.
	if !hasUsername || !hasPassword {
		return CredentialStateUnreadable
	}

	if isEmptyCredentialValue(username) || isEmptyCredentialValue(password) {
		return CredentialStateEmpty
	}

	return CredentialStatePresent
}

// credentialFieldValue extracts the value of a `Field: value` line. It reports
// whether the line was present at all, which is the distinction between the
// empty and unreadable states.
func credentialFieldValue(output, field string) (string, bool) {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, field) {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(trimmed, field)), true
	}
	return "", false
}

// isEmptyCredentialValue reports whether grdctl signalled an unset credential.
func isEmptyCredentialValue(value string) bool {
	return value == "" || value == "(empty)" || value == "(null)"
}

// denialWindowMinutes is the fixed lookback for the client-denial signal.
const denialWindowMinutes = 15

// denialProbeTimeout bounds the journal read. A prior incident on a Vrooli host
// recorded a CombinedOutput call that never returned while a supervisor held
// the pipe, so this probe carries an explicit deadline.
const denialProbeTimeout = 15 * time.Second

// credentialDenialMarker is the daemon's message when it refuses a client for
// want of credentials.
const credentialDenialMarker = "Credentials are not set, denying client"

// denialCounts reports how many clients the daemon turned away recently.
type denialCounts struct {
	// CredentialDenials counts the specific "credentials are not set" refusal.
	CredentialDenials int
	// TotalDenials counts every refusal, as a broader secondary signal that
	// survives a change to the specific message text.
	TotalDenials int
	// Readable reports whether the journal could be read at all. A failed read
	// is not evidence of health.
	Readable bool
}

// recentRDPDenials counts clients the GNOME Remote Desktop daemon denied within
// the lookback window.
//
// A zero count is never evidence of health: it usually means nobody attempted a
// connection. Callers must only use a positive count to raise severity, never a
// zero count to lower it.
func (c *RDPCheck) recentRDPDenials(ctx context.Context) denialCounts {
	ctx, cancel := context.WithTimeout(ctx, denialProbeTimeout)
	defer cancel()

	entries, err := journal.NewReader(c.executor).QueryLogs(ctx, journal.QueryOpts{
		UserUnit: []string{"gnome-remote-desktop"},
		Since:    fmt.Sprintf("%d minutes ago", denialWindowMinutes),
	})
	if err != nil {
		return denialCounts{Readable: false}
	}

	counts := denialCounts{Readable: true}
	for _, entry := range entries {
		message := entry.Message
		if message == "" {
			message = entry.Raw
		}
		if strings.Contains(message, credentialDenialMarker) {
			counts.CredentialDenials++
		}
		if strings.Contains(message, "denying client") {
			counts.TotalDenials++
		}
	}
	return counts
}

// keyringFormatMarker is gnome-keyring-daemon's message when it refuses a
// keyring file outright.
//
// This signal exists because the check previously had no way to tell a locked
// keyring from an unloadable one, and the two are opposite faults with opposite
// remedies. A locked keyring is a keyring the daemon holds but cannot open; an
// unloadable one is a file the daemon discarded, taking every secret in it with
// it. On the host that motivated this code the daemon logged this line at boot
// and the check still reported "GDM autologin cannot unlock the login keyring",
// sending the operator to disable autologin — a change requiring root, costing
// them their session, and fixing nothing, because the file would not have
// parsed with a password either.
const keyringFormatMarker = "keyring was in an invalid or unrecognized format"

// keyringUnlockFailureMarker is the PAM module's report of the same event, from
// the login side. It is a corroborating signal only: it is also emitted for a
// genuinely locked keyring, so it can never identify the fault on its own.
const keyringUnlockFailureMarker = "couldn't unlock the login keyring"

// keyringJournalTimeout bounds the journal read, matching the denial probe. It
// is distinct from session.go's keyringProbeTimeout, which bounds a D-Bus call.
const keyringJournalTimeout = 15 * time.Second

// keyringLoadState reports whether the keyring daemon accepted its keyring
// files this boot.
type keyringLoadState struct {
	// Readable reports whether the journal could be read. A failed read is
	// never evidence that the keyring loaded.
	Readable bool
	// FormatRejected is the decisive signal: the daemon parsed a keyring file
	// and threw it away.
	FormatRejected bool
	// RejectedPath is the file the daemon named, when it named one.
	RejectedPath string
	// PAMUnlockFailed corroborates from the login side.
	PAMUnlockFailed bool
	// RepairPending means the file was rejected at boot but parses now: a
	// repair has landed and only a new login is missing.
	//
	// Without this the check could never go green after a successful repair.
	// The rejection is boot-scoped evidence and stays in the journal until
	// reboot, so a check reading only that signal would keep reporting a
	// malformed file that is no longer malformed, and an operator following its
	// advice would run the repair again and again.
	RepairPending bool
}

// keyringFileLoadable asks the Vrooli CLI whether a keyring file parses today.
//
// It shells out rather than parsing the file here on purpose: the format and
// the encoding written back belong to `securestore`, and a second reader in
// autoheal would be a copy that drifts from the writer it is meant to verify.
func (c *RDPCheck) keyringFileLoadable(ctx context.Context, path string) (bool, bool) {
	if path == "" {
		return false, false
	}
	ctx, cancel := context.WithTimeout(ctx, keyringJournalTimeout)
	defer cancel()

	output, err := c.executor.Output(ctx, "vrooli", "credentials", "keyring", "inspect", "--path", path, "--format", "json")
	if err != nil {
		return false, false
	}
	// The envelope shape is the CLI's contract; it is asserted on the producing
	// side too. An earlier version of this parser read a bare array, which is
	// what the command emitted before it grew a second field — and because the
	// test mocked the assumed shape rather than the emitted one, the mismatch
	// survived a green suite and only showed up against the live CLI.
	var payload struct {
		Reports []struct {
			Loadable bool `json:"loadable"`
		} `json:"reports"`
	}
	if err := json.Unmarshal(output, &payload); err != nil || len(payload.Reports) == 0 {
		return false, false
	}
	return payload.Reports[0].Loadable, true
}

// readKeyringLoadState asks the system journal whether gnome-keyring rejected a
// keyring file during this boot.
//
// The window is the boot rather than the denial probe's 15 minutes because the
// daemon reads its keyring files once, at session start. A 15-minute window
// would find this evidence only if a client happened to be denied within
// minutes of login, which is exactly when nobody is looking.
func (c *RDPCheck) readKeyringLoadState(ctx context.Context) keyringLoadState {
	ctx, cancel := context.WithTimeout(ctx, keyringJournalTimeout)
	defer cancel()

	entries, err := journal.NewReader(c.executor).QueryLogs(ctx, journal.QueryOpts{
		Boot: "0",
		Grep: "keyring",
	})
	if err != nil {
		return keyringLoadState{Readable: false}
	}

	state := keyringLoadState{Readable: true}
	for _, entry := range entries {
		message := entry.Message
		if message == "" {
			message = entry.Raw
		}
		if strings.Contains(message, keyringFormatMarker) {
			state.FormatRejected = true
			if path := keyringPathFromMessage(message); path != "" {
				state.RejectedPath = path
			}
		}
		if strings.Contains(message, keyringUnlockFailureMarker) {
			state.PAMUnlockFailed = true
		}
	}
	if state.FormatRejected {
		if loadable, known := c.keyringFileLoadable(ctx, state.RejectedPath); known && loadable {
			state.RepairPending = true
		}
	}
	return state
}

// keyringPathFromMessage extracts the file gnome-keyring named. The operator
// needs the path to repair the right file on a host with several keyrings.
func keyringPathFromMessage(message string) string {
	_, after, found := strings.Cut(message, keyringFormatMarker)
	if !found {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(after), ":"))
}

// corruptKeyringRemedies is the operator path out of an unloadable keyring.
//
// Every step is user-scoped. That is the substantive difference from
// keyringModelRemedies: those two remedies both require root and one of them
// destroys the operator's session, which is a heavy price for a check to
// recommend on evidence that turned out to be circumstantial.
func corruptKeyringRemedies(path string) []string {
	target := "the keyring file"
	if path != "" {
		target = path
	}
	return []string{
		"Run `vrooli credentials keyring inspect` to see which entries gnome-keyring rejected in " + target + ".",
		"Run `vrooli credentials keyring repair` to rewrite the malformed Vrooli-owned entries. It backs the file up first, declines entries other applications own, and needs no elevated privileges.",
		"Log out and back in, or reboot, so gnome-keyring-daemon reloads the repaired file.",
		"Confirm the RDP password survived with `grdctl status`; if it reads (empty), set it again with `grdctl rdp set-credentials <username> <password>`.",
	}
}

// repairPendingRemedies is the path for a host whose keyring file is already
// valid again. Re-running the repair here would be a no-op that reads like
// progress, so it is deliberately absent from this list.
func repairPendingRemedies(path string) []string {
	target := "the keyring file"
	if path != "" {
		target = path
	}
	return []string{
		"No further repair is needed: " + target + " parses correctly again.",
		"Log out and back in, or reboot, so gnome-keyring-daemon reloads it. Until then the running daemon still holds the file in its rejected state, and credential writes will hang on an unlock prompt nobody can answer.",
		"After logging back in, check `grdctl status`; if the RDP password reads (empty), set it again with `grdctl rdp set-credentials <username> <password>`.",
	}
}

// CredentialModel identifies where GNOME Remote Desktop keeps its RDP
// credentials, which determines whether automated repair is safe at all.
type CredentialModel string

const (
	// CredentialModelSystem means credentials live in the root-owned store of
	// the system gnome-remote-desktop.service. The failure has a deterministic,
	// non-interactive remedy and a real systemd unit to act on.
	CredentialModelSystem CredentialModel = "system"
	// CredentialModelUserSession means credentials live in the user's GNOME
	// login keyring. Autoheal cannot repair this: unlocking the keyring needs a
	// secret it must not hold, and writing a fresh password would mean autoheal
	// minting remote-access credentials on its own initiative.
	CredentialModelUserSession CredentialModel = "user-session"
)

// modelProbeTimeout bounds the credential-model probe.
const modelProbeTimeout = 10 * time.Second

// gnomeRDPCredentialModel reports which credential-storage model this host uses.
func (c *RDPCheck) gnomeRDPCredentialModel(ctx context.Context) CredentialModel {
	ctx, cancel := context.WithTimeout(ctx, modelProbeTimeout)
	defer cancel()

	output, err := c.executor.Output(ctx, "systemctl", "is-enabled", "gnome-remote-desktop.service")
	if err != nil {
		return CredentialModelUserSession
	}

	status := strings.TrimSpace(string(output))
	if status != "enabled" && status != "static" {
		return CredentialModelUserSession
	}

	// The unit exists and is enabled, but that alone does not prove the daemon
	// runs as a system service. Confirm it is active at the system level.
	active, err := c.executor.Output(ctx, "systemctl", "is-active", "gnome-remote-desktop.service")
	if err != nil || strings.TrimSpace(string(active)) != "active" {
		return CredentialModelUserSession
	}

	return CredentialModelSystem
}

// keyringModelRemedies returns the operator remedies for a host whose RDP
// credentials are locked in an unopened login keyring. Autoheal deliberately
// does not perform either of these.
func keyringModelRemedies() []string {
	return []string{
		"Disable GDM autologin in /etc/gdm3/custom.conf and log in interactively once, so pam_gnome_keyring unlocks the login keyring with the account password.",
		"Or migrate this host to the system-level gnome-remote-desktop.service credential store, where credentials do not depend on a user keyring and autoheal can repair the fault automatically.",
	}
}

// getAutoLoginUser returns the configured GDM auto-login user through the seam.
func (c *RDPCheck) getAutoLoginUser() string {
	if c.autoLoginUserProvider != nil {
		return c.autoLoginUserProvider()
	}
	return autoLoginUser()
}

// isGnomeRDPRunning checks if gnome-remote-desktop-daemon is running
func (c *RDPCheck) isGnomeRDPRunning(ctx context.Context) bool {
	// Check for the process using pgrep
	output, err := c.executor.Output(ctx, "pgrep", "-f", "gnome-remote-desktop-daemon")
	if err == nil && len(strings.TrimSpace(string(output))) > 0 {
		return true
	}
	return false
}

// isGnomeRDPConfigured checks if GNOME Remote Desktop is enabled in settings
// using grdctl status. This detects configuration even when the daemon isn't running.
func (c *RDPCheck) isGnomeRDPConfigured(ctx context.Context) bool {
	output, err := c.executor.Output(ctx, "grdctl", "status")
	if err != nil {
		return false
	}
	// grdctl status shows "Status: enabled" when RDP is configured
	return strings.Contains(string(output), "Status: enabled")
}

func (c *RDPCheck) ID() string    { return "infra-rdp" }
func (c *RDPCheck) Title() string { return "Remote Desktop" }
func (c *RDPCheck) Description() string {
	return "Checks RDP service (GNOME Remote Desktop, xrdp, or Windows TermService)"
}

func (c *RDPCheck) Importance() string {
	return "Required for remote desktop access to this machine"
}
func (c *RDPCheck) Category() checks.Category { return checks.CategoryInfrastructure }
func (c *RDPCheck) IntervalSeconds() int      { return 60 }
func (c *RDPCheck) Platforms() []platform.Type {
	return []platform.Type{platform.Linux, platform.Windows}
}

func (c *RDPCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	// Detect which RDP service is available on this system
	serviceInfo := c.detectRDPService(ctx)
	result.Details["service"] = serviceInfo.ServiceName
	result.Details["type"] = string(serviceInfo.Type)
	result.Details["isUserSession"] = serviceInfo.IsUserSession

	if !serviceInfo.Checkable {
		// No RDP service detected - this is informational, not a failure
		result.Status = checks.StatusOK
		result.Message = "No RDP service installed (remote desktop not configured)"
		result.Details["note"] = "RDP is optional; install xrdp or enable GNOME Remote Desktop if needed"
		return result
	}

	// Cache the service info for recovery actions
	c.cachedServiceInfo = &serviceInfo

	// Execute type-specific service check
	switch serviceInfo.Type {
	case RDPTypeGnome:
		return c.checkGnomeRDP(ctx, result)
	case RDPTypeXrdp:
		return c.checkLinuxXRDP(ctx, result)
	case RDPTypeTermService:
		return c.checkWindowsTermService(ctx, result)
	default:
		result.Status = checks.StatusOK
		result.Message = "No RDP service installed"
		return result
	}
}

// checkGnomeRDP verifies GNOME Remote Desktop daemon is running
func (c *RDPCheck) checkGnomeRDP(ctx context.Context, result checks.Result) checks.Result {
	isRunning := c.isGnomeRDPRunning(ctx)
	result.Details["configured"] = true
	result.Details["running"] = isRunning

	if !isRunning {
		// Configured but NOT running - this is a problem that needs attention
		result.Status = checks.StatusWarning
		result.Message = "GNOME Remote Desktop is configured but not running"
		result.Details["status"] = "inactive"
		result.Details["note"] = "Service may have crashed or been stopped; auto-heal can restart it"
		return result
	}

	result.Details["status"] = "active"

	// Get additional info about the daemon
	if output, _ := c.executor.Output(ctx, "pgrep", "-a", "-f", "gnome-remote-desktop-daemon"); len(output) > 0 {
		result.Details["processInfo"] = strings.TrimSpace(string(output))
	}

	// A running daemon is not a serviceable daemon. Clients that reach
	// authentication are denied when no credentials are set, so liveness alone
	// must never produce an OK verdict.
	credentialState := c.readGnomeRDPCredentialState(ctx)
	result.Details["credentialState"] = string(credentialState)

	// Ask the daemon's own journal whether clients are being turned away. This
	// distinguishes a latent misconfiguration from an in-progress lockout.
	denials := c.recentRDPDenials(ctx)
	result.Details["denialWindowMinutes"] = denialWindowMinutes
	result.Details["journalReadable"] = denials.Readable
	if denials.Readable {
		result.Details["recentDenials"] = denials.TotalDenials
		result.Details["recentCredentialDenials"] = denials.CredentialDenials
	}

	// Host posture. These are recorded on every run because they explain *why*
	// a credential fault exists, but posture alone never changes the status: a
	// host whose operator unlocked the keyring by hand matches the same posture
	// while working perfectly.
	_, sessionAvailable := graphicalSessionAvailable(ctx, c.executor, "")
	autoLogin := c.getAutoLoginUser()
	keyringPresent := loginKeyringCollectionPresent(ctx, c.executor)

	result.Details["sessionAvailable"] = sessionAvailable
	result.Details["autoLoginUser"] = autoLogin
	result.Details["loginKeyringCollectionPresent"] = keyringPresent

	// Did the keyring daemon actually accept its keyring files? This is asked
	// before the posture is interpreted, because an unloadable keyring produces
	// the same three posture booleans as a locked one while meaning something
	// else entirely.
	keyringLoad := c.readKeyringLoadState(ctx)
	result.Details["keyringJournalReadable"] = keyringLoad.Readable
	result.Details["keyringFileRejected"] = keyringLoad.FormatRejected
	result.Details["keyringUnlockFailureLogged"] = keyringLoad.PAMUnlockFailed
	result.Details["keyringRepairPending"] = keyringLoad.RepairPending
	if keyringLoad.RejectedPath != "" {
		result.Details["keyringFilePath"] = keyringLoad.RejectedPath
	}

	// The known-bad posture: an autologin host running a user-session daemon
	// whose login keyring never unlocked.
	isUserSession, _ := result.Details["isUserSession"].(bool)
	credentialFault := credentialState != CredentialStatePresent
	corruptKeyring := credentialFault && keyringLoad.FormatRejected
	result.Details["keyringCorrupt"] = corruptKeyring

	// The posture claim is withdrawn when the daemon has already said it threw
	// the file away. The three booleans below are still recorded as facts —
	// they are true — but "locked" is a diagnosis, and it is the wrong one for
	// a file that never loaded. Reporting both at once is what sent an operator
	// to disable autologin for a fault autologin did not cause.
	lockedKeyringPosture := autoLogin != "" && isUserSession && !keyringPresent && !keyringLoad.FormatRejected
	result.Details["lockedKeyringPosture"] = lockedKeyringPosture

	credentialModel := c.gnomeRDPCredentialModel(ctx)
	result.Details["credentialModel"] = string(credentialModel)

	// Carry the operator remedy on the result itself. The incident pipeline
	// reads these detail keys, so a durable incident inherits the remedy
	// without this check writing to the incident store directly.
	switch {
	case corruptKeyring:
		remedies := corruptKeyringRemedies(keyringLoad.RejectedPath)
		safe := "Run the infra-rdp repair-keyring action to rewrite the malformed entries and restore the keyring file."
		if keyringLoad.RepairPending {
			remedies = repairPendingRemedies(keyringLoad.RejectedPath)
			safe = "No repair action is needed; the file is already valid. Log out and back in so the keyring daemon reloads it."
		}
		result.Details["operatorActions"] = remedies
		result.Details["recommendations"] = remedies
		result.Details["safeActions"] = []string{safe}
		result.Details["postChecks"] = []string{
			"vrooli credentials keyring inspect",
			"vrooli-autoheal check get infra-rdp",
		}
	case credentialFault && credentialModel == CredentialModelUserSession:
		result.Details["operatorActions"] = keyringModelRemedies()
		result.Details["recommendations"] = keyringModelRemedies()
		result.Details["safeActions"] = []string{
			"Run the infra-rdp diagnose action to collect the credential state, autologin posture, and keyring evidence.",
		}
		result.Details["postChecks"] = []string{
			"vrooli-autoheal check get infra-rdp",
		}
	}

	switch credentialState {
	case CredentialStateEmpty:
		result.Status = checks.StatusCritical
		result.Message = "GNOME Remote Desktop is running but has no credentials set - every remote client is denied"
	case CredentialStateUnreadable:
		result.Status = checks.StatusWarning
		result.Message = "GNOME Remote Desktop is running but its credential state could not be read from this context"
	default:
		result.Status = checks.StatusOK
		result.Message = "GNOME Remote Desktop is running and has credentials set"
	}

	// Name the root cause. A rejected keyring file outranks the autologin
	// posture: it is the daemon's own statement about what happened, where the
	// posture is an inference from three booleans that are also true on hosts
	// where RDP works.
	switch {
	case corruptKeyring && keyringLoad.RepairPending:
		result.Message += " - the keyring file has been repaired and is valid again, but gnome-keyring rejected it at boot" +
			" and will not re-read it until the next login; log out and back in, then re-set the RDP password if it is still empty"
	case corruptKeyring:
		rejected := "its keyring file"
		if keyringLoad.RejectedPath != "" {
			rejected = keyringLoad.RejectedPath
		}
		result.Message += " - gnome-keyring rejected " + rejected +
			" as malformed, so no secret stored in it can be read; this is a file fault, not a locked keyring"
	case lockedKeyringPosture && credentialFault:
		result.Message += " - GDM autologin cannot unlock the login keyring, so the user-session daemon cannot read its credentials"
	}

	// Observed denials are proof of an outage in progress and outrank any
	// credential verdict. A zero count is never used to relax a verdict: it
	// only means no client tried.
	if denials.TotalDenials > 0 {
		result.Status = checks.StatusCritical
		result.Message = fmt.Sprintf(
			"GNOME Remote Desktop is actively denying clients (%d denial(s) in the last %d minutes)",
			denials.TotalDenials, denialWindowMinutes)
	}

	// A missing graphical session is the deepest cause this check can name, so
	// it takes the message last. Report it as RDP's own degradation rather than
	// restating display-manager health: infra-display owns that layer.
	if !sessionAvailable {
		result.Status = checks.StatusCritical
		result.Message = "GNOME Remote Desktop is degraded because no graphical session exists"
	}

	return result
}

// checkLinuxXRDP checks xrdp service status on Linux systems with systemd
func (c *RDPCheck) checkLinuxXRDP(ctx context.Context, result checks.Result) checks.Result {
	output, err := c.executor.Output(ctx, "systemctl", "is-active", "xrdp")
	status := strings.TrimSpace(string(output))
	result.Details["status"] = status

	if err != nil || status != "active" {
		result.Status = checks.StatusWarning
		result.Message = "xrdp service not active"
		return result
	}
	result.Status = checks.StatusOK
	result.Message = "xrdp is running"
	return result
}

// checkWindowsTermService checks TermService status on Windows
func (c *RDPCheck) checkWindowsTermService(ctx context.Context, result checks.Result) checks.Result {
	output, err := c.executor.Output(ctx, "sc", "query", "TermService")
	if err != nil {
		result.Status = checks.StatusWarning
		result.Message = "Unable to check RDP service"
		return result
	}

	// Decision: Windows sc query returns "STATE : X RUNNING" when service is running
	if strings.Contains(string(output), "RUNNING") {
		result.Status = checks.StatusOK
		result.Message = "RDP service is running"
	} else {
		result.Status = checks.StatusWarning
		result.Message = "RDP service not running"
	}
	return result
}

// RecoveryActions returns available recovery actions for RDP service issues
// [REQ:HEAL-ACTION-001]
func (c *RDPCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	// Use cached service info if available, otherwise detect
	var serviceInfo RDPServiceInfo
	if c.cachedServiceInfo != nil {
		serviceInfo = *c.cachedServiceInfo
	} else {
		serviceInfo = c.detectRDPService(context.Background())
	}

	isRunning := false
	if lastResult != nil {
		if status, ok := lastResult.Details["status"].(string); ok {
			isRunning = status == "active" || strings.Contains(status, "RUNNING")
		}
	}

	// Credential context drives which repair, if any, is appropriate.
	credentialModel := CredentialModelUserSession
	credentialState := CredentialState("")
	if lastResult != nil {
		if model, ok := lastResult.Details["credentialModel"].(string); ok && model != "" {
			credentialModel = CredentialModel(model)
		}
		if state, ok := lastResult.Details["credentialState"].(string); ok {
			credentialState = CredentialState(state)
		}
	}
	credentialFault := credentialState == CredentialStateEmpty || credentialState == CredentialStateUnreadable

	// A rejected keyring file is the one credential fault autoheal may repair on
	// the user-session model. The standing refusal exists so autoheal never
	// mints remote-access credentials it would have to hold a secret to create;
	// rewriting a malformed entry Vrooli itself wrote creates no credential,
	// reads no value, and needs no elevated privilege, so it falls outside the
	// reason for the refusal rather than being an exception to it.
	keyringCorrupt := false
	if lastResult != nil {
		keyringCorrupt, _ = lastResult.Details["keyringCorrupt"].(bool)
	}

	// Actions differ based on RDP type
	switch serviceInfo.Type {
	case RDPTypeGnome:
		// GNOME Remote Desktop is a user session daemon - we can still restart it via systemctl --user
		return []checks.RecoveryAction{
			{
				ID:          "start",
				Name:        "Start Service",
				Description: "Start the GNOME Remote Desktop user session service",
				Dangerous:   false,
				Available:   !isRunning,
			},
			{
				ID:   "restart",
				Name: "Restart Service",
				Description: "Restart the GNOME Remote Desktop user session service. " +
					"This cannot repair a credential fault, so it is offered only when the daemon is down.",
				Dangerous: false,
				Available: !isRunning,
			},
			{
				ID:   "repair-credentials",
				Name: "Repair Credentials",
				Description: "Reload RDP credentials from the system credential store. " +
					"Available only on the system-service model.",
				Dangerous: false,
				Available: credentialModel == CredentialModelSystem && credentialFault,
			},
			{
				ID:   "repair-keyring",
				Name: "Repair Corrupt Keyring",
				Description: "Rewrite the malformed Vrooli-owned entries in the keyring file gnome-keyring refused to load. " +
					"Backs the file up first, declines entries other applications own, and needs no elevated privileges. " +
					"A re-login is still required afterwards so the keyring daemon reloads the file.",
				Dangerous: false,
				Available: keyringCorrupt,
			},
			{
				ID:   "raise-incident",
				Name: "Report Credential Fault",
				Description: "Report the credential fault and its operator remedy. " +
					"Autoheal does not hold the secret needed to unlock the login keyring, so it takes no mutating action.",
				Dangerous: false,
				// Withdrawn when the fault is a corrupt keyring: that one has a
				// repair, and offering "report it" beside "fix it" invites the
				// operator to pick the useless one.
				Available: credentialModel == CredentialModelUserSession && credentialFault && !keyringCorrupt,
			},
			{
				ID:          "status",
				Name:        "Check Status",
				Description: "Get detailed GNOME Remote Desktop status",
				Dangerous:   false,
				Available:   true,
			},
			{
				ID:          "diagnose",
				Name:        "Diagnose",
				Description: "Gather diagnostic information about GNOME Remote Desktop",
				Dangerous:   false,
				Available:   true,
			},
			{
				ID:          "logs",
				Name:        "View Logs",
				Description: "View recent GNOME Remote Desktop logs",
				Dangerous:   false,
				Available:   true,
			},
			{
				ID:          "open-settings",
				Name:        "Open Settings",
				Description: "Show command to open GNOME Remote Desktop settings",
				Dangerous:   false,
				Available:   true,
			},
		}

	case RDPTypeXrdp:
		return []checks.RecoveryAction{
			{
				ID:          "start",
				Name:        "Start Service",
				Description: "Start the xrdp service",
				Dangerous:   false,
				Available:   !isRunning,
			},
			{
				ID:          "restart",
				Name:        "Restart Service",
				Description: "Restart the xrdp service",
				Dangerous:   false,
				Available:   true,
			},
			{
				ID:          "status",
				Name:        "Service Status",
				Description: "Get detailed xrdp service status",
				Dangerous:   false,
				Available:   true,
			},
			{
				ID:          "logs",
				Name:        "View Logs",
				Description: "View recent xrdp logs",
				Dangerous:   false,
				Available:   true,
			},
		}

	case RDPTypeTermService:
		return []checks.RecoveryAction{
			{
				ID:          "start",
				Name:        "Start Service",
				Description: "Start the Windows RDP service",
				Dangerous:   false,
				Available:   !isRunning,
			},
			{
				ID:          "restart",
				Name:        "Restart Service",
				Description: "Restart the Windows RDP service",
				Dangerous:   false,
				Available:   true,
			},
			{
				ID:          "status",
				Name:        "Service Status",
				Description: "Get detailed RDP service status",
				Dangerous:   false,
				Available:   true,
			},
		}

	default:
		// No RDP detected - provide informational action
		return []checks.RecoveryAction{
			{
				ID:          "install-info",
				Name:        "Installation Info",
				Description: "Show how to install RDP on this system",
				Dangerous:   false,
				Available:   true,
			},
		}
	}
}

// ExecuteAction runs the specified recovery action
// [REQ:HEAL-ACTION-001]
func (c *RDPCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	// Use cached service info if available, otherwise detect
	var serviceInfo RDPServiceInfo
	if c.cachedServiceInfo != nil {
		serviceInfo = *c.cachedServiceInfo
	} else {
		serviceInfo = c.detectRDPService(ctx)
	}

	// Handle actions based on RDP type
	switch actionID {
	case "status":
		return c.executeStatus(ctx, result, serviceInfo, start)

	case "diagnose":
		return c.executeDiagnose(ctx, result, serviceInfo, start)

	case "open-settings":
		return c.executeOpenSettings(ctx, result, start)

	case "install-info":
		return c.executeInstallInfo(ctx, result, start)

	case "start":
		if serviceInfo.Type == RDPTypeGnome {
			return c.executeGnomeRDPAction(ctx, result, "start", start)
		}
		return c.executeServiceAction(ctx, result, "start", serviceInfo, start)

	case "restart":
		if serviceInfo.Type == RDPTypeGnome {
			return c.executeGnomeRDPAction(ctx, result, "restart", start)
		}
		return c.executeServiceAction(ctx, result, "restart", serviceInfo, start)

	case "logs":
		return c.executeLogs(ctx, result, serviceInfo, start)

	case "repair-credentials":
		return c.executeRepairCredentials(ctx, result, start)

	case "repair-keyring":
		return c.executeRepairKeyring(ctx, result, start)

	case "raise-incident":
		return c.executeRaiseIncident(ctx, result, start)

	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// executeServiceAction starts or restarts the RDP service
func (c *RDPCheck) executeServiceAction(ctx context.Context, result checks.ActionResult, action string, serviceInfo RDPServiceInfo, start time.Time) checks.ActionResult {
	var output []byte
	var err error

	if c.caps.Platform == platform.Linux {
		output, err = c.executor.CombinedOutput(ctx, "sudo", "systemctl", action, serviceInfo.ServiceName)
	} else if c.caps.Platform == platform.Windows {
		// Windows: use sc command
		if action == "restart" {
			// Windows doesn't have restart, need to stop then start
			stopOutput, stopErr := c.executor.CombinedOutput(ctx, "sc", "stop", serviceInfo.ServiceName)
			if stopErr != nil {
				output = stopOutput
				err = stopErr
			} else {
				time.Sleep(2 * time.Second)
				output, err = c.executor.CombinedOutput(ctx, "sc", "start", serviceInfo.ServiceName)
			}
		} else {
			output, err = c.executor.CombinedOutput(ctx, "sc", action, serviceInfo.ServiceName)
		}
	}

	result.Output = string(output)

	if err != nil {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to " + action + " " + serviceInfo.ServiceName
		return result
	}

	// Verify service is running
	return c.verifyRecovery(ctx, result, action, start)
}

// executeGnomeRDPAction starts or restarts GNOME Remote Desktop user session service.
// GNOME Remote Desktop runs as a user session service (systemctl --user), not a system service.
// This function handles the complexity of running user-session commands with proper context.
func (c *RDPCheck) executeGnomeRDPAction(ctx context.Context, result checks.ActionResult, action string, start time.Time) checks.ActionResult {
	var output []byte
	var err error
	var outputBuilder strings.Builder

	actionTitle := action
	if len(action) > 0 {
		actionTitle = strings.ToUpper(action[:1]) + action[1:]
	}
	outputBuilder.WriteString(fmt.Sprintf("=== %s GNOME Remote Desktop ===\n", actionTitle))

	// GNOME Remote Desktop is a user session service.
	// We need to determine who owns the graphical session and run as that user.

	// First, try to find the active graphical session user
	sessionUser := c.findGraphicalSessionUser(ctx)
	if sessionUser == "" {
		// Fallback to current user or SUDO_USER
		sessionUser = os.Getenv("SUDO_USER")
		if sessionUser == "" {
			sessionUser = os.Getenv("USER")
		}
	}

	outputBuilder.WriteString(fmt.Sprintf("Target user: %s\n", sessionUser))

	// Get the user's UID for XDG_RUNTIME_DIR
	uidOutput, uidErr := c.executor.Output(ctx, "id", "-u", sessionUser)
	uid := strings.TrimSpace(string(uidOutput))
	if uidErr != nil || uid == "" {
		uid = "1000" // Common default for first user
	}
	xdgRuntimeDir := fmt.Sprintf("/run/user/%s", uid)

	outputBuilder.WriteString(fmt.Sprintf("XDG_RUNTIME_DIR: %s\n", xdgRuntimeDir))

	// Try different approaches to restart the service

	// Approach 1: Direct systemctl --user (works if we're already the correct user)
	output, err = c.executor.CombinedOutput(ctx, "systemctl", "--user", action, "gnome-remote-desktop")
	if err == nil {
		outputBuilder.WriteString("Method: direct systemctl --user\n")
		outputBuilder.WriteString(string(output))
		result.Output = outputBuilder.String()
		return c.verifyRecovery(ctx, result, action, start)
	}

	// Approach 2: Use sudo -u with XDG_RUNTIME_DIR and DBUS_SESSION_BUS_ADDRESS
	dbusAddr := fmt.Sprintf("unix:path=%s/bus", xdgRuntimeDir)
	output, err = c.executor.CombinedOutput(ctx,
		"sudo", "-u", sessionUser,
		fmt.Sprintf("XDG_RUNTIME_DIR=%s", xdgRuntimeDir),
		fmt.Sprintf("DBUS_SESSION_BUS_ADDRESS=%s", dbusAddr),
		"systemctl", "--user", action, "gnome-remote-desktop")

	if err == nil {
		outputBuilder.WriteString("Method: sudo -u with environment\n")
		outputBuilder.WriteString(string(output))
		result.Output = outputBuilder.String()
		return c.verifyRecovery(ctx, result, action, start)
	}

	// Approach 3: Use machinectl to enter user session (more robust for some setups)
	output, err = c.executor.CombinedOutput(ctx,
		"sudo", "machinectl", "shell", fmt.Sprintf("%s@.host", sessionUser),
		"/bin/systemctl", "--user", action, "gnome-remote-desktop")

	if err == nil {
		outputBuilder.WriteString("Method: machinectl shell\n")
		outputBuilder.WriteString(string(output))
		result.Output = outputBuilder.String()
		return c.verifyRecovery(ctx, result, action, start)
	}

	// All approaches failed
	outputBuilder.WriteString(fmt.Sprintf("\nAll restart methods failed. Last error: %v\n", err))
	outputBuilder.WriteString(string(output))

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = false
	result.Error = err.Error()
	result.Message = fmt.Sprintf("Failed to %s GNOME Remote Desktop - may need manual intervention", action)
	return result
}

// findGraphicalSessionUser finds the user who owns the active graphical session
func (c *RDPCheck) findGraphicalSessionUser(ctx context.Context) string {
	return seat0SessionUser(ctx, c.executor)
}

// repairActionTimeout bounds the credential repair action.
const repairActionTimeout = 60 * time.Second

// executeRepairCredentials reloads RDP credentials on the system-service model.
//
// It refuses the user-session model outright. There, repair would require
// either a secret autoheal must not hold (to unlock the login keyring) or
// minting a fresh remote-access password on autoheal's own initiative. That is
// a real expansion of blast radius and this check declines it.
//
// On the system model the repair is a restart of the system unit so the daemon
// re-reads its own root-owned credential store. Note that no credential value
// passes through autoheal at any point: the daemon reads its own store. Do not
// "improve" this by reading credentials out and writing them back, which would
// put a remote-access password in autoheal's memory, output, and logs.
func (c *RDPCheck) executeRepairCredentials(ctx context.Context, result checks.ActionResult, start time.Time) checks.ActionResult {
	ctx, cancel := context.WithTimeout(ctx, repairActionTimeout)
	defer cancel()

	model := c.gnomeRDPCredentialModel(ctx)
	if model != CredentialModelSystem {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = "repair-credentials is not available on the user-session credential model"
		result.Message = "Autoheal does not hold the secret needed to unlock the login keyring, " +
			"and will not create remote-access credentials on its own initiative. " +
			"Use the raise-incident action to surface the operator remedy."
		result.Output = strings.Join(keyringModelRemedies(), "\n")
		return result
	}

	output, err := c.executor.CombinedOutput(ctx, "sudo", "systemctl", "restart", "gnome-remote-desktop.service")
	result.Output = string(output)
	if err != nil {
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to reload credentials from the system credential store"
		return result
	}

	return c.verifyRecovery(ctx, result, "repair-credentials", start)
}

// executeRaiseIncident reports the credential fault and its operator remedy.
//
// This action is deliberately non-mutating. The durable incident record is
// raised by the incident pipeline from this check's non-OK result, which also
// resolves it automatically once the check returns OK again. This action exists
// so an operator can read the full remedy on demand without waiting for the
// next incident sweep.
// executeRepairKeyring delegates to the Vrooli CLI, which owns the keyring file
// format and the encoding that must be written back.
//
// Autoheal deliberately does not reimplement the repair. The correct rewritten
// form is whatever `securestore` would write today, and a second implementation
// here would be a copy that drifts — which is precisely how a file this check
// is supposed to protect would get corrupted a second time, by the supervisor
// meant to prevent it.
func (c *RDPCheck) executeRepairKeyring(ctx context.Context, result checks.ActionResult, start time.Time) checks.ActionResult {
	ctx, cancel := context.WithTimeout(ctx, repairActionTimeout)
	defer cancel()

	output, err := c.executor.CombinedOutput(ctx, "vrooli", "credentials", "keyring", "repair")
	result.Output = strings.TrimSpace(string(output))
	result.Duration = time.Since(start)
	if err != nil {
		result.Success = false
		result.Message = "Keyring repair failed: " + err.Error()
		return result
	}

	result.Success = true
	// The repair fixes the file; only a new login makes the daemon read it. A
	// message that stopped at "repaired" would leave the operator watching a
	// check that stays critical and concluding the repair did not work.
	result.Message = "Repaired the keyring file. Log out and back in, or reboot, so gnome-keyring-daemon reloads it; " +
		"then confirm the RDP password with `grdctl status` and set it again if it reads (empty)."
	return result
}

func (c *RDPCheck) executeRaiseIncident(ctx context.Context, result checks.ActionResult, start time.Time) checks.ActionResult {
	var out strings.Builder

	credentialState := c.readGnomeRDPCredentialState(ctx)
	autoLogin := c.getAutoLoginUser()
	keyringPresent := loginKeyringCollectionPresent(ctx, c.executor)

	out.WriteString("=== GNOME Remote Desktop credential fault ===\n\n")
	out.WriteString(fmt.Sprintf("Credential state: %s\n", credentialState))
	if autoLogin != "" {
		out.WriteString(fmt.Sprintf("GDM autologin: enabled for %s\n", autoLogin))
	} else {
		out.WriteString("GDM autologin: not configured\n")
	}
	out.WriteString(fmt.Sprintf("Login keyring collection present: %v\n", keyringPresent))
	out.WriteString(fmt.Sprintf("Credential model: %s\n\n", CredentialModelUserSession))

	out.WriteString("Operator remedies:\n")
	for i, remedy := range keyringModelRemedies() {
		out.WriteString(fmt.Sprintf("  %d. %s\n", i+1, remedy))
	}
	out.WriteString("\nAutoheal takes no mutating action here by design. " +
		"It does not hold the secret needed to unlock the login keyring, " +
		"and it will not create remote-access credentials on its own initiative.\n")

	result.Duration = time.Since(start)
	result.Output = out.String()
	result.Success = true
	result.Message = "Credential fault reported with operator remedy; no mutating action taken"
	return result
}

// executeStatus gets detailed service status
func (c *RDPCheck) executeStatus(ctx context.Context, result checks.ActionResult, serviceInfo RDPServiceInfo, start time.Time) checks.ActionResult {
	var outputBuilder strings.Builder

	switch serviceInfo.Type {
	case RDPTypeGnome:
		outputBuilder.WriteString("=== GNOME Remote Desktop Status ===\n")

		// Check process status
		output, err := c.executor.CombinedOutput(ctx, "pgrep", "-a", "-f", "gnome-remote-desktop-daemon")
		if err == nil && len(output) > 0 {
			outputBuilder.WriteString("Process: RUNNING\n")
			outputBuilder.WriteString(string(output))
		} else {
			outputBuilder.WriteString("Process: NOT RUNNING\n")
		}

		// Check port 3389
		outputBuilder.WriteString("\n=== Port 3389 Status ===\n")
		portOutput, _ := c.executor.CombinedOutput(ctx, "ss", "-tlnp", "sport = :3389")
		if len(portOutput) > 0 {
			outputBuilder.Write(portOutput)
		} else {
			outputBuilder.WriteString("No listener on port 3389\n")
		}

	case RDPTypeXrdp:
		output, _ := c.executor.CombinedOutput(ctx, "systemctl", "status", "xrdp")
		outputBuilder.Write(output)

	case RDPTypeTermService:
		output, _ := c.executor.CombinedOutput(ctx, "sc", "query", "TermService")
		outputBuilder.Write(output)

	default:
		outputBuilder.WriteString("No RDP service detected on this system.\n")
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Service status retrieved"
	return result
}

// executeDiagnose gathers diagnostic information about RDP
func (c *RDPCheck) executeDiagnose(ctx context.Context, result checks.ActionResult, serviceInfo RDPServiceInfo, start time.Time) checks.ActionResult {
	var outputBuilder strings.Builder

	outputBuilder.WriteString("=== RDP Diagnostics ===\n")
	outputBuilder.WriteString(fmt.Sprintf("Detected RDP Type: %s\n", serviceInfo.Type))
	outputBuilder.WriteString(fmt.Sprintf("Service Name: %s\n", serviceInfo.ServiceName))
	outputBuilder.WriteString(fmt.Sprintf("User Session: %v\n\n", serviceInfo.IsUserSession))

	// Check port 3389 listener
	outputBuilder.WriteString("=== Port 3389 Status ===\n")
	portOutput, _ := c.executor.CombinedOutput(ctx, "ss", "-tlnp", "sport = :3389")
	if len(portOutput) > 0 {
		outputBuilder.Write(portOutput)
	} else {
		outputBuilder.WriteString("No listener on port 3389\n")
	}

	// Network interfaces
	outputBuilder.WriteString("\n=== Network Interfaces ===\n")
	ifOutput, _ := c.executor.CombinedOutput(ctx, "ip", "addr", "show")
	if len(ifOutput) > 0 {
		// Just show the first few interfaces
		lines := strings.Split(string(ifOutput), "\n")
		for i, line := range lines {
			if i >= 20 {
				outputBuilder.WriteString("...(truncated)\n")
				break
			}
			outputBuilder.WriteString(line + "\n")
		}
	}

	// Firewall status
	outputBuilder.WriteString("\n=== Firewall Port 3389 ===\n")
	if c.caps.Platform == platform.Linux {
		fwOutput, _ := c.executor.CombinedOutput(ctx, "sudo", "iptables", "-L", "-n", "--line-numbers")
		if strings.Contains(string(fwOutput), "3389") {
			outputBuilder.WriteString("Port 3389 found in iptables rules\n")
		} else {
			// Check ufw if iptables doesn't show it
			ufwOutput, _ := c.executor.CombinedOutput(ctx, "sudo", "ufw", "status")
			if strings.Contains(string(ufwOutput), "3389") {
				outputBuilder.WriteString("Port 3389 found in UFW rules\n")
			} else {
				outputBuilder.WriteString("Port 3389 not explicitly allowed (may use default policy)\n")
			}
		}
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Diagnostic information gathered"
	return result
}

// executeOpenSettings provides information about opening RDP settings
func (c *RDPCheck) executeOpenSettings(ctx context.Context, result checks.ActionResult, start time.Time) checks.ActionResult {
	var outputBuilder strings.Builder

	outputBuilder.WriteString("=== GNOME Remote Desktop Settings ===\n\n")
	outputBuilder.WriteString("To configure GNOME Remote Desktop:\n\n")
	outputBuilder.WriteString("1. Open GNOME Settings:\n")
	outputBuilder.WriteString("   gnome-control-center sharing\n\n")
	outputBuilder.WriteString("2. Or navigate to:\n")
	outputBuilder.WriteString("   Settings → Sharing → Remote Desktop\n\n")
	outputBuilder.WriteString("3. Enable 'Remote Desktop' toggle\n")
	outputBuilder.WriteString("4. Configure username and password\n")
	outputBuilder.WriteString("5. Optionally enable 'Remote Control' for input\n")

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Settings information provided"
	return result
}

// executeInstallInfo provides information about installing RDP
func (c *RDPCheck) executeInstallInfo(ctx context.Context, result checks.ActionResult, start time.Time) checks.ActionResult {
	var outputBuilder strings.Builder

	outputBuilder.WriteString("=== RDP Installation Options ===\n\n")

	if c.caps.Platform == platform.Linux {
		outputBuilder.WriteString("Option 1: GNOME Remote Desktop (recommended for GNOME desktop)\n")
		outputBuilder.WriteString("  • Built into GNOME 42+\n")
		outputBuilder.WriteString("  • Enable in Settings → Sharing → Remote Desktop\n\n")

		outputBuilder.WriteString("Option 2: xrdp (traditional RDP server)\n")
		outputBuilder.WriteString("  • Install: sudo apt install xrdp\n")
		outputBuilder.WriteString("  • Start: sudo systemctl enable --now xrdp\n")
		outputBuilder.WriteString("  • Note: May conflict with GNOME Remote Desktop\n")
	} else if c.caps.Platform == platform.Windows {
		outputBuilder.WriteString("Windows Remote Desktop:\n")
		outputBuilder.WriteString("  • Enable in Settings → System → Remote Desktop\n")
		outputBuilder.WriteString("  • Or: sysdm.cpl → Remote tab\n")
	} else {
		outputBuilder.WriteString("RDP is not typically available on this platform.\n")
		outputBuilder.WriteString("Consider using VNC or SSH instead.\n")
	}

	result.Duration = time.Since(start)
	result.Output = outputBuilder.String()
	result.Success = true
	result.Message = "Installation information provided"
	return result
}

// executeLogs gets recent service logs
func (c *RDPCheck) executeLogs(ctx context.Context, result checks.ActionResult, serviceInfo RDPServiceInfo, start time.Time) checks.ActionResult {
	var output []byte
	var err error

	switch serviceInfo.Type {
	case RDPTypeGnome:
		// GNOME Remote Desktop logs to user journal
		output, err = journal.NewReader(c.executor).Tail(ctx, journal.QueryOpts{
			UserUnit: []string{"gnome-remote-desktop"},
			Tail:     100,
		})
		if err != nil || len(output) == 0 {
			// Fallback to grep in syslog
			output, err = c.executor.CombinedOutput(ctx, "grep", "-i", "gnome-remote-desktop", "/var/log/syslog")
		}

	case RDPTypeXrdp:
		output, err = journal.NewReader(c.executor).Tail(ctx, journal.QueryOpts{
			Unit: []string{"xrdp"},
			Tail: 100,
		})

	case RDPTypeTermService:
		// Windows event logs would need different approach
		output = []byte("Windows event logs require Event Viewer. Run: eventvwr.msc")
		err = nil

	default:
		output = []byte("No RDP service detected to retrieve logs from")
		err = nil
	}

	result.Duration = time.Since(start)
	result.Output = string(output)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to retrieve logs"
		return result
	}

	result.Success = true
	result.Message = "Service logs retrieved"
	return result
}

// verifyRecovery checks that RDP is working after a recovery action using polling
func (c *RDPCheck) verifyRecovery(ctx context.Context, result checks.ActionResult, action string, start time.Time) checks.ActionResult {
	// Use polling to verify recovery instead of fixed sleep
	pollConfig := checks.PollConfig{
		Timeout:      30 * time.Second,
		Interval:     2 * time.Second,
		InitialDelay: 3 * time.Second, // Service needs time to initialize
	}

	pollResult := checks.PollForSuccess(ctx, c, pollConfig)
	result.Duration = time.Since(start)

	if pollResult.Success {
		result.Success = true
		result.Message = "RDP service " + action + " successful and verified running"
		if pollResult.FinalResult != nil {
			result.Output += "\n\n=== Verification ===\n" + pollResult.FinalResult.Message
		}
		result.Output += fmt.Sprintf("\n(verified after %d attempts in %s)", pollResult.Attempts, pollResult.Elapsed.Round(time.Millisecond))
	} else {
		result.Success = false
		result.Error = "RDP not running after " + action
		result.Message = "RDP service " + action + " completed but verification failed"
		if pollResult.FinalResult != nil {
			result.Output += "\n\n=== Verification Failed ===\n" + pollResult.FinalResult.Message
		}
		result.Output += fmt.Sprintf("\n(failed after %d attempts in %s)", pollResult.Attempts, pollResult.Elapsed.Round(time.Millisecond))
	}

	return result
}

// Ensure RDPCheck implements HealableCheck
var _ checks.HealableCheck = (*RDPCheck)(nil)
