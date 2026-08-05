# Remote Desktop Check (infra-rdp)

Verifies that a remote client can actually connect **and authenticate** to this
host over RDP.

This check reports *serviceability*, not liveness. A running daemon that denies
every client is a failure, and this check says so.

## Overview

| Property | Value |
|----------|-------|
| Check ID | `infra-rdp` |
| Category | Infrastructure |
| Interval | 60 seconds |
| Platforms | Linux (GNOME Remote Desktop, xrdp), Windows (TermService) |

## Boundary with infra-display

These two checks are deliberately separate, and the split is by layer:

| Check | Owns | Answers |
|-------|------|---------|
| `infra-display` | Graphical-session dependency layer | Is there a working desktop session? |
| `infra-rdp` | RDP service layer | Can a remote client connect and authenticate? |

`infra-display` makes **no** statement about RDP. It has no RDP subcheck, no
`rdpPortListening` detail, and never claims RDP availability in its message.
Configuration, port, credentials, and denial signals all belong here.

`infra-rdp` reports the dependency through its own `sessionAvailable` detail
rather than restating display-manager health. When no graphical session exists
it reports "degraded because no graphical session exists" and leaves the
diagnosis of *why* to `infra-display`.

The checks stay separate because `infra-display` carries `Dangerous: true`
recovery actions that destroy the logged-in session, while every `infra-rdp`
action is safe. Merging them would expose a session-destroying action on a check
that trips over a missing password.

## What It Monitors

| Platform | Service | Detection |
|----------|---------|-----------|
| Linux | GNOME Remote Desktop | `grdctl status` reports enabled |
| Linux | xrdp | `xrdp.service` unit exists |
| Windows | TermService | `sc query TermService` |
| Other | none | Not checkable; reports OK |

For GNOME Remote Desktop the check reads five independent signals:

1. **Daemon liveness** — is `gnome-remote-desktop-daemon` running?
2. **Credential state** — can the daemon authenticate anyone? (three states, below)
3. **Client denials** — is the daemon actively turning clients away?
4. **Keyring loadability** — did gnome-keyring accept its keyring files at all?
5. **Host posture** — does this host match the configuration that predicts a
   credential fault?

## Credential state

The single most important field. A daemon can be configured, running, and
listening while denying every client, so liveness alone never yields OK.

| State | Meaning | Verdict |
|-------|---------|---------|
| `present` | Username and password are both set | OK |
| `empty` | No usable credentials; every client is denied | Critical |
| `unreadable` | State could not be determined from the calling context | Warning |

**Absence of credential output is never read as presence of credentials.**

This matters because `grdctl status` prints the credential lines only when it
can reach the session user's D-Bus bus. Without that environment it still prints
`Status: enabled` while omitting the credential lines entirely:

```
# with XDG_RUNTIME_DIR and DBUS_SESSION_BUS_ADDRESS set
Username: (empty)
Password: (empty)

# without them
Failed to read credentials: Cannot autolaunch D-Bus without X11 $DISPLAY.
RDP:
	Status: enabled
	Port: 3389
```

A check that greps for `(empty)` therefore produces a **false negative** from the
autoheal daemon's own context. The check runs `grdctl` with an explicit session
bus environment resolved from the active seat0 session, and classifies missing
lines as `unreadable`, which is non-OK.

The check never passes `--show-credentials` and never records, logs, or returns
a credential value. Only the classified state is stored.

## Credential models

Where credentials live determines whether automated repair is safe at all.

| Model | Storage | Automated repair |
|-------|---------|------------------|
| `system` | Root-owned store of `gnome-remote-desktop.service` | **Yes** — `repair-credentials` |
| `user-session` | The user's GNOME login keyring | **No** — `raise-incident` only |

On the **user-session** model autoheal deliberately refuses to repair. Unlocking
the login keyring requires a secret autoheal must not hold, and writing a fresh
RDP password would mean autoheal minting remote-access credentials on its own
initiative. That is a real expansion of blast radius and this check declines it.
There is no `set-credentials` action, by design.

On the **system** model the remedy is deterministic and non-interactive: restart
the system unit so the daemon re-reads its own credential store. No credential
value passes through autoheal at any point.

## Client denials

The check counts refusals in the daemon's own journal over a fixed 15-minute
window, which distinguishes a latent misconfiguration from an in-progress
lockout.

```
Aug 01 14:27:11 host gnome-remote-de[3623]: [RDP] Credentials are not set, denying client
```

| Detail | Meaning |
|--------|---------|
| `recentDenials` | Every refusal in the window |
| `recentCredentialDenials` | Refusals specifically naming missing credentials |
| `denialWindowMinutes` | Window size (15) |
| `journalReadable` | Whether the journal could be read at all |

**A zero denial count is never evidence of health** — it usually means no client
tried. A positive count raises severity to critical; a zero count never lowers
one. A failed journal read likewise never rescues a non-OK verdict.

## Keyring loadability

A locked keyring and an unloadable one produce the same symptom and need
opposite remedies, so the check asks the keyring daemon directly rather than
inferring from posture.

| Detail | Meaning |
|--------|---------|
| `keyringFileRejected` | gnome-keyring parsed a keyring file and discarded it |
| `keyringFilePath` | The file it named |
| `keyringUnlockFailureLogged` | `gkr-pam` reported it could not unlock the login keyring |
| `keyringJournalReadable` | Whether the evidence could be read at all |
| `keyringCorrupt` | A credential fault **and** a rejected file — the root cause |
| `keyringRepairPending` | The file was rejected at boot but parses now: repaired, awaiting a login |

`keyringRepairPending` is what lets the check leave this state. The rejection is
boot-scoped evidence that stays in the journal until reboot, so a check reading
only that signal would keep reporting a malformed file after it had been fixed,
and an operator following its advice would re-run the repair indefinitely. When
it is set, the remedies drop the repair step and name only the re-login.

The loadability answer comes from `vrooli credentials keyring inspect --format
json` rather than from a second parser inside autoheal, because the file format
and the encoding written back belong to `securestore`. Its JSON envelope is
pinned by a test on the producing side; a bare-array parser here once survived a
green suite because the test mocked the assumed shape rather than the emitted
one.

The window is the current boot, not the denial probe's 15 minutes, because the
daemon reads its keyring files once at session start:

```
gnome-keyring-daemon[2953]: keyring was in an invalid or unrecognized format:
    /home/user/.local/share/keyrings/login.keyring
gdm-autologin][2912]: gkr-pam: couldn't unlock the login keyring.
```

**A rejected file outranks the autologin posture.** The posture is an inference
from three booleans that are also true on hosts where RDP works; the rejection
is the daemon's own statement about what it did. When a file was rejected,
`lockedKeyringPosture` is reported as `false` — the posture may still match, but
"locked" is the wrong diagnosis for a file that never loaded, and acting on it
costs root and a desktop session for nothing.

`keyringUnlockFailureLogged` is corroborating only. It is emitted for a genuinely
locked keyring too, so it never identifies the fault alone.

**A failed journal read never asserts a healthy keyring**, on the same principle
that a zero denial count never lowers a verdict.

### Why a keyring file becomes unloadable

GNOME Keyring stores a passwordless keyring in GKeyFile's textual format, where
a value occupies exactly one line. A value carrying a real newline — a PEM
private key is the usual way — makes the file unparseable, and the daemon
rejects the **entire keyring** rather than the one entry. Every secret in it
goes dark at the next login, including ones written years earlier by unrelated
applications.

Vrooli's credential store now encodes any value that is not single-line safe
before it reaches a backend, so it cannot create this state again. See
`internal/resources/securestore/values.go`.

## Host posture

These fields predict the failure class before any client attempts a connection.

| Detail | Meaning |
|--------|---------|
| `autoLoginUser` | The GDM autologin user, if configured |
| `loginKeyringCollectionPresent` | Whether the login keyring is unlocked and registered on the session bus |
| `isUserSession` | Whether the daemon runs as a user-session daemon |
| `lockedKeyringPosture` | All three of the above match the known-bad shape |
| `sessionAvailable` | Whether a graphical session exists to share |

The known-bad posture is GDM autologin **plus** a user-session daemon **plus** an
absent login keyring collection. The `gdm-autologin` PAM stack authenticates
through `pam_permit` with no password, so `pam_gnome_keyring` has nothing to
unlock the keyring with. Anything stored in that keyring — including the RDP
credentials — is unreadable for the life of the session.

**Posture alone never changes the status.** An operator who unlocks the keyring
by hand after boot still matches the posture while RDP works perfectly. The
posture only annotates the root cause when the credential state is also `empty`
or `unreadable`.

## Status meanings

| Status | Condition |
|--------|-----------|
| **OK** | No RDP service installed, or the daemon is running with credentials present and no denials |
| **Warning** | Configured but not running, or credential state is `unreadable` |
| **Critical** | Credentials are `empty`, clients are being actively denied, or no graphical session exists |

Hosts with no RDP service installed report OK. The credential probe runs only
after RDP detection succeeds, so headless hosts are never alarmed by it.

## Declared desired state

When the root host contract declares `remote_desktop_access`, `infra-rdp` reads
its resolved configuration without changing it. An opted-in requirement is
compared with the observed provider and experience:

| Verdict | Meaning | Recovery behavior |
|---|---|---|
| `unmanaged` | The operator has not opted into the safeguard, or its declaration cannot be read safely. | Informational; no recovery action. |
| `matching` | The observed remote-desktop experience satisfies the declared experience and provider. | Existing health and credential checks continue. |
| `drifted` | The observed experience/provider differs from the declared intent. | Warning only; autoheal does not override the operator's display policy. |

`observe-only` is an explicit no-op declaration and reports the observed state
without requesting recovery. This comparison does not authorize autoheal to
mint, read, or repair user-session credentials. A host with no RDP service
remains informational, preserving the check's optional-capability behavior.

## Recovery actions

| Action | Available when | Dangerous |
|--------|----------------|-----------|
| `start` | Daemon is not running | No |
| `restart` | Daemon is not running | No |
| `repair-credentials` | `credentialModel=system` **and** credential fault | No |
| `repair-keyring` | `keyringCorrupt=true` | No |
| `raise-incident` | `credentialModel=user-session`, credential fault, **and not** `keyringCorrupt` | No |
| `status`, `diagnose`, `logs`, `open-settings` | Always | No |

`repair-keyring` is the one automated repair autoheal performs on the
`user-session` model. The standing refusal below exists so autoheal never mints
remote-access credentials it would need to hold a secret to create; rewriting a
malformed entry Vrooli itself wrote creates no credential, reads no value, and
needs no elevated privilege, so it falls outside the reason for the refusal
rather than being an exception to it.

It delegates to `vrooli credentials keyring repair` rather than reimplementing
the rewrite. A second implementation would be a copy that drifts — which is
exactly how the file this check protects would get corrupted again, by the
supervisor meant to prevent it.

`raise-incident` is withdrawn when a repair is available, so the operator is not
offered "report it" beside "fix it".

`restart` is offered only when the daemon is down. A restart cannot repair a
credential that is unreadable because a keyring is locked, so it is never
presented as the remedy for a credential fault.

`raise-incident` is non-mutating. It reports the fault and its operator remedy
on demand; the durable incident record is raised from this check's non-OK result
by the incident pipeline, which also resolves it automatically once the check
returns OK again.

## Operator remedies for an unloadable keyring

All of these are user-scoped; none needs root.

```bash
# Which entries did gnome-keyring reject?
secrets-manager keyring inspect

# Rewrite the malformed Vrooli-owned entries (backs up first, declines
# entries other applications own, and sweeps abandoned *.keyring.temp-* files)
secrets-manager keyring repair
```

Then log out and back in, or reboot, so `gnome-keyring-daemon` reloads the file.
A repair alone does not fix the running session: the daemon reads its keyring
files at session start, and until it does, reads may succeed while writes hang
on an unlock prompt nobody can answer.

Finally, confirm the RDP password survived. Repair restores the *file*, not a
value that was already empty inside it:

```bash
grdctl status                                          # Password: (empty)?
grdctl rdp set-credentials <username> <password>       # then set it again
```

## Operator remedies for a locked keyring

These apply only when `keyringCorrupt` is false. Autoheal reports them rather
than performing them:

1. Disable GDM autologin in the system GDM configuration (`$GDM_CONFIG`, commonly `gdm3/custom.conf`) and log in interactively
   once, so `pam_gnome_keyring` unlocks the login keyring with the account
   password.
2. Or migrate the host to the system-level `gnome-remote-desktop.service`
   credential store, where credentials do not depend on a user keyring — and
   where autoheal *can* repair this fault automatically.

## Troubleshooting

```bash
# Full check output including every detail field
vrooli-autoheal check get infra-rdp --json

# Credential state as the daemon sees it (never use --show-credentials)
XDG_RUNTIME_DIR=/run/user/$(id -u) \
  DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$(id -u)/bus \
  grdctl status

# Is the login keyring unlocked and registered?
gdbus call --session --dest org.freedesktop.secrets \
  --object-path /org/freedesktop/secrets \
  --method org.freedesktop.DBus.Properties.Get \
  org.freedesktop.Secret.Service Collections

# Recent denials
journalctl --user-unit gnome-remote-desktop --since "15 minutes ago"
```

Note that `gdbus introspect` on the collection path is **not** a usable
existence test: it returns an empty node rather than an error when the object
does not exist. Read the `Collections` property instead.

## xrdp and Windows

For hosts running xrdp or Windows TermService the check remains a service-status
check; the credential, denial, and posture signals above are specific to GNOME
Remote Desktop.

```bash
# Linux (xrdp)
sudo systemctl status xrdp
sudo journalctl -u xrdp --since "10 minutes ago"
sudo ss -tlnp | grep 3389
```

```powershell
# Windows
Get-Service TermService
Get-NetFirewallRule -DisplayGroup "Remote Desktop"
```

## Security notes

- Autoheal never creates, writes, reads, or logs a remote-access credential value.
- RDP traffic should be encrypted (TLS for GNOME RDP and xrdp, NLA for Windows).
- Consider a VPN or Cloudflare Tunnel instead of exposing port 3389.

## Related checks

- **infra-display** — owns the graphical-session layer this check depends on.
- **infra-network** — remote access requires network connectivity.

---

*Back to [Check Catalog](../check-catalog.md)*
