# Quickstart

## In a browser

```bash
cd scenarios/vrooli-onboarding
make start
```

Open the lifecycle-managed UI and work through the ten steps:

1. **Welcome** — choose a purpose profile and review its recommendations. A
   profile can recommend capabilities without selecting an optional scenario.
   Wait for the profile session to report **Saved** before leaving this step.
2. **Scenarios** — search the catalog and pick at least one optional scenario
   when none is selected. Required system capabilities stay included. Each
   choice shows what it pulls in.
3. **Core supervision set** — review the shared recovery plane, choose the seed, and
   inspect its computed scenario/resource closure.
4. **Resources** — required resources are already selected; add optional ones.
5. **Credentials** — each card explains what the credential unlocks and links to
   where you get one. Values go straight to the credential authority.
6. **Integrations** — review declared requirements as deferred. The current
   OpenRouter Hub pilot is separate; this step does not create bindings.
7. **Host** — consent to the tools and safeguards your selection needs. Risk and
   privilege are shown before you choose.
8. **Operating mode** — confirm lifecycle auto-restart preferences; this does
   not change supervision membership.
9. **Apply** — this is the step that changes your host. You get a per-item report.
10. **Validation** — live probes. Fix anything required, then recheck.

Reopening the wizard resumes at the first unsatisfied step using saved state. If
a profile save reports an error, retry and confirm **Saved** before closing;
unsaved answers cannot be recovered from durable state. Earlier choices remain
editable until you apply. Apply is the explicit host-change boundary.

## In a terminal

```bash
vrooli-onboarding wizard run --interactive=true
vrooli-onboarding wizard run --accept-recommendation=true --non-interactive=true
vrooli-onboarding wizard status
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
vrooli credentials store init
printf '%s' "$PASSPHRASE" | vrooli credentials store init
vrooli credentials doctor
```

## If something is not ready

The validation step names the cause and the next action for every item. Run
`vrooli-onboarding wizard status --json` for the same report as data. Backend
problems specifically are diagnosed by `vrooli credentials doctor` — an unset
value, an unreachable store, and a host with no backend each have a different
fix.

See [troubleshooting](guides/troubleshooting.md) and the
[runbook](operations/RUNBOOK.md).
