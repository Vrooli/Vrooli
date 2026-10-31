# Security — Tech Tree Designer

## Purpose Of This Document

Define target safety boundaries for repository-wide drafts and application. These controls require implementation evidence; their specification does not assert that the current proto materializer provides them.

## Data Sensitivity

Repository drafts can contain proprietary source, architecture, requirements, proposed instructions and personal or secret material. Graph topology may reveal sensitive infrastructure. Logs, diffs and exports require the same access scope as their source. Exclude credentials, runtime databases, environment files and generated secret-bearing outputs by default.

## Auth And Authorization

Swarm or the qualified authority owner supplies the grant; the API/service verifies it before effects. Bind review to immutable revision and selected entries. A plan link, local host access or draft creation is not publication, execution, billing, commit or deployment authorization.

Recheck scope, expiry/revocation and owner capability at apply time. Material amendments require disposition under the governing mandate; do not silently reuse approval for changed bytes or effects. Reject unknown identities/grants rather than infer permission. Remote and multi-user deployment remain unqualified until identity, tenancy and access controls are tested.

## Secrets

Draft manifests and receipts contain identifiers and redacted findings, not credential values. Use existing secret owners for authorized runtime needs; never copy live credentials into proposal storage. Scan imports/exports at the appropriate owner boundary and report exclusions without leaking contents.

## Threat Model

| Risk | Required control and qualification |
|---|---|
| Path traversal, symlink escape, ambiguous root, overlapping selections | Canonicalize within an approved repository root; validate owner-specific paths and reject unsafe or overlapping entries before writes |
| Draft instructions becoming executable | Keep proposed skills/programs outside live discovery; publication uses their canonical owner and separate applicable authority |
| Prompt injection in imported files or graph text | Treat content as data; it cannot expand grants, select tools or override the mandate |
| Concurrent edits and replay | Recheck relevant base identities; use durable operation IDs and per-entry receipts |
| Partial writes or malicious rollback | Expose partial state; reconcile owner receipts; never restore over unrelated newer edits |
| Resource exhaustion | Enforce scope, byte, traversal, queue and retention limits; exercise adversarial graphs/artifacts |
| Experiment leakage | Qualified workspace plus control-plane runtime identity, ports, databases and network/billing restrictions; no execution by merely opening a draft |
| Stale or forged fulfillment | Keep provenance and evidence revision; a mapping or applied document is not proof of capability |

## Security Gaps

Generic proposal grants, publication isolation, bounded imports, partial-apply recovery and remote access are target obligations until tested. A documentation-only mandate does not authorize experiments, payment calls, publishing skills or changing live service configuration.

## Cross-References

- [Data](../concepts/DATA.md)
- [Flows](../concepts/FLOWS.md)
- [Testing](TESTING.md)
