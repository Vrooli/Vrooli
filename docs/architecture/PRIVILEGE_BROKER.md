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
field. The v1 registry accepts only these actions:

- `bridge.ufw.inspect`
- `bridge.ufw.allow`
- `bridge.ufw.verify`
- `bridge.ufw.revoke`

All require scenario `vrooli-bridge`, a routable unicast IPv4 or IPv6 address,
and port `18767`. The policy constructs fixed UFW argv and tags the exact rule
with `vrooli-bridge-admission-v1`; it never accepts a CIDR, range, protocol, or
alternate port. Responses contain only typed state/evidence (`changed`,
`already_present`, `verified`, `unavailable`, or `failed`) and deterministic
error codes. Request ids are correlation values, not authentication or a
replay bypass; each mutation is independently idempotent.

## Lifecycle and failure modes

Setup writes the service unit, executable, policy metadata and audit location
as root-owned artifacts, then enables and verifies the service. A re-run
replaces only managed artifacts and converges. Unsupported platforms and
non-elevated setup are successful setup states with an explicitly unavailable
capability and a concrete `sudo vrooli setup` recovery instruction; no browser
password is collected or persisted.

The broker appends redacted audit events (action, caller uid, subject and
outcome; never credentials or request bodies). It supports exact managed-rule
revocation for rollback. Direct scenario `sudo`, forwarding a browser password,
setuid helpers, a generic sudoers command, and a TCP broker are rejected
because each would either widen commands beyond this immutable policy or make
the long-lived root boundary remotely reachable.
