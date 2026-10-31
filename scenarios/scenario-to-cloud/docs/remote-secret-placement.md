# Remote secret placement decision

Status: accepted for the current deployment workflow; extended by the
credential lifecycle (see `reference/credential-lifecycle.md`).

## Decision

`scenario-to-cloud` provisions install-scoped values into the target host's
credential authority before the selected resources start. Each value crosses
the transport on standard input (SSH: the ingest payload piped to
`vrooli cloud-target credential ingest`) or inside the Bridge sealed grant
channel (bridge transport); no plaintext `secrets.json` is created, read, or
retained anywhere. The target must therefore have a working Vrooli control
plane and credential authority before this stage runs, and a locked or
unavailable store fails the deploy closed (`credential_store_locked`) rather
than falling back to a file.

The local control plane's credential authority remains the authority for local
scenario credentials. The target host's authority is the authority for the
deployment's credentials. scenario-to-cloud is a client of both and an authority
for neither: it records bindings, versions, consumer acknowledgements and
lifecycle receipts, never values.

## Rejected option

The rejected option was to retain a bootstrap `secrets.json` for first-install
compatibility. That file was a second plaintext credential authority and made
remote recovery and deletion ambiguous: a value could be "deleted" from the
authority and still be readable from the file, and a recovery bundle could not
say which copy was current. Deployments now fail closed if the remote control
plane or credential authority is unavailable; the encrypted file store can use
an operator passphrase or a host-bound wrap without exposing values in process
arguments.

## Lifecycle after placement

Placement is the first version of a binding. Rotation, revocation and recovery
are versioned operations over the same authority:

- rotation prepares the provider, distributes the new version, verifies every
  authorized consumer's acknowledgement and only then retires the predecessor;
- revocation purges the target store and denies new distribution, and states
  on its receipt that it cannot prove a compromised host forgot a value it had
  already read;
- recovery re-provisions a replacement host from the encrypted recovery bundle
  through `vrooli credentials recovery verify|restore` with the passphrase on
  standard input.

## Revisit trigger

Reopen this decision only if a supported first-install path needs to bootstrap
the control plane itself. Any such path must preserve the same invariants:
typed credential-authority writes, standard-input or sealed-channel delivery,
no values in command arguments, logs, receipts or responses, and an explicit
recovery/rotation proof on the remote host.
