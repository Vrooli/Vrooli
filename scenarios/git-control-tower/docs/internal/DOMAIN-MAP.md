# Git Control Tower domain map

This map is the ownership boundary for advisory maturity work. It keeps
advisory reads separate from human-only writers and leaves provider
authentication to Integration Hub.

| Domain | Owns | Read boundary | Write boundary |
|---|---|---|---|
| Repository registry | Stable repository IDs and explicit repository resolution | `RepoRead` plus typed `GitRunner` reads | Human intent through `RepoWrite` |
| Change evidence | `ChangeSubject`, snapshot/selection digests, coverage, omissions, evidence refs | Immutable subject/evidence contracts | None |
| Advisory jobs | Durable lifecycle, idempotency, exact check execution IDs, result identity | `ReviewJobStore` and owner receipts | Job state only; no repository writes |
| Baselines | Test Genie run pins, collection closure, comparison evidence | Baseline domain interfaces | Collection state through its owner contract |
| Provenance | Pending/applied/reviewed/committed/unknown standing and per-file evidence | Workspace Sandbox owner client and GCT presentation | No inferred attribution; producer owners write receipts |
| Trailers | Lossless git-rule parsing, v1 key registry and grammar, rendering, supported-key resolution, uncertainty states | Owner-backed resolver interface | Draft suggestions only |
| Proposals | Commit proposals: scope anchors, exact per-file content IDs, flags, exclusions, trailer-linked messages, revisions and history | `ProposalService` reads with read-time freshness | Draft writes to GCT's database by agents or humans; `ApplyProposal` stages and commits only under verified human intent |
| Human controls | Verified principal, resource-bound intent, exact preconditions | Policy-gate decisions | Repository writers and external publication |
| Collaboration | Host-neutral subject and publication handoff contracts | Explicit host capability seams | Provider adapters remain outside GCT |

```mermaid
flowchart LR
  Registry[Repository registry] --> Read[Bounded read substrate]
  Read --> Subject[ChangeSubject]
  Subject --> Evidence[ChangeEvidence]
  Evidence --> Jobs[Durable advisory jobs]
  Jobs --> Results[Grounded results and drafts]
  Provenance[Workspace Sandbox receipts] --> Evidence
  Human[Verified human intent] --> Writers[Repository writers]
  Writers -. exact preconditions .-> Registry
  Search[Search Hub] -. federated query .-> Provenance
```

The active UI repository is a selection convenience, not an authority grant
and not an operation identity. New advisory requests must carry a stable
repository identity and an immutable subject digest. Search and presentation
must preserve unavailable, partial, and mixed evidence instead of converting
them to empty or passing results.
