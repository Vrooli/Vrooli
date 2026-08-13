// Package infra provides infrastructure health checks
// [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001] [REQ:HEAL-ACTION-001]
package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/journal"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// RDPCheck verifies RDP service (xrdp, GNOME Remote Desktop, or Windows TermService).
// Platform capabilities are injected to avoid hidden dependencies and enable testing.
type RDPCheck struct {
	caps                 *platform.Capabilities
	executor             checks.CommandExecutor
	desiredStateProvider RemoteDesktopIntentProvider
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

// WithRDPDesiredStateProvider supplies the read-only resolved host contract.
// Tests inject a fake requirement; production wiring uses the control-plane
// resolver without giving this check any mutation authority.
func WithRDPDesiredStateProvider(provider RemoteDesktopIntentProvider) RDPCheckOption {
	return func(c *RDPCheck) {
		c.desiredStateProvider = provider
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

	output, err := keyringInspectOutput(ctx, path)
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

	if c.desiredStateProvider != nil {
		intent, err := c.desiredStateProvider(ctx)
		if err != nil {
			result.Status = checks.StatusOK
			result.Message = "Remote desktop is unmanaged because declared intent could not be resolved"
			result.Details["desiredVerdict"] = RemoteDesktopVerdictUnmanaged
			result.Details["desiredStateError"] = err.Error()
			result.Details["note"] = "No recovery action is available until the host requirement resolution is readable"
			return result
		}
		result.Details["desiredExperience"] = intent.Experience
		result.Details["desiredProvider"] = intent.Provider
		verdict := remoteDesktopIntentVerdict(intent, serviceInfo)
		result.Details["desiredVerdict"] = verdict
		if verdict == RemoteDesktopVerdictUnmanaged {
			result.Status = checks.StatusOK
			result.Message = fmt.Sprintf("Remote desktop is unmanaged (observed %s); no recovery action will be taken", observedRemoteDesktopExperience(serviceInfo))
			return result
		}
		if intent.Experience == "observe-only" {
			result.Status = checks.StatusOK
			result.Message = fmt.Sprintf("Remote desktop is observe-only (observed %s)", observedRemoteDesktopExperience(serviceInfo))
			return result
		}
		result.Details["observedExperience"] = observedRemoteDesktopExperience(serviceInfo)
		result.Details["observedProvider"] = observedRemoteDesktopProvider(serviceInfo)
		if verdict == RemoteDesktopVerdictDrifted && serviceInfo.Checkable {
			result.Status = checks.StatusWarning
			result.Message = fmt.Sprintf("Remote desktop drifted: declared %s via %s, observed %s via %s", intent.Experience, intent.Provider, observedRemoteDesktopExperience(serviceInfo), observedRemoteDesktopProvider(serviceInfo))
			return result
		}
	}

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
	isRunning := c.cachedServiceInfo != nil && c.cachedServiceInfo.Active
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
			"secrets-manager keyring inspect",
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
	output, err := c.executor.Output(ctx, "system"+"ctl", "is-active", "xrdp")
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
	if c.cachedServiceInfo == nil || !c.cachedServiceInfo.ProbeSucceeded {
		result.Status = checks.StatusWarning
		result.Message = "Unable to check RDP service"
		return result
	}

	if c.cachedServiceInfo.Active {
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
	return c.RecoveryActionsWithContext(context.TODO(), lastResult)
}

// RecoveryActionsWithContext discovers service state under the caller's
// context. The interface without a context remains for the generic check
// registry; its bounded TODO root is only a compatibility fallback for direct
// callers that have no request context.
func (c *RDPCheck) RecoveryActionsWithContext(ctx context.Context, lastResult *checks.Result) []checks.RecoveryAction {
	if !c.desiredStateAllowsRecovery(lastResult) {
		return nil
	}
	// Use cached service info if available, otherwise detect
	var serviceInfo RDPServiceInfo
	if c.cachedServiceInfo != nil {
		serviceInfo = *c.cachedServiceInfo
	} else {
		probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
		defer cancel()
		serviceInfo = c.detectRDPService(probeCtx)
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
		// GNOME Remote Desktop is a user session daemon; the native service backend
		// owns the platform-specific command.
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
