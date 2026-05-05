# Documentation Progress

## Rewrite Status

Completed across the current rewrite passes:

- rewrote the docs hub at `path:docs/README.md`
- added `path:docs/manifest.json`
- added canonical `path:docs/QUICKSTART.md`
- added `path:docs/concepts/ARCHITECTURE.md`
- added `path:docs/concepts/GLOSSARY.md`
- added the strategy layer and moved project framing docs under `path:docs/strategy/`
- rewrote the top-level contributor, testing, business, and production docs
- rebuilt `path:docs/deployment/` around clear current-vs-reference-vs-planning boundaries
- tightened `path:docs/reference/` against the live Makefile and CLI help surfaces
- tightened `path:docs/scenarios/` into a smaller cross-scenario canon layer
- tightened `path:docs/resources/` into a smaller cross-resource canon layer
- fixed moved-path seams and link rot across the major docs sections

## Structural Reorganization Status

Completed:

- added `path:docs/guides/`, `path:docs/operations/`, and `path:docs/strategy/`
- moved active DevOps guidance into guides, operations, reference, and deployment reference locations
- relocated scenario-specific project docs into the owning scenario docs where appropriate
- updated `path:docs/README.md` and `path:docs/manifest.json` to match the new tree
- removed the temporary archive layer after updating or deleting its callers
- removed the now-empty top-level `path:docs/devops/` subtree

## Current State

The docs tree now has a coherent canonical structure and the major section rewrites are in place.

What remains is mostly:

- specialized leaf cleanup
- continued content tightening where pages are intentionally conservative
- future maintenance to keep the canonical layer aligned with evolving CLI and platform behavior
