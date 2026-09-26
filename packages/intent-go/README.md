# intent-go

`intent-go` is the shared extraction contract for Vrooli intent alignment:
PRD operational targets, requirements, normalized validation references, and
neutral ladder findings.

The source-of-truth model is `docs/reference/intent-alignment.md`. This package
implements the frozen `CapabilityClaim` contract from that doctrine and is the
only shared home for PRD and requirements parsing. Cartographer keeps ownership
of domain derivation and adapts these claims to `DerivedDomainMap`.

Core APIs:

- `FilePRDExtractor.ExtractPRDClaims(root)` emits outcome claims from `PRD.md`.
- `FileRequirementsExtractor.ExtractRequirementClaims(root)` emits requirement
  claims from `requirements/**/module.json`.
- `NormalizeRef(raw, validationType)` handles `#fragment`, `::Member`, legacy
  `file.go:Member`, globs, doc refs, and manual validation refs.
- `CheckPRDRefResolves`, `CheckOrphanOutcome`, and `CheckRefExists` return
  neutral `Finding` values for producers to map into their native finding
  envelope.
