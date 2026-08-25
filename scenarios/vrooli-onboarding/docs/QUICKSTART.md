# Quickstart

## In a browser

```bash
cd scenarios/vrooli-onboarding
make start
```

Open the lifecycle-managed UI, then work the nine steps (including welcome and
resume state):

1. **Scenarios** — search the catalog and pick the capabilities you want. System
   capabilities stay locked on. Each choice names what it pulls in.
2. **Resources** — required resources are already selected; add optional ones.
3. **Credentials** — each card explains what the credential unlocks and links to
   where you get one. Values go straight to the credential authority.
4. **Integrations** — empty until integration-hub ships.
5. **Host** — consent to the tools and safeguards your selection needs. Risk and
   privilege are shown before you choose.
6. **Operating mode** — confirm which scenarios stay running.
7. **Apply** — this is the step that changes your host. You get a per-item report.
8. **Validation** — live probes. Fix anything required, then recheck.

Reopening the wizard resumes at the first unsatisfied step. Nothing you have
already decided is lost or read-only.

## In a terminal

```bash
vrooli-onboarding wizard run --interactive      # same capability flow used by the UI
vrooli-onboarding wizard run --accept-recommendation --non-interactive # explicit starter profile
vrooli-onboarding wizard status   # readiness and committed state as JSON/text
```

## Without prompts

For automation, CI, vrooli-bridge, or a VPS:

```bash
cat > selection.json <<'JSON'
{
  "scenarios": ["swarm-manager", "browser-automation-studio"],
  "optional_resources": ["ollama"],
  "host": { "safeguards": ["kernel_config"] },
  "apply": true
}
JSON

vrooli-onboarding wizard commit --selection "selection.json"
vrooli-onboarding wizard status --json
```

## On a host with no desktop session

A VPS, a CI runner, and a headless bundle host have no native credential store.
Initialize the encrypted file store once, before provisioning anything:

```bash
vrooli credentials store init                              # a reachable TPM needs no passphrase
printf '%s' "$PASSPHRASE" | vrooli credentials store init  # otherwise
vrooli credentials doctor                                  # names the backend and the fix
```

## If something is not ready

The validation step names the cause and the next action for every item. Run
`vrooli-onboarding wizard status --json` for the same report as data. Backend
problems specifically are diagnosed by `vrooli credentials doctor` — an unset
value, an unreachable store, and a host with no backend each have a different
fix.

See [troubleshooting](guides/troubleshooting.md) and the
[runbook](operations/RUNBOOK.md).
