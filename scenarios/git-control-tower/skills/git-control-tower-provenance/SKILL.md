---
name: "git-control-tower-provenance"
description: "Investigate exact AI and work provenance for a ChangeSubject without treating path overlap as authorship proof."
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["git-control-tower", "advisory", "provenance", "attribution"]
  status: "active"
  revision: 1
  requires:
    scenarios: ["git-control-tower"]
    commands: []
  origin: {kind: "authored"}
---

# Investigate provenance

Resolve run, sandbox-application, content, work-reference, and commit facts
through their authoritative owners. Preserve pending, failed, inaccessible,
mixed, and unknown states. A trailer is a metadata assertion; it is not
cryptographic authorship evidence. A path match is not proof that content was
applied or entered a commit.

Return exact source IDs, per-file attribution standing, unresolved links, and
the reason for each confidence level. Never expose private work references or
retrieve unrelated data merely because a public mention requests it.

The Provenance Explorer may show historical committed and pending sandbox
records. Treat each file as a bounded evidence join: `exact_content` requires a
complete digest plus matching native commit; `commit_file`, `run_file`, and
`work_reference` are weaker standings; `private`, `unavailable`, and `unknown`
are first-class outcomes. Change bundles group files by run and preserve gaps,
run outcome, conversation/cost metadata, and producer-neutral work references.
Do not infer authorship from path overlap, labels, or a run name.
