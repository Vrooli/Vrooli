# Runbook: rotate, revoke or recover credentials

Outcome: every consumer of a binding holds and has acknowledged the new
version, the predecessor is revoked everywhere it was distributed, and no
value ever appeared on a command line, in a plan, a receipt or a log.
Contract: `docs/reference/credential-lifecycle.md`.

## 0. Read the bindings first

```bash
scenario-to-cloud credential list "$DEPLOYMENT_ID" --json
```

Each binding shows its lifecycle class, active version, consumers and their
acknowledgement standing. One open operation per binding: a rotate while
another rotation, break-glass or deploy-time materialisation is open is
`credential_rotation_conflict` (exit 2). Finish or resume the open one first
(`rotation-get`).

## 1. Rotate one binding

Provider-generated value (the provider mints the new version):

```bash
scenario-to-cloud credential rotate "$DEPLOYMENT_ID" --binding "$BINDING_ID" --request-key "rotate-$BINDING_ID-$DATE"
```

Operator-supplied value (never on argv; read from standard input once):

```bash
printf '%s' "$NEW_VALUE" | scenario-to-cloud credential rotate "$DEPLOYMENT_ID" --binding "$BINDING_ID" --value-stdin --request-key "rotate-$BINDING_ID-$DATE"
```

The command returns the rotation id and standing. States you will see
(`credential-lifecycle.md` §"Rotation state machine"):

| Standing | Meaning | Do |
|---|---|---|
| `complete` (exit 0) | new version active, predecessor revoked | step 3 |
| `consumers_updated` with `unreached[]` (exit 3) | at least one consumer has not acknowledged; the active version is **not** advanced | bring the consumer back (its node, its enrollment), then `rotation-resume` |
| `verified` with `resume_after` (exit 3) | overlap window running | wait until `resume_after`, then `rotation-resume` |
| `pending_operator_input` (exit 3) | the provider cannot revoke the predecessor by API | revoke it at the provider, then `rotation-resume --operator-confirmed` |
| `failed` (exit 1) | provider rejected, or distribution/verification failed and recovery retired what was distributed | the predecessor remains active; read `rotation-get` receipts, fix the cause, rotate again (the failed attempt's version number is burned) |

## 2. Resume or inspect a parked rotation

```bash
scenario-to-cloud credential rotation-get "$DEPLOYMENT_ID" --rotation "$ROTATION_ID" --json
scenario-to-cloud credential rotation-resume "$DEPLOYMENT_ID" --rotation "$ROTATION_ID"
scenario-to-cloud credential rotation-resume "$DEPLOYMENT_ID" --rotation "$ROTATION_ID" --operator-confirmed
```

`--operator-confirmed` is a statement you make; the receipt records it under
your identity.

## 3. Verify

```bash
scenario-to-cloud credential list "$DEPLOYMENT_ID" --json
scenario-to-cloud deployment health "$DEPLOYMENT_ID" --json
```

Required: the binding's active version is the new one; every consumer
`acknowledged`; health `HEALTHY`/`CURRENT` (a consumer that restarted on the
new value is proven by the observation, not by the rotation's success).

## 4. Revoke (compromise, off-boarding)

```bash
scenario-to-cloud credential revoke "$DEPLOYMENT_ID" --binding "$BINDING_ID" --request-key "revoke-$BINDING_ID-$DATE"
```

Revocation deletes the active version from every target authority it was
distributed to and marks the version so it can never be re-ingested. A
target that is unreachable leaves the operation `revocation_incomplete`
with the unreached node list retained; it is not complete until a purge
receipt closes it (`rotation-get` shows the standing). A revoked binding with
consumers still declared makes the next plan `needs_input`: rotate (step 1)
to supply a replacement.

## 5. Recover the credential store onto a replacement host

Used together with `restore.md` when the host is lost. The sealed recovery
bundle reference and its passphrase come from your recovery custody, never
from the target being restored:

```bash
printf '%s' "$RECOVERY_PASSPHRASE" | scenario-to-cloud credential recover "$DEPLOYMENT_ID" --bundle-ref "$BUNDLE_REF" --passphrase-stdin --request-key "recover-$DEPLOYMENT_ID-$DATE"
```

A wrong passphrase or an unavailable bundle is a typed refusal
(`recovery_key_unavailable`, `credential_store_locked`); nothing partial is
written. Verify with step 3 afterwards.

## Non-binding secrets

Workspace and scenario secrets that are not deployment credential bindings
(the values a bundle ships from the secrets owner) are managed with the
`secrets` group; the same rule applies, values are read from `--value` only
for non-production convenience and from `--generate` otherwise:

```bash
scenario-to-cloud secrets verify "$SECRET_KEY" --deployment "$DEPLOYMENT_ID"
scenario-to-cloud secrets set "$SECRET_KEY" --generate 32 --deployment "$DEPLOYMENT_ID" --restart
```

## Never

- paste a value into a shell command, a manifest, a plan or a ticket;
- "fix" an unreached consumer by re-running the rotation: the version is
  burned and a new one would be minted; resume instead;
- treat `complete` as proof the workload uses the value: the observation is.
