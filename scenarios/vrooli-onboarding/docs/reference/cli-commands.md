# CLI Commands

`vrooli-onboarding` is a full peer of the web UI, not a diagnostic shim. Every
decision the UI can make, the CLI can make — interactively for a human at a
terminal, declaratively for automation, vrooli-bridge, and scenario-to-cloud.

Identical choices through the UI, the interactive CLI, and the declarative CLI
produce byte-identical operator state. All three write through one service.

Run `vrooli-onboarding <group> --help` for the current flags. This page carries
the contract, not the flag list.

## The wizard

```bash
vrooli-onboarding wizard run --interactive     # same nine-step flow used by the UI
vrooli-onboarding wizard run --accept-recommendation --non-interactive # apply the explicit starter profile
vrooli-onboarding wizard status                # step pointer + which steps are satisfied
vrooli-onboarding wizard commit --selection "<file>"   # non-interactive, no prompts
vrooli-onboarding wizard export --output "<file>"     # current selection as a selection document
```

`wizard run` without a mode is a read-only catalog response for compatibility
with scripts that only need to inspect scenarios. `--non-interactive` never
guesses: it returns `needs_input` unless `--accept-recommendation` explicitly
authorizes the manifest-derived starter profile. If bootstrap has queued a
typed operator input, resolve it through the onboarding UI/API first; the
declarative CLI does not accept passphrases or other secrets as flags.

The interactive wizard walks the same nine steps in the same order, with the
same derived consequences and the same locked system set. The presentation
differs; the decisions do not.

A **selection document** names the capabilities an operator wants, not the
internal state shape:

```json
{
  "scenarios": ["swarm-manager", "browser-automation-studio"],
  "optional_resources": ["ollama"],
  "host": { "tools": ["scenario-tool"], "safeguards": ["kernel_config"] },
  "operating_mode": { "swarm-manager": { "auto_restart": true } },
  "apply": true
}
```

This is the surface automation drives. It is stable, reviewable, and diffable,
and it is what makes remote onboarding possible without hand-editing JSON over
SSH.

## Inspecting the derived stack

```bash
vrooli-onboarding scenarios list                 # deps and effective choices
vrooli-onboarding closure                        # transitive closure of the selection
vrooli-onboarding resources list                 # required · optional · standalone
vrooli-onboarding wizard export --output "<file>" # current selection as a target document
```

`union export` is what bundle packaging, VPS provisioning, and vrooli-bridge
consume to decide what to ship.

## Credentials

```bash
vrooli-onboarding credentials list               # descriptors + configured status, never values
printf '%s' "$VALUE" | vrooli-onboarding credentials provision --logical-id <id> --field <field>
vrooli-onboarding credentials doctor             # backend condition and its fix
vrooli-onboarding store status                   # metadata-only encrypted-store status
printf '%s' "$PASSPHRASE" | vrooli-onboarding store init
printf '%s' "$PASSPHRASE" | vrooli-onboarding store unlock
printf '%s\n%s\n' "$CURRENT" "$NEW" | vrooli-onboarding store change-passphrase
printf '%s' "$PASSPHRASE" | vrooli-onboarding store rewrap
```

A value is read from standard input only. A value-bearing flag is **rejected**,
not warned about — argv is visible in the process table and in shell history.
The store commands follow the same rule for passphrases. Backend selection is
metadata-only: `vrooli-onboarding store select --backend native` (or
`encrypted-file`) never accepts a secret.

`secrets-manager` owns credential lifecycle beyond provisioning: listing
declarations, keyring inspect and repair, and recovery-bundle export and
restore. Onboarding does not duplicate that surface.

On a host with no graphical session — a VPS, a CI runner, a headless bundle host
— no native store exists, so initialize the encrypted file store once before
provisioning anything. `doctor` names this condition explicitly; see
[troubleshooting](../guides/troubleshooting.md).

## Host tools and safeguards

```bash
vrooli-onboarding host list                                        # tools + safeguards with risk and privilege
vrooli-onboarding host list
vrooli-onboarding host set-config --name <safeguard> --key <k> --value-json <json>
```

`host list` shows `risk`, `privilege`, `bundling`, and supported platforms before
a choice, because a safeguard modifies host state. `set-config` validates against
the safeguard manifest's own schema and rejects an invalid value with the failing
path named.

## Apply and readiness

```bash
vrooli-onboarding apply                                       # install, apply, enable, start — per-item report
vrooli-onboarding readiness [--json]                          # blockers control the exit code
vrooli-onboarding readiness acknowledge-degraded [--digest D] # accept the current optional gaps
vrooli-onboarding wizard status [--json]                      # readiness and committed state
```

`readiness` prints every blocker with its reason and remediation, then exits
non-zero while one remains. Automation cannot branch on prose; the exit code is
what lets bridge, cloud provisioning, and CI gate on a real result. The verdict
comes from the API's typed blockers rather than from a second derivation in the
CLI, so the wizard and the completion marker cannot disagree.

Optional gaps do not block, but they do need an explicit acceptance before
configuration is reported complete. `readiness` names the digest of the current
degraded set and `readiness acknowledge-degraded` records it; with no `--digest`
the command reads the current one. The acknowledgement is durable operator state
and names the exact set it accepted, so accepting one gap never authorises
completion over a different gap later.

## Operator state

```bash
vrooli-onboarding operator show [--effective]     # committed document, or resolved values
vrooli-onboarding operator patch --body-file <patch.json>
```

`patch` sends an [RFC 7386](https://www.rfc-editor.org/rfc/rfc7386) JSON Merge
Patch. Only the named fields change; everything else in the stored document is
preserved, including fields this binary does not model. A `null` removes a key.

This is the escape hatch for a decision the wizard has not surfaced yet. It is
not the normal path — if you find yourself reaching for it routinely, the wizard
is missing a step and that is the defect to file.

## Glossary and status

```bash
vrooli-onboarding glossary [--query <term>]
vrooli-onboarding status
```

## Retired

| Command | Replacement |
|---|---|
| `setup-order` | `closure` |
| `operator apply --body-file` (whole-document replace) | `operator patch --body-file` (merge patch) |
| `operator set-safeguard-config` | `host set-config` |
| `config generate` / `config validate` | None. Onboarding never authors `service.json`; use `readiness` to check an install |

A command that targets a route the router does not register is a test failure,
not a runtime surprise.
