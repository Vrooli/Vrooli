# Runbook

## Readiness is not green

Read the item's remediation text — it is specific to the cause, not generic.

| Symptom | Action |
|---|---|
| A required credential is `unconfigured` | Provision it: `printf '%s' "$V" \| vrooli-onboarding credentials provision --identity <id> --field <f>` |
| A credential is `unsupported` | Run `vrooli credentials doctor`. It distinguishes an unset value, an unreachable store, and a host with no backend |
| A host tool is `missing` | Opt in and re-apply, or install it and recheck |
| A safeguard is `missing` | Re-apply with the required privilege. Do not hand-write the verification file |
| A resource is unreachable | Start it, or deselect the capability that needs it |
| Any item is `unsupported` | The requirement is not declared for this platform. Change target or deselect the capability. Do not create a fallback file |

After any fix, use **Recheck** rather than reloading. The probe pass is cheap and
the reload loses your position.

## Generic operator capabilities

The Credentials step is descriptor-driven. Setup discovers capabilities without
asking for values; onboarding then presents candidate metadata, typed inputs,
the exact preview mutations, confirmation, and metadata-only evidence:

```bash
vrooli capability status --json
```

Select a sink only from the provider's candidates. Review the containment,
writability, stable identity, and physical-independence evidence before
confirming. The recovery passphrase is write-only and ephemeral. It must be
held separately from the encrypted recovery artifact and is never entered in a
shell argument or saved in operator state.

An apply is ready only when its owner reports verified evidence. For credential
escrow that means an encrypted root-copy read-back, an encrypted recovery-bundle
decrypt/read-back with current coverage, and a supported schedule (or an
explicit degraded/manual schedule result). For durable data protection, the
data-backup-manager owner separately supplies destination readiness, a
successful snapshot, a scratch restore checksum, and a verified recovery
drill. Onboarding renders these owner reports; it does not execute backup or
restore operations.

If an apply returns `degraded` or `retryable_failure`, keep the existing valid
artifact, follow the returned remediation, and retry after correcting that
condition. A missing candidate, unknown physical independence, stale coverage,
or unsupported scheduler is not a success. Re-running setup, status, preview,
or apply is safe and converges through the provider's idempotency key.

Do not promote the current disabled temporary sink during migration. Preserve
the old configuration until the operator has selected a new sink and reviewed
the first successful evidence set.

## Apply reported partial

The report names each failed item, its remediation, and which dependants were
skipped because of it. Fix the named cause and re-run apply — it is idempotent,
so already-satisfied items are reported and not repeated.

## An operator-state write was rejected

The rejection names the failing schema path. The stored document is untouched, so
there is nothing to roll back. Correct the value and patch again.

Never hand-edit `.vrooli/operator-state.json` to work around a rejection: the
schema sets `additionalProperties: false`, and an invalid hand edit is
unrecoverable without another hand edit.

## Operator state looks wrong or truncated

1. `vrooli-onboarding operator show` — read the committed document.
2. Compare against `.vrooli/schemas/operator-state.schema.json`.
3. If `trust_posture` or `core` are missing, a writer bypassed the state service.
   That is a defect, not a configuration problem — file it. Restore the values
   through `operator patch`, not by editing the file.

## A wizard step reports a missing catalog

The step names which catalog it could not resolve. That is a deployment-tier
problem, not a host fault:

- Repository install — `VROOLI_ROOT` is unset or points somewhere else.
- Desktop bundle — the bundle did not stage that catalog. The bundle contract
  declares the required paths and packaging verifies them, so a bundle that
  reaches an operator in this state is a packaging defect.

## Onboarding a remote host

```bash
# through vrooli-bridge, against the remote host
vrooli-onboarding wizard apply --selection selection.json
vrooli-onboarding wizard status --json    # readiness and committed state
```

If credential provisioning fails, the remote host almost certainly has no
graphical session. Initialize the encrypted file store there first
(`vrooli credentials store init`), then re-run.

## Escalation

- Credential lifecycle, keyring repair, recovery bundles → `secrets-manager`.
- Host tool installation and safeguard application internals → the control plane.
- Connector and OAuth flows → integration-hub, when it ships.
- Anything in [Problems](../internal/PROBLEMS.md) is known and tracked; check
  there before filing.
