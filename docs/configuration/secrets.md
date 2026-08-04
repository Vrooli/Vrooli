# Configuring credentials

Vrooli declares credentials only in resource and scenario manifests. A
descriptor identifies a credential without selecting a storage backend:

```json
"credentials": {"descriptors": [{
  "logical_id": "vrooli/openrouter",
  "field": "api-key",
  "env": "OPENROUTER_API_KEY",
  "label": "OpenRouter API Key",
  "required": true,
  "obtain_url": "https://openrouter.ai/keys"
}]}
```

`logical_id` and `field` are stable backend-neutral names. `env` is only the
process-scoped injection name; it is not durable storage.

Resources and scenarios use the same block and the same descriptor shape, so a
scenario-owned credential is the same thing to the store, to `doctor`, and to a
recovery bundle as a resource-owned one. A credential nothing declares is a
credential no backup captures.

### `env` is optional, and declaring it is a decision

Declare `env` **only when the consumer is a process Vrooli does not author** — a
database container, a third-party CLI — which can receive a value no other way.
That is what the field exists for.

Code Vrooli writes resolves the value itself instead, through the
credential-authority binding for its language (`packages/credential-authority-go`
for Go). Omitting `env` keeps the value out of the process environment, where it
would be readable at `/proc/<pid>/environ` and inherited by every subprocess the
consumer spawns — and scenarios shell out constantly.

A descriptor with no `env` is injected nowhere but is otherwise fully declared:
it is diagnosed, listed, and captured by `recovery export --all`.
`scenarios/tunnel-manager` is the reference: it declares three Cloudflare
credentials with no `env` and re-resolves them per operation, so a rotated token
takes effect without restarting the scenario. The local and desktop
authority is a native OS secure store; on a host that has none — headless
servers, CI runners, a Raspberry Pi — it is the encrypted file store described
under [Platform backends](#platform-backends). Vault may be a scoped mirror or a
capability-specific service, but it is not the ordinary credential authority.
See also the [repository contract](../repo-contract.md); it names the same
credential authority and storage boundary.

## The degradation contract

**A missing or unreadable manifest-declared credential never blocks a scenario
process from starting.** A scenario starts whenever its processes and ports can
start. Credential state changes what a resource can *do*, never whether the
control plane runs.

This rule covers application credentials declared by a resource or scenario
manifest. Resource-native bootstrap material has a separate fail-closed
requirement: for example, Vault root and unseal material must be persisted in
secure storage before a managed Vault instance is published as usable. If that
storage is unavailable, the Vault resource remains unavailable rather than
starting with recovery material that cannot be safely recovered. That is a
resource bootstrap failure, not a fallback to plaintext or a failure to resolve
an application credential.

Concretely:

- `vrooli scenario start <name>` succeeds when a manifest-declared credential is
  unset, when the OS secure store is unreachable, and when the host has no
  secure store at all, provided the scenario's own required processes and
  resource bootstraps can start.
- The affected resource reports `unhealthy` with a named remediation. Check it
  with `vrooli resource status <name>`.
- The credential's variable is **omitted** from the process environment rather
  than injected as an empty string, so a consumer checking presence never sees a
  configured-but-blank credential. When the manifest declares another source for
  the same variable — a bootstrap default in `runtime.env`, for example — that
  value stands in and the resolver emits a warning naming the stand-in.
- `required: true` keeps its meaning: the resource cannot do its job without the
  value. For an application credential, the precise consequence is an
  unhealthy resource, not a failed start. Resource-native bootstrap material
  follows the fail-closed rule described above.

Write paths are the deliberate exception and stay fail-closed. Recovery export
and restore refuse to run when the provider is unavailable, because a bundle
written from a store nobody could read would look like a backup and not be one.

## Start then configure

Provisioning a credential after the scenario is already running reaches the
working state with **no control-plane restart**. Credential values are read on
demand and are never cached:

```bash
vrooli scenario start web-console                 # runs; openrouter degraded
printf '%s' "$OPERATOR_VALUE" | vrooli credentials provision \
  --identity vrooli/openrouter --field api-key
vrooli resource status openrouter                 # now healthy
```

## The three failure kinds

Exactly three conditions exist, they are separately detectable, and each has a
different operator action. Never treat one as another.

| Kind | Meaning | What the operator does |
|---|---|---|
| Provider unavailable | The host secure store exists but cannot be reached right now — a locked keychain, a stopped service, a session Vrooli could not repair on its own | `vrooli credentials doctor` names the cause and the fix. |
| Provider absent | This host has no native secure store | Initialize the encrypted file store, or install a native backend. `vrooli credentials doctor` names the command. |
| Unconfigured | The store works and holds no value for this identity and field | `vrooli credentials provision --identity <id> --field <field>` |

The distinction is not cosmetic. Before it existed, an unreachable session bus
was reported as "credential is not configured", so an operator with a perfectly
provisioned key was told to provision it again.

## Diagnosing a host

`vrooli credentials doctor` explains this host's backend, then every declared
credential on it:

```
$ vrooli credentials doctor
Credential provider
  Platform:  linux
  Adapter:   libsecret
  Storage:   unencrypted-keyring — values are readable with a text editor; file mode is the only protection
  Condition: unavailable
  Writable:  not-checked
  Why:       operating-system secure storage is unavailable: read secure resource material: exit status 1: secret-tool: Could not connect: Permission denied
  Fix:       XDG_RUNTIME_DIR=/run/user/0 is owned by uid 0 but this process runs as uid 1000; export XDG_RUNTIME_DIR=/run/user/1000

Declared credentials (27)
  RESOURCE    VARIABLE            IDENTITY           FIELD    REQUIRED  STATE
  openrouter  OPENROUTER_API_KEY  vrooli/openrouter  api-key  yes       provider_unavailable
  ...

Unresolved (27) — the scenario still starts; these application resources stay
degraded until fixed:
  openrouter → OPENROUTER_API_KEY
      the credential store is unreachable; run `vrooli credentials doctor` for the host diagnosis
```

On Linux the diagnosis names a uid/session mismatch or a headless host with no
Secret Service, rather than surfacing a bare `Permission denied`. Both commands
accept `--format json`.

`Fix` always names the condition that is blocking **now**. The remedy travels
with the error from the layer that detected it, so a half-loaded Secret Service
collection reports "log out and back in so the keyring daemon reloads the
keyring file" rather than something inferred from a different layer's symptoms.
`Repaired` is a separate field precisely because a correction Vrooli already
made is not an action the operator still owes.

### `doctor` is read-only by default

`Writable: not-checked` is the normal answer. Proving a store can be written
means storing, reading back, and deleting a throwaway value in the operator's
real credential store — and `doctor` is the command someone runs when that store
is already misbehaving, where a write is exactly what raises the unlock prompt
nobody is there to answer.

Pass `--check-writes` when the question really is "can I provision right now":

```bash
vrooli credentials doctor --check-writes
```

Onboarding's readiness step passes it, because its next action is a write.

### Automatic session repair

On Linux the Secret Service is reached over a session bus, so libsecret depends
on `XDG_RUNTIME_DIR` and `DBUS_SESSION_BUS_ADDRESS`. A non-login user switch —
`su someone` from a root shell — keeps root's values, and a perfectly healthy
keyring then reports a bare `Could not connect: Permission denied`.

Vrooli corrects this itself rather than asking an operator to know about D-Bus.
When the variables name a session belonging to another user, or name nothing at
all, the credential subprocess is given this user's own session instead, and
`doctor` reports what it did:

```
Condition: available
Repaired:  XDG_RUNTIME_DIR=/run/user/0 is owned by uid 0 but this process runs
           as uid 1000; used this user's own session at /run/user/1000 instead
```

The repair is deliberately narrow, because it decides where credentials are read
from:

- The only destination ever considered is `/run/user/<this process's uid>`. No
  input can aim it at another user's session.
- That directory must exist, be a directory, be owned by this uid, and grant
  nothing to group or other; its `bus` must be a socket owned by this uid. Paths
  are examined without following symlinks.
- A setting that already resolves correctly is never touched, including a custom
  runtime directory the operator owns.
- If any check fails there is no repair, and the condition is reported as before.

The correction applies to the credential subprocess only. Vrooli's own
environment and every scenario's environment are left alone, so a broken login
stays visible to the other session-scoped tools it affects — which is also why
`doctor` discloses the repair instead of performing it silently.

`vrooli credentials list` prints the same declaration table without the provider
section. Neither command can print a stored value.

`vrooli credentials status --identity <id> --field <field>` reports one
credential and always carries the provider state alongside it, so
`unconfigured` can never be misread while the store is down:

```
$ vrooli credentials status --identity vrooli/openrouter --field api-key
Credential vrooli/openrouter/api-key: unconfigured (provider libsecret, available)
```

## Provisioning

Provision through onboarding or standard input, never a command argument:

```bash
printf '%s' "$OPERATOR_VALUE" | vrooli credentials provision \
  --identity vrooli/openrouter --field api-key
```

Removing a credential is the deprovision half of the same surface, and is
explicit because it is unrecoverable without a recovery bundle:

```bash
vrooli credentials delete --identity vrooli/openrouter --field api-key --yes
```

Delete refuses rather than reporting success when the provider is unreachable —
otherwise an operator could believe a leaked key was revoked while it was still
in the store. It is idempotent, and says plainly when there was nothing to
remove.

Runtime code resolves credentials through the control-plane authority into the
target process only. It must not read YAML inventories, resource-private
credential files, generic environment fallbacks, or Vault content directly.

## Platform backends

Every supported platform has a native adapter, and every adapter passes one
shared conformance suite. See
[the resource deployment contract](../resources/deployment-contract.md) for the
adapter table.

Every native adapter is a desktop-session facility. A headless host — a server,
a CI runner, a Raspberry Pi — has none of them, so the **encrypted file store**
is the backend there. It seals each value with AES-256-GCM under a data key that
is never written unwrapped; the key is wrapped by the host TPM through
`systemd-creds`, by an operator passphrase, or by both.

### Values are single-line at the seam

A second storage rule: **a value that leaves `securestore` for a backend is
single-line.** Any value containing a newline, carriage return, or NUL is
base64-encoded behind a versioned marker on the way in and decoded on the way
out, so callers store and read whatever bytes they hold. Single-line values pass
through untouched, which keeps every credential written before this rule
readable without a migration.

This is a property of the seam rather than of the caller because the failure it
prevents is silent at write time and only surfaces at the next login. GNOME
Keyring stores a passwordless keyring in GKeyFile's textual format, one line per
value. Handed a PEM private key — multi-line by construction — it wrote the
newlines verbatim and produced a file it could no longer parse. It then rejected
the **entire keyring**, not the offending entry, taking every unrelated secret
in it down with it, including credentials other applications had stored years
earlier.

A host already in that state is repaired with `vrooli credentials keyring
inspect` and `vrooli credentials keyring repair`, which need no elevated
privileges. See the [infra-rdp check](../../scenarios/vrooli-autoheal/docs/reference/checks/infra-rdp.md)
for how autoheal detects it.

The credential seam does not put values in process arguments and does not use a
plaintext fallback. The encrypted file store satisfies the stronger rule that
the stored value is not recoverable from its file alone: the data key is held by
a TPM, host key, or operator passphrase. The native GNOME Secret Service
adapter is different. A passwordless `[keyring]` file is readable with a text
editor, so its confidentiality rests on file permissions. `credentials doctor`
reports `unencrypted-keyring` and this caveat; this changes the description,
not the Linux default or the existing data.

`vrooli credentials keyring inspect` also reports whether the stale-daemon
comparison ran. If the daemon start time is unavailable, it says that the
check did not run and makes no stale-daemon claim.

An environment variable is an injection target, not durable credential storage.
The authority resolves declared values before a process starts.

The legacy `packages/api-core/secrets` package has been **deleted**, along with
scenario-to-cloud's `/local-secrets` API — the read/write surface that
maintained `~/.vrooli/secrets.json` and recreated it on every deploy. There is
no plaintext credential reader left to bound, and no compatibility shim: a new
reference to that package would not compile, which is a stronger guarantee than
a documented reference count.

Cloud deployment provisions into the **target host's own** credential authority
rather than shipping a file. Each value crosses the SSH connection on standard
input, one call per credential, so no secret enters a command string — a command
string is argv for the local `ssh` process and the argument to the remote shell,
which made every value readable in both process listings.

Selection is a chain, and the rule that makes it safe is narrow:

| Native adapter reports | What happens |
|---|---|
| available | The native store is the authority. Nothing changes. |
| `ErrUnavailable` — exists, unreachable now | Stay degraded. The encrypted store is **never** opened. |
| `ErrAbsent` — this host has no native store | The encrypted store is the authority. |

Falling back on `ErrUnavailable` would split credentials across two backends
according to session health: a value written while the keyring was up would
vanish when it went down. A degraded resource is an honest state an operator can
fix; a silent second store is not.

`vrooli credentials doctor` always names the active backend and, for the
encrypted store, the active key wrap — because the wraps are not equally strong.
On a host with a TPM the host-bound wrap resists possession of the disk. On a
host without one, `systemd-creds` falls back to a root-owned host key on the
same disk, so possession of the disk (or the Pi's SD card) is enough. Doctor
reports which of the two is in use rather than presenting one uniform level of
protection.

### The host-bound wrap needs TPM group access

Having a TPM is not the same as being able to use one. Distributions ship
`/dev/tpmrm0` owned by a `tss` group, and the `systemd-creds` host-key fallback
reads a root-only secret. A Vrooli process running as an ordinary user is
therefore refused by **both** paths on a stock host, and the passphrase wrap
becomes the only one that works — which means a passphrase after every reboot,
not the unattended boot the host-bound wrap exists to provide.

**Setup owns this repair.** The `tpm_credential_access` host safeguard adds the
invoking operator account to the group owning the TPM device, during the one
privileged pass `vrooli setup` already runs. It is not something an operator
repairs by hand:

```bash
vrooli setup --include-optional
```

It is optional because a host that is happy typing a passphrase does not need
it, and it auto-skips where it cannot help — no TPM device, or a device whose
mode grants its group nothing. `vrooli setup --dry-run` names the exact account
and group it would change before changing anything.

Group membership is fixed at login, so the grant takes effect in a **new login
session**. After that:

```bash
vrooli credentials store rewrap
```

`rewrap` adds the host-bound wrap to an existing store. The data key does not
change, so no stored value is re-encrypted and nothing can be lost — this is
also the upgrade path for a host that gains a TPM later.

`store init`, `store status`, and `doctor` all name the blocking group while it
is unresolved, and point at `vrooli setup` rather than at a raw privileged
command. Setup is the only place this project changes host state, so a fix
issued anywhere else would leave the host in a condition no config describes.

One consequence worth stating: membership in that group permits TPM use for any
purpose, which is why the safeguard grants it to the account running Vrooli and
nothing broader.

## Recovery

Recovery export and restore are explicit encrypted operations protected by an
operator-held passphrase, and both stay fail-closed when the provider is
unavailable. Retired `config/secrets.yaml` files are migration inputs only and
never a runtime contract.

Create a recovery bundle by naming the credential metadata and passing the
passphrase through standard input. The output path is created once with owner-
only permissions; neither command prints a value or passphrase:

```bash
printf '%s' "$RECOVERY_PASSPHRASE" | vrooli credentials recovery export \
  --entry vrooli/openrouter:api-key --output ./vrooli-recovery.bundle

# Back up every configured credential declared by a resource manifest.
printf '%s' "$RECOVERY_PASSPHRASE" | vrooli credentials recovery export \
  --all --output ./vrooli-recovery-all.bundle

printf '%s' "$RECOVERY_PASSPHRASE" | vrooli credentials recovery restore \
  --input ./vrooli-recovery.bundle

# Replace only the passphrase wrap. Read current and new passphrases on
# separate stdin lines; stored values are not re-encrypted.
printf 'CURRENT_PASSPHRASE\nNEW_PASSPHRASE\n' | vrooli credentials store change-passphrase
```
