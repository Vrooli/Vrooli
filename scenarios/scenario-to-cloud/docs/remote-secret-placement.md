# Remote secret placement decision

Status: accepted for the current deployment workflow.

## Decision

`scenario-to-cloud` keeps the remote VPS bootstrap path for this plan. It
generates install-scoped values and sends them to the target work directory's
`secrets.json` before the selected resources start. The VPS is not assumed to
have a Vrooli installation, a running control plane, or a credential-store
adapter at the time this bootstrap runs.

The local control plane's credential authority remains the authority for local
scenario credentials. A remote bootstrap file is a separate remote-host
delivery surface, not a second local credential authority.

## Rejected option

The rejected option is to replace the bootstrap file with
`vrooli credentials provision` on the VPS. That would be stronger when the VPS
already runs Vrooli, but it cannot work during a fresh install before the
control plane and encrypted store exist. It would also make deployment depend
on a session, TPM, or operator passphrase on the remote host.

The chosen file path has the opposite tradeoff: the value is temporarily
recoverable from the remote file, so deployment must restrict its permissions,
avoid logging it, and remove or rotate it as part of the remote lifecycle.

## Revisit trigger

Reopen this decision when the deployment contract guarantees that every target
VPS boots a supported Vrooli control plane and can initialize its encrypted
credential store before resource bootstrap. The follow-up must then migrate
remote delivery atomically and add remote-host recovery and rotation proof;
removing only the file write would strand first-install deployments.
