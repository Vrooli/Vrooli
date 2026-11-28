# Problems — Scenario to Plugin

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

Append entries as they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-08-19 — Published versions are never re-verified against a changed CLI

**Symptom:** A skill published today documents commands that exist today. If
the wrapped scenario later renames or removes one of those commands, the
published package silently becomes wrong. Users discover it; we do not.

**Root cause:** The drift gate (`PLG-CONF-DRIFT`) runs at composition time
only. Nothing re-runs it against already-published versions when the wrapped
CLI moves.

**Workaround:** None. Drift in a published version is currently discovered by
report. Revoke, fix the skill, republish.

**Real fix:** A scheduled job that re-resolves each published version's
documented commands against the current `cli-manifest`, using the recorded
`manifest_pins` revision as the baseline for the comparison. This is the
reason `PLG-CONF-DRIFT-PIN` records the revision — the data will be there.

**Owner:** unassigned.

**Refs:** `docs/internal/SECURITY.md` (Security Gaps), `OT-P0-002`,
`PLG-CONF-DRIFT-PIN`, `docs/operations/OBSERVABILITY.md` (Telemetry Gaps).

### 2026-08-19 — Retention rules are documented but not enforced

**Symptom:** Artifact trees, rehearsal logs, and superseded declaration
snapshots accumulate without bound.

**Root cause:** `docs/concepts/DATA.md` defines retention windows, but no
scheduled or event-driven lifecycle owner covers the declared data classes.

**Partial disposition (2026-11-24):** Composition now prunes package rows and
artifact trees older than 90 days when a new eligible package is composed. It
excludes every package with a publication row (including withdrawn rows),
serializes pruning against in-flight Publish, and refuses artifact paths
outside its scenario-owned package directory. See the PT2 evidence for the
owner regression tests. This does not run while composition is idle and does
not prune readiness history, rehearsal logs, declaration snapshots, or install
attributions.

**Remaining fix:** Add an owned scheduled lifecycle for all DATA.md classes,
with `publications`, `revocations`, and their referenced attestations
explicitly excluded — losing publication history loses the revocation fan-out.
Until then, do not perform manual capture-store cleanup of referenced data.

**Owner:** unassigned.

**Refs:** `docs/concepts/DATA.md` (Retention And Deletion).

### 2026-08-19 — The `skill-registries` channel has no machine-evaluable trigger

**Symptom:** Offer Desk carries `skill-registries` as `CANDIDATE`, but
`offers gates-evaluate` returns no evaluation for it. The channel cannot be
promoted from evidence — only by argument.

**Root cause:** The channel's activation trigger exists as prose in
`docs/monetization/catalogs/channels/skill-registries.md` and was never
registered with `offer-desk offers gates-trigger`.

**Workaround:** None needed yet; the channel's prerequisites are unmet
regardless.

**Real fix:** Register the trigger. It is cheap and does not depend on this
scenario shipping. Not this scenario's code, but it is this scenario's
downstream consumer, so it is tracked here until someone owns it.

**Owner:** unassigned (monetization).

**Refs:** Offer Desk channel node `153e6270-47db-442b-ae22-bf34a95b8090`,
`docs/business/MONETIZATION.md` (Current Status).

### 2026-08-19 — The business-health wizard could not be used to author this PRD

**Symptom:** Every `WizardService` RPC — `StartSession`, `SubmitAnswers` —
returns an empty reply after exactly 51 seconds. The API logs
`status=200 duration=51.04s` but the client receives nothing.

**Root cause:** The wizard's state-render path runs a capability-dedup search
whose reranker calls Ollama. Ollama reports healthy but `generate` returns
`signal: killed` on every attempt, so the search retries for ~51s and the
response never lands. It is an environmental Ollama wedge plus a missing
degradation path in the dedup search — the search should degrade to no hints
rather than blocking the RPC.

**Workaround:** Author `PRD.md` directly against
`scenarios/business-health/docs/reference/canonical-prd-template.md`, then
validate with `vrooli scenario requirements validate`. The validator, not the
wizard, is the contract; the wizard is a conformant-by-construction
convenience. This PRD validates clean with zero findings.

**Real fix:** Belongs to business-health, not here: bound the dedup search and
degrade to zero hints on reranker failure. Filed as an observation rather than
a fix because the defect is in another scenario.

**Owner:** unassigned (business-health).

**Refs:** `business-health` API log, `WizardService/StartSession`;
`docs/internal/PROGRESS.md` 2026-08-19.

### 2026-08-19 — Every control in this scenario is designed, not proven

**Symptom:** Owner tests now exercise local broken packages through composition,
conformance, dry-run attestation, and publish refusal, but the full release
checklist remains unproven as one complete, green release run.

**Root cause:** The scenario was designed contract-first. Individual controls
and focused tests landed before an integrated fixture run; that run exposed
gaps in checksum mapping and credential handling. The local refusal path is
now exercised, but the whole checklist and a managed successful release have
not been run.

**Workaround:** Cite only the specific control and owner test that has been
verified. Do not infer end-to-end release readiness from focused tests.

**Real fix:** Keep the local gate self-test in `docs/operations/DEPLOYMENT.md`
step 7 current and expand its broken fixtures as gates change. Then run all
release checklist steps with their supported owner tools. External publication
and signing policy remain separate operator boundaries.

**Evidence update 2026-11-24:** Owner tests found and closed two fail-open
paths in those controls: a hash command in a comment or a digest-only command
could satisfy the install checksum check, and `Publish` queried
deployment-manager before rejecting a credential-bearing artifact. The
checksum gate now requires at least as many SHA-256/SHA-512 check-mode lines
as detected downloads, and publication refuses an artifact secret before the
release-gate request. Six checksum fixtures and a request-counting release-gate
test pass with the full pipeline owner package; the trimpath API suite also
passes after making fixture-root lookup independent of `runtime.Caller`. This
does not prove that each verifier line checks the corresponding downloaded
file. The broad entry remains open: the release checklist has not been
exercised as one broken-package ramp across all gates. See
`evidence/PT2-SCENARIO-TO-PLUGIN-PREFLIGHT-GATES.md`.

**Evidence update 2026-11-24:** The OCI publish boundary also accepted any
referrer index with three entries, including repeated signature descriptors
or entries with no evidence type, without fetching those manifests. Publish
now requires distinct signature, provenance and SBOM descriptor types, fetches
each referenced manifest, verifies its content digest and artifact type, and
checks that its subject is the uploaded package digest. Local registry tests
cover the valid three-manifest path and refuse missing or duplicate evidence,
unknown or untyped descriptors, a manifest bound to another package, and an
unavailable manifest. The focused pipeline package and full trimpath API suite
pass. The Test Genie unit run still fails on existing UI `type-check` and
`test:coverage` commands. External registry publication and the complete
deliberately broken-package ramp remain unverified; retain this entry open.
See `evidence/PT2-SCENARIO-TO-PLUGIN-OCI-REFERRER-CONFIRMATION.md`.

**Evidence update 2026-11-24:** The local self-test now composes a drifted
command fixture and a mutable-download fixture, confirms their named
conformance refusals, confirms dry-run attestation and publish refuse to
advance them, and verifies no credential or evidence artifacts were emitted.
A credential-literal fixture is rejected during Compose before archive/package
creation or the local release-gate request. The broad token scanner now
matches token-shaped values rather than `sk-`/`ghp_` substrings, so the normal
Workspace Sandbox binary is not falsely rejected for embedding the scanner's
own regex. Checksum validation associates each distinct verifier with the
download output's exact `.sha256` or `.sha512` sidecar name; comments, digest
printing, weak SHA-1, unmapped/duplicate verifiers, and mutable URLs fail.
Fixtures cover curl/wget output forms and multiline commands. Owner package
and complete trimpath API tests pass. Test Genie unit `20261124-025937-88fcfabe`
still fails on existing UI type-check and coverage commands. This closes the
local step-7 refusal proof for these fixtures only; complete checklist,
managed signing, and external publication remain open. See
`evidence/PT2-SCENARIO-TO-PLUGIN-RELEASE-CHECKLIST-LOCAL.md`.

**Evidence update 2026-11-24:** A local fake downloader now supplies real
payload bytes and checksum-sidecar bytes to the install fixture. The actual
`sha256sum --check` accepts the matching digest/path and refuses both a wrong
digest and a sidecar naming another file. The full local API flow also reaches
Publish with schema-shaped test evidence, a local release-gate response, and
a fake OCI registry: the valid package confirms its artifact and three
referrers, while an index omitting the SBOM refuses with `PLG-DIST-CONFIRM`.
No actual signing, operator release policy, external registry, or publication
was used. Focused fake-flow tests, full pipeline package, and full trimpath
API suite pass. Test Genie unit `20261124-031214-9621667b` fails on existing
UI type-check and coverage commands. This verifies selected local success and
refusal rows but does not complete the full release checklist. See
`evidence/PT2-SCENARIO-TO-PLUGIN-CHECKSUM-CONTENT-AND-LOCAL-PUBLISH.md`.

**Evidence update 2026-11-24:** The synthetic managed Cosign gate accepted a bundle with a message signature object but no digest binding, so a bundle for a different payload could pass Attest's structural check. It now requires `messageSignature.messageDigest` to use `SHA2_256` and match the exact composed artifact digest. Owner tests accept the matching digest and refuse missing, wrong-algorithm, and mismatched digests; the focused release fake and full API suite pass. These JSON fixtures are synthetic and do not verify a cryptographic signature or managed authority. Real signing, external registry behavior/publication, and the complete release checklist remain unverified; this broad finding stays open. See `evidence/PT2-SCENARIO-TO-PLUGIN-COSIGN-DIGEST-BINDING.md`.

**Evidence update 2026-11-24:** The local fixture for checklist step 8 reproduced a secret disclosure: rehearsal output redaction replaced only selected marker prefixes, leaking token suffixes and other supported token prefixes. The redactor now removes complete detector-matched tokens and complete PEM private-key blocks. A local end-to-end fixture checks the composed archive, dry-run evidence, API responses, rehearsal capture, and bytes sent to a local fake verdict receiver; it constructs six token-shaped values only at runtime and confirms the secret material is absent from outputs. Focused owner tests and full non-coverage Go API tests pass. Cosign cryptographic verification remains unproven: this repository has no verifier dependency/binary or configured trusted identity material, and a synthetic self-managed key would not prove the managed keyless authority. Keep the broader release-ramp item open. See `evidence/PT2-SCENARIO-TO-PLUGIN-SECRET-CAPTURE-AND-COSIGN-BOUNDARY.md`.

**Evidence update 2026-11-24:** The next local UI gate reproduced two concrete failures: the existing Input tests imported a missing `ui/src/components/ui/input.tsx`, and the AppShell accessibility test did not wait for its asynchronous health query. Added the forwarded-ref Input primitive and mocked the health API in the accessibility test, waiting for its settled status. Focused UI tests pass (7), `pnpm run type-check` passes, and UI coverage passes (72 tests; 89.62% statements, 83.92% functions). Requirements validation still reports stale business evidence; orientation returns a Template Manager 500; Test Genie unit `20261124-033211-502ca384` failed before scenario tests at shared Scenario-to-Cloud proto preflight (`buf.validate.field` unknown). The release checklist and broad finding remain open. See `evidence/PT2-SCENARIO-TO-PLUGIN-LOCAL-UI-REQUIREMENTS-GATES.md`.

**Evidence update 2026-11-24:** The local requirements sync changed no files or statuses (0/18 targets and 0/36 sync-complete requirements); the registry still has 40 planned requirements, 36 live statuses pending, and none complete. Test Genie business phase `20261124-035727-249ffade` passes but reports `business_evidence_stale`: the local snapshot predates that run, so a business-only phase does not complete the documented comprehensive-suite sync. The active Settings page still lacks desktop/mobile/mobile-landscape captures. BAS is healthy, but normal Scenario-to-Plugin start failed when the required running Workspace Sandbox rebuild hit its dirty missing `./ui/card` source; neither dependency nor Template Manager paths were changed. A direct production UI build passes to a temporary output directory. Static UI health remains `NOT_EVALUATED` in static-only mode with existing slot/component findings, including the test-only Input primitive. Keep the broad release-ramp issue open. See `evidence/PT2-SCENARIO-TO-PLUGIN-REQUIREMENTS-EXPERIENCE-GATES.md`.

**Owner:** unassigned.

**Refs:** `docs/operations/DEPLOYMENT.md` (Release Checklist),
`docs/internal/SECURITY.md` (Security Gaps).

## Architecture Drift

Use this section for deferred findings from `architecture-lenses`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| `docs/internal/SEAMS.md` | Still describes only the template's example seams. The six real pipeline domains register no seams yet. | Tests that depend on domain seams cannot be written until the registry is real. | Register each domain's seams as that domain lands, before writing its tests. |
| `docs/reference/api-endpoints.md`, `cli-commands.md` | Describe the planned surface, not an implemented one. Marked `draft` in `docs/manifest.json` to say so. | Reference docs that read as current truth would misreport maturity. | Regenerate from the proto and `cli/manifest.json` once the first domain ships. |
| `experience/` | Page specs are L0/L1 intent for routes that do not exist yet. | Experience maturity cannot exceed intent until routes are real. | Raise to L2+ as each route lands with stable roles and testids. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
