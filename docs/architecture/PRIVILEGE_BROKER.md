# Setup-managed privilege broker

`vrooli setup` is the one consent and elevation boundary for host actions that
must remain available after an owner no longer has terminal access. On a
supported elevated Linux setup it installs a root-owned `vrooli-privilege-
broker` system service. The service has no TCP listener: it accepts one JSON
request per connection on its Unix-domain socket.

## Trust boundary

The socket is owned by root and is reachable only by the uid selected when
setup ran (normally `SUDO_UID`). The broker obtains the connecting process uid
from `SO_PEERCRED` and rejects a peer which does not match that configured uid.
The Bridge API additionally requires its normal owner authentication before it
ever sends a request. Unix peer identity is the host boundary; scenario owner
authentication is the product-intent boundary. Neither is treated as a general
root grant.

Same-uid local processes are within the selected owner's local trust boundary.
They still cannot turn the broker into a shell: every request is validated by
the immutable capability registry before an adapter receives it. Future
dedicated service identities can narrow that boundary further without changing
the wire contract.

## Versioned contract

The v1 request is a single JSON object:

```json
{
  "version": "v1",
  "request_id": "opaque-correlation-id",
  "action": "bridge.ufw.inspect",
  "subject": {"scenario": "vrooli-bridge", "candidate_ip": "192.168.1.176", "port": 18767}
}
```

There is deliberately no executable, arguments, environment, path, or shell
field. The bridge family accepts these actions:

- `bridge.ufw.inspect`
- `bridge.ufw.allow`
- `bridge.ufw.verify`
- `bridge.ufw.revoke`

Bridge actions require scenario `vrooli-bridge`, a routable unicast IPv4 or IPv6 address,
and port `18767`. The policy constructs fixed UFW argv and tags the exact rule
with `vrooli-bridge-admission-v1`; it never accepts a CIDR, range, protocol, or
alternate port. Responses contain only typed state/evidence (`changed`,
`already_present`, `verified`, `unavailable`, or `failed`) and deterministic
error codes. Request ids are correlation values, not authentication or a
replay bypass; each mutation is independently idempotent.

## Lifecycle and failure modes

### Temporary process observation

`storage.temporary-activity.observe` is a read-only storage capability. It
accepts an empty subject and no other action-family subject. The target UID is
the broker's configured, authenticated Unix peer, never a request field.
The control-plane observer reads process descriptors, working directories,
mappings and known temporary storage environment references. It publishes only
temporary path metadata for that UID, the observed roots, a timestamp and a
completion flag. It never deletes files or returns arbitrary environment values.
Incomplete inspection refuses cleanup. The service uses host `/tmp` and
`/var/tmp` as read-only mounts rather than a private temporary namespace.

`broker.capabilities.inspect` reports the compiled action registry without
effects. Setup status consumes this live response, rather than equating an
open socket with availability of every action. Clients verify the root Unix
peer, bound request/response sizes and waiting time, close on cancellation, and
match response version, action and request identity.

The other registered action families are volume filesystem checks/repairs,
runtime-home ownership repair, managed log rotation, journald vacuum, unused
Docker data pruning and the fixed cloud deployment adapters. Each retains its
own closed subject validator.

### Setup convergence

Setup writes the service unit, executable, policy metadata and audit location
as root-owned artifacts, then enables and verifies the service. A re-run
replaces only managed artifacts and converges. Unsupported platforms and
non-elevated setup are successful setup states with an explicitly unavailable
capability and a concrete `vrooli setup --sudo-mode=ask` recovery instruction; no browser
password is collected or persisted.

The default non-elevated setup inspects existing capability state without
prompting. `--sudo-mode=ask` elevates only the fixed internal broker installation
child through the existing host requirement elevation wrapper; the rest of
setup retains the owner's session. An already-elevated setup also converges
the same broker. No separate permission installer is required.

The broker appends redacted audit events (action, caller uid, subject and
outcome; never credentials or request bodies). It supports exact managed-rule
revocation for rollback. Direct scenario `sudo`, forwarding a browser password,
setuid helpers, a generic sudoers command, and a TCP broker are rejected
because each would either widen commands beyond this immutable policy or make
the long-lived root boundary remotely reachable.

## Selected local-service admission

`local-service.admission.inspect`, `.ensure` and `.revoke` accept only a
`local_admission.binding` identifier. Project setup installs the binding policy
from the optional `local_service_admission` safeguard's validated operator
configuration. Each binding names a scenario's live UI listener, one private
IPv4 source, the host's private address and its home interface. Requests cannot
supply a port, address, interface, executable or path. The installed policy is
root-controlled at `/etc/vrooli/local-service-admission.json`; persistent
revocation state is under `/var/lib/vrooli/local-service-admission`.

The broker reads the canonical runtime registry without writes. It requires one
bound live UI claim, a running instance with a current heartbeat, a listening
socket owned by the setup-selected UID, matching scenario/variant/UI-port process
metadata, and the selected home interface/subnet. Missing or ambiguous evidence
does not admit a new listener. An IP scopes packets, not device identity;
application authentication/pairing is separate. Address/interface changes require
updated operator configuration and setup rather than broad network admission.

The Linux adapter uses fixed UFW arguments and exact source/destination/port/
interface/comment matches. It never enables UFW, opens a port range or treats an
unrelated broad rule as its managed rule. An inactive firewall is reported as
such. A verified new rule precedes removal of stale managed rules. Revocation
persists before removal and survives broker restarts. Setup removal of a binding
causes removal of its orphaned managed rules. A thirty-second broker convergence
cycle follows live listener changes; it does not control or poll the TV.
Changed/failed automatic operations carry binding identifiers in the broker
audit. This is not proof of end-to-end TV access or application authorization.

The systemd unit sets `TMPDIR=/run/vrooli` inside its existing writable runtime
directory. UFW uses Python temporary files when persisting rules. Keep this
temporary-storage contract even though `/tmp` and `/var/tmp` remain read-only:
an affected UFW version can report a successful live update after failing to
save configuration. Require exact readback rather than relying on command exit
status or successful live traffic.

`vrooli setup broker --sudo-mode=ask` is the focused project-setup operation for
installing/upgrading the broker and this saved policy. It reuses the same fixed
installation child as full setup. It excludes other host requirements, resource
or credential changes, onboarding launch and project-completion markers.
`vrooli setup broker --dry-run --json` inspects the running broker's capability
set without installing anything. A broker with an older capability registry can
be available while still needing this upgrade. Administrator authentication
belongs in the operator terminal; neither a scenario nor the broker collects a
password over an API.
