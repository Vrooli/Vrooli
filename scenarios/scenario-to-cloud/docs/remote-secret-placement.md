# Remote secret placement decision

Status: accepted for the current deployment workflow.

## Decision

`scenario-to-cloud` provisions install-scoped values into the target host's
credential authority before the selected resources start. Each value crosses
SSH on standard input to `vrooli credentials provision`; no plaintext
`secrets.json` is created, read, or retained. The target must therefore have a
working Vrooli control plane and credential authority before this stage runs.

The local control plane's credential authority remains the authority for local
scenario credentials. A remote bootstrap file is a separate remote-host
delivery surface, not a second local credential authority.

## Rejected option

The rejected option was to retain a bootstrap `secrets.json` for first-install
compatibility. That file was a second plaintext credential authority and made
remote recovery and deletion ambiguous. Deployments now fail closed if the
remote control plane or credential authority is unavailable; the encrypted
file store can use an operator passphrase or a host-bound wrap without exposing
values in process arguments.

The chosen file path has the opposite tradeoff: the value is temporarily
recoverable from the remote file, so deployment must restrict its permissions,
avoid logging it, and remove or rotate it as part of the remote lifecycle.

## Revisit trigger

Reopen this decision only if a supported first-install path needs to bootstrap
the control plane itself. Any such path must preserve the same invariants:
typed credential-authority writes, standard-input delivery, no values in command
arguments or logs, and an explicit recovery/rotation proof on the remote host.
