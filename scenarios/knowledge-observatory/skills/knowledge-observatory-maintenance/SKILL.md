---
name: "knowledge-observatory-maintenance"
description: "Clean up, consolidate, reorganize, correct, move, and retire documentation while preserving agent task knowledge, decisions, references, and OS applicability."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["documentation", "maintenance", "consolidation", "cleanup", "knowledge"]
  status: "active"
  revision: 1
  createdAt: "2026-09-06T00:00:00Z"
  updatedAt: "2026-09-06T00:00:00Z"
  requires:
    scenarios: ["knowledge-observatory", "program-runtime"]
    commands: ["knowledge-observatory knowledge-base", "program-runtime library run"]
  origin:
    kind: "authored"
---
## Tools focus: Knowledge Observatory Maintenance

Preserve the knowledge that readers need while correcting, consolidating, or
reorganizing a selected document family. Produce a disposition for every selected
source and check each preservation obligation before declaring completion.

Read `prompt-manager skill read knowledge-observatory` for authority, applicability,
and the single recall/capture loop. Read
`path:scenarios/knowledge-observatory/docs/guides/document-maintenance.md` for the
input contracts and worked example. Apply `documentation-health` for placement and
code/document traceability, and `writing-standards` for the artifact's writing rules.
This feature skill owns editorial maintenance judgment. It does not own accepted
policy, plan closeout, a second learning loop, or an autonomous deletion policy.

### Select the maintenance route

For a broad cleanup candidate, first run the read-only inventory. It returns one
record per bounded source with a source hash, owner hypothesis, artifact class,
portability signals, and separate `inspect:` / `propose:` handles:

```text
knowledge-observatory inventory --roots=docs,scenarios --max-files=100
```

Use Git Control Tower blame for line authorship, then request proposals. Native
Git author data and maintenance evidence are different dimensions: exact-content
standing requires a complete content digest and matching commit; path overlap,
run IDs, trailers, and search rank remain downgraded evidence. Private evidence
is withheld. Inventory and proposal commands never edit files.

| Observed condition | Decision and next step |
|---|---|
| A single local mistake has an established correction | Inspect the source revision; apply the authorized correction and check its references. Keep the inspection a CLI leaf. [S1] |
| Multiple documents overlap, conflict, or need new locations | Run `knowledge-observatory.prepare-change` with exact paths, intent, and a scan root covering their known consumers. Complete its dispositions using the table below. [S3] |
| HTML or loose artifacts contain unique knowledge | Inspect the original source. Promote accepted reusable knowledge into an existing canonical document. Keep necessary experiment evidence with its plan owner. [S1] |
| The reviewed dispositions and preservation obligations are ready | Reinspect source revisions. Apply the authorized edits through the source owner's workflow. Update incoming references, navigation, manifest declarations, and headings together. [S0] |
| An edit is ready for verification | Run `knowledge-observatory.verify-change` with intended revisions, preserved excerpts, absent paths, retrieval cases, and the unchanged preparation baseline. [S3] |
| A recurring failure impedes maintenance | Route the observed failure and comparable task evidence to `knowledge-observatory-improve`. [S0] |

| An owner-approved proposal is ready | Use `knowledge-observatory route` in dry-run mode first. The receipt is revision-guarded and idempotent; Plan Manager owns candidate application, artifact placement, and closeout. |

### Complete the disposition

| Evidence | Action |
|---|---|
| Accepted instructions disagree with current owner contract | **correct**; cite the contract or tested behavior. Preserve the intended requirement when implementation is the defect. |
| Sources repeat the same accepted knowledge | **consolidate**; choose the existing canonical destination and preserve distinct qualifications. |
| A supplement holds unique accepted knowledge | **promote**; transfer the knowledge and provenance to the canonical destination. |
| Knowledge is valid but located outside its reader task or owner | **relocate**; map old paths and headings to their replacements. |
| Historical rationale, unresolved work, or unique evidence remains useful | **retain**; mark its time/scope/status and owner. |
| All useful knowledge and required evidence have destinations, and consumers are accounted for | **retire** within the authorized scope; state why nothing else must survive. |
| Authority, destination, preservation, or ownership is unresolved | Leave the decision **proposed** or **unresolved**. Resolve it from owner evidence or ask only for the missing decision. |

A disposition names source, action, destinations, reason, preservation obligations,
and decision evidence. File equality is evidence of byte duplication. It is not
a reason by itself to retire a source. Search rank is not authority.

Compare claims, instructions, exceptions, prerequisites, failure recovery, OS scope,
and unresolved issues. Record the important units that must survive. Cite their
original revisions. Split oversized families by reader task, while retaining
cross-family consumers in the coverage account. Use existing canonical locations;
load placement guidance rather than inventing a directory taxonomy.

### Completion and evidence

Read the whole relevant source, including pages after a truncated excerpt. Preserve
reviewed reader questions before editing. Review the resulting source against each
obligation. An excerpt match only proves those bytes at that revision; it does not
prove that surrounding prose keeps their meaning.

Interpret `verify-change` output as follows:

| Result | Next step |
|---|---|
| `failed` / `revision_changed` | Reinspect and prepare against current sources. |
| `failed` / `verification_failed` | Repair the named mismatch or new reference finding, then reverify. |
| `partial` / `editorial_review_pending` | Complete the source comparison; retain unresolved claims as gaps. |
| `ok` with retained reference debt | Report the retained debt and the scope. Mechanical non-regression passed. |
| `ok` with editorial review incomplete | Finish editorial preservation review before claiming maintenance complete. |
| `ok` with all obligations reviewed | Confirm the requested reader tasks using the final sources, then capture the actual task outcome through the usage skill. |

Keep working dispositions and receipts in the task's designated plan artifact
location. For a small task without a plan, keep the review in the response and
record reusable learning in Memory. Durable policy, rationale, and unresolved
issues belong in their existing owner documents. Use JSON when transferring exact
program evidence and hashes; use normal CLI output for orientation.

### Troubleshooting & Edge Cases

- **Scan or excerpt truncated:** inspect the remaining source pages; narrow the
  scan or increase its declared limit. State uncovered consumers. A bounded scan
  cannot establish repository-wide reference closure.
- **Baseline incompatible or incomplete:** obtain a complete baseline before
  editing. Do not manufacture a clean baseline after a change to hide regressions.
- **Known findings changed location:** full finding identity is conservative.
  Inspect the apparent new finding; document the comparison rather than suppress it.
- **Index lags source:** inspect live revisions and report retrieval separately.
  Use text mode for a lexical check; retain the original semantic test as pending.
- **Policy conflict:** compare scope, owner contract, and evidence. Ask the owner
  only when the approved task and available evidence do not resolve the decision.
- **Program discovery misses the task:** use the declared program name above.
  Record the query as friction; improve discovery instead of adding a wrapper program.
- **Legacy healing or log reset offered:** check its owner contract and preservation
  requirements. A rising health score or old timestamp does not authorize retirement.

This workflow uses repository-relative paths. It requires no Git checkout, reset,
clean, or branch operation. A Linux observation does not certify Windows or macOS.

The Maintenance Queue is the bounded UI companion to this workflow. It makes
inventory truncation, unavailable hashes, portability signals, evidence
standing, owner hypotheses, and proposal uncertainty visible. Owner routing is
revision-guarded and idempotent; dry-run is the default. A route requires a
source hash, a reason, preservation and evidence notes, and owner authorization
for any non-dry-run request. The queue does not delete, move, or silently
rewrite candidate artifacts.
