# Documentation Progress

## Rewrite Status

Completed across the current rewrite passes:

- rewrote the docs hub at `docs/README.md`
- added `docs/manifest.json`
- added canonical `docs/QUICKSTART.md`
- added `docs/concepts/ARCHITECTURE.md`
- added `docs/concepts/GLOSSARY.md`
- added the strategy layer and moved project framing docs under `docs/strategy/`
- rewrote the top-level contributor, testing, business, and production docs
- rebuilt `docs/deployment/` around clear current-vs-reference-vs-planning boundaries
- tightened `docs/reference/` against the live Makefile and CLI help surfaces
- tightened `docs/scenarios/` into a smaller cross-scenario canon layer
- tightened `docs/resources/` into a smaller cross-resource canon layer
- fixed moved-path seams and link rot across the major docs sections

## Structural Reorganization Status

Completed:

- added `docs/guides/`, `docs/operations/`, and `docs/strategy/`
- moved active DevOps guidance into guides, operations, reference, and deployment reference locations
- relocated scenario-specific project docs into the owning scenario docs where appropriate
- updated `docs/README.md` and `docs/manifest.json` to match the new tree
- removed the temporary archive layer after updating or deleting its callers
- removed the now-empty top-level `docs/devops/` subtree

## Current State

The docs tree now has a coherent canonical structure and the major section rewrites are in place.

What remains is mostly:

- specialized leaf cleanup
- continued content tightening where pages are intentionally conservative
- future maintenance to keep the canonical layer aligned with evolving CLI and platform behavior
