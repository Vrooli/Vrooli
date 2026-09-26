# Security — Scenario to Plugin

This scenario has an unusual security position: it is the last checkpoint
before Vrooli-signed content reaches machines Vrooli does not control, and
it processes untrusted content as its normal input.

Two consequences shape everything below:

1. **Blast radius is the channel, not the artifact.** One flagged package
   lowers the trust score of every other Vrooli package. A permissive
   decision here is not a local risk.
2. **Its inputs are adversarial by construction.** A skill body under
   review may contain prompt injection aimed at whatever reads it — a
   scanner, a reviewer, a log aggregator, or an agent. This scenario must
   treat its own inputs as hostile.

## Purpose Of This Document

Use this document to answer:

- What data does this scenario handle, and how sensitive is it?
- Who may do what?
- How are credentials handled?
- What are the threats, and which control addresses each?
- What is knowingly unprotected today?

## Data Sensitivity

| Data | Sensitivity | Handling |
|---|---|---|
| Skill bodies and install scripts under review | **Untrusted input** | Never executed on the host. Never logged in full. Findings carry file and offset, not content. |
| Rehearsal command output | Potentially sensitive | Redacted at capture time, not at read time. |
| Artifact digests, signatures, provenance, SBOMs | Public by design | Published alongside the artifact; permanently retrievable once published. |
| Registry and signing credentials | **Secret** | Never held. Only references, resolved at call time through `secrets-manager`. |
| Publication and revocation history | Operationally critical | Retained indefinitely; the revocation fan-out is derived from it. |
| Operator identity on publish/revoke | Attributable action record | Retained. Release actions must be attributable. |
| End-user personal data | None | This scenario's subjects are scenarios, packages, and channels. |

## Auth And Authorization

| Action | Who | Enforcement |
|---|---|---|
| Read readiness, packages, findings, evidence | Any operator with scenario access | Read-only; no gate can be changed by reading. |
| Compose, check, attest, rehearse | Any operator with scenario access | These produce records, not outward effects. Safe to run freely — and running them freely is encouraged, because unpublished failures are cheap. |
| **Publish** | Operator, **and** a `deployment-manager` release decision for the same source commit | Two-party by construction: the ramp cannot authorize itself. |
| **Revoke** | Operator | Deliberately *not* gated on a release decision. Withdrawal must be fast under incident conditions; requiring approval to stop shipping something is the wrong default. |

The publish/revoke asymmetry is intentional. Making something reachable
requires two parties; making it unreachable requires one.

## Secrets

- This scenario holds **no** credential literals and **no** signing keys.
  `secrets-manager` owns credentials; the managed release-signing
  authority owns keys. This scenario holds references and resolves them at
  the moment of the call.
- `PLG-ATTEST-NO-SECRETS` fails a package if a credential literal appears
  in an artifact, an SBOM, or an attestation. The check runs **before any
  network call**, because a published attestation is permanently
  retrievable and cannot be recalled by deleting the artifact.
- No secret is ever written to a log, a verdict, a finding, or a rehearsal
  capture.
- **Privilege is never escalated.** The ramp runs entirely as the invoking
  user, and `vrooli setup` remains the only place privilege is requested.
  `PLG-CONF-INSTALL-PRIV` additionally fails any packaged install script
  that requests elevation or writes outside a user-scoped prefix — a
  published artifact runs on a machine we do not own, so an install that
  needs root is out of scope for this channel regardless of intent.

## Threat Model

| # | Threat | Control | Requirement |
|---|---|---|---|
| T1 | A compromised or misconfigured pipeline signs a package that never passed its checks | Signing is refused unless a passing conformance record exists; ordering is enforced in the domain, not by caller discipline | `PLG-ATTEST-ORDER` |
| T2 | An install script fetches different bytes later than it did at review time | Immutable references only, plus an independent checksum verification | `PLG-CONF-INSTALL-PIN`, `PLG-CONF-INSTALL-SUM` |
| T3 | Prompt injection hidden in a skill body manipulates a consuming agent | Hidden-Unicode, bidi-mark, NFC, and angle-bracket rules fail the package; findings never echo the body | `PLG-CONF-UNICODE`, `PLG-CONF-ANGLE` |
| T4 | Over-broad `allowed-tools` enables privilege escalation in the consuming runtime | Unrestricted shell, network, or filesystem grants are rejected | `PLG-CONF-TOOLS` |
| T5 | A skill documents commands the wrapped CLI no longer has, producing undefined behavior | Every documented command is resolved against a pinned `cli-manifest`; the revision is recorded | `PLG-CONF-DRIFT`, `PLG-CONF-DRIFT-PIN` |
| T6 | A vulnerable transitive dependency ships inside a package | SBOM attached as a referrer so downstream scanners can correlate; scanner findings fail the package | `PLG-ATTEST-SBOM`, `PLG-ATTEST-SCAN` |
| T7 | A credential or user data leaks through an artifact, attestation, or log | Pre-publication literal check; capture-time redaction; references-only logging | `PLG-ATTEST-NO-SECRETS` |
| T8 | A package advertised as standalone quietly pulls the full Vrooli runtime | Clean-room rehearsal compares actual acquisitions against the declaration | `PLG-REHEARSE-NO-STEALTH` |
| T9 | A published artifact is found to be harmful and cannot be withdrawn | Revocation fan-out derived from recorded publication history; partial revocation reported truthfully | `PLG-DIST-REVOKE`, `PLG-DIST-REVOKE-PARTIAL` |
| T10 | An approval for one build is reused to publish a different one | The release decision is bound to an exact source commit | `PLG-DIST-GATE` |
| T11 | An install corrupts a user's machine when an agent re-runs it after a partial failure | Install is executed twice in one rehearsal and must be a no-op the second time | `PLG-REHEARSE-IDEMPOTENT` |
| T12 | A package is recorded as published when the registry silently dropped it | Publication is confirmed by retrieval at the published digest | `PLG-DIST-CONFIRM` |

### Structural advantages

Two properties are ours by architecture rather than by control, and they
are worth stating because they change what the controls above have to
carry:

- **Wrap-not-use.** The wrapped CLI enforces auth, scope, and audit at its
  own layer. A skill cannot widen the CLI's permissions, so a narrow
  `allowed-tools` declaration is honest rather than cosmetic.
- **Sandboxed rehearsal.** The clean-room install runs under
  `workspace-sandbox`, which already owns the accountability substrate.
  We do not verify installs by running them on a developer's host.

### Design rule: no model in a gate

No gate in this pipeline may depend on model output. Every gate must be
deterministic and reproducible by a third party from the emitted evidence
— a registry, an auditor, or a consuming agent. A model-judged
conformance check would make the verdict unreproducible and the trust
claim unverifiable. Advisory output (for example, a drift-repair
suggestion) may use a model; a decision may not.

## Security Gaps

Known and unprotected today. None of these should be described as covered.

| Gap | Risk | Status |
|---|---|---|
| Published versions are not re-verified after the wrapped CLI changes | A published skill can drift post-publication; users discover it, we do not | Open. `manifest_pins` makes it possible; the job is unscheduled. Tracked in `PROBLEMS.md`. |
| No implemented code exists yet | Every control above is designed, not proven | Expected — this scenario is pre-implementation. The release checklist requires a gate self-test against deliberately broken fixtures before any real publication. |
| Scanner coverage depends on third-party tools | A scanner blind spot becomes our blind spot | Accepted. Mitigated by the drift gate, which no external scanner performs. |
| Retention is documented but not enforced | Artifacts and rehearsal logs accumulate | Open. Tracked in `PROBLEMS.md`. |
| Composite packages multiply the revocation surface | A composite would need fan-out across several scenarios' channels | Deferred by design (`OT-P2-002`); do not implement before single-scenario revocation is proven. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — retention, redaction, privacy posture
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency failure modes; every failure closes a gate
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — where each control lives
- [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md) — incident response for a bad published package
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist, including the gate self-test
- [`../../requirements/03-conformance/module.json`](../../requirements/03-conformance/module.json) — the conformance requirements
