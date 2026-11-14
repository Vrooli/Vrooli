# UI Architecture

## Shared contract

Read the [UI Architecture shared guide](/scenarios/template-manager/docs/concepts/UI-ARCHITECTURE.md). Template Manager owns
the common contract and its implementation references.

## Scenario details

Use this scenario’s UI manifest and source tree for its actual layout.
Record local layout or adoption constraints here; the current template guide
does not establish that an older scenario has migrated.

## F9 coding-bounty review target · proposed

**Concept:** Evidence Workbench. The review surface is organized around an
operator's evidence chain: public task identity, sealed source/candidate/winner
identities, candidate patch, independent Test Genie outcomes, and a durable
review packet. It is an operator workbench, not a proposal-submission client.

**Content and actions:** queue/filter coding jobs; inspect the public issue and
winning PR; read the candidate patch and tests; compare preimage/candidate/winner
results and their seal/evaluator digests; inspect events and interrupted runs;
copy a code-free evidence packet; open the public source; resume an existing run.
The solver receives only task text plus its disposable checkout, and the scorer
receives receipt facts/digests only. The source/reference remain hidden from both.

**Composition:** Desktop is a simultaneous queue, selected-case summary, and
review pane. Mobile is a sequential queue → case → code/tests flow with a fixed
thumb-zone action bar. The concept and side-by-side mockup are proposed, not
operator-ratified. See `docs/mockups/f9-coding-bounty-review.html`.

**Current constraint:** the durable CodingJob record contains the task, source
seal ref, solver run ref, receipt ref/digest, score, revision, and events. Its
`solver_completed` event stores the candidate seal digest in `evidence_sha256`,
but not the sandbox ID. Workspace Sandbox's `replay-seal/resolve` owner path
can verify and resolve a `(caseId, candidate role, digest)` to the immutable
snapshot root for the declared Program Runtime; it returns a host path, not a
browser-safe list/read response. The Program Runtime result is intentionally
code-free. Workspace Sandbox now has an internal bounded candidate-file/page
reader, and CodingJob has a pure resolver for the durable candidate digest. They
are not wired together through an authenticated endpoint. The explicit
caller-access rule for the operator review surface is also still open. The UI
must not accept a caller-supplied host path, sandbox ID, reference role, or raw
digest.

**Required read contract before product UI:** the caller supplies only the
CodingJob ID and a bounded page of normalized relative file paths. Client
Rehearsal loads the durable job, requires `reviewable` state, and obtains the
candidate digest only from its durable `solver_completed` event. Workspace
Sandbox verifies the candidate seal once per page, checks each returned file
against its manifest digest, caps each file at 2 MiB and each page at 64 files / 8
MiB, and returns content plus file digests without returning a host path. A
single-file read is a one-item page. The operation has no write behavior and
cannot resolve source or reference roles. The read route must also be
restricted to the operator review surface; the inspected global API middleware
supplies logging and security headers but no caller identity, so this access
rule is still an open design item. The owner helpers are implemented, but this
service contract is not exposed as an API and product UI remains pending
ratification.

**Maturity:** L1 mockup + concept proposed; concept and selected mockups await
the two required human ratifications. Product UI code has not started. No
tokens or shared primitives are changed.
