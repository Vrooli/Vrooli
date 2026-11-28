# Headless credential-unlock socket

This package lets a local Vrooli process request an in-memory store passphrase
from its node agent. The socket is under the runtime state directory, is
created in a `0700` directory with `0600` socket permissions, and the client
refuses a socket or parent directory it cannot verify as owned by its user.
The server replies only when the peer UID matches its own.

Before the control plane has delivered a value, the server returns an explicit
not-delivered response and the client reports `ErrUnavailable` within the
bounded request timeout. A caller may retry after delivery; no value is
persisted by this transport. The provider callback must return an owned byte
copy. The server zeroes that copy after constructing the response, while the
agent's retained copy is unchanged. Logs and error responses contain no
passphrase.

The authenticated Unix-socket flow is supported on Linux and macOS, where peer
UID credentials are available. Windows deliberately skips this socket path and
uses its DPAPI-backed store. This package's tests use only an in-process local
Unix socket and synthetic bytes: they prove unavailable→delivered recovery and
buffer ownership, not Bridge synchronization, real node startup, or a served
credential-store open.
