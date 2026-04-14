# Documentation Progress

## First-Pass Rewrite Status

Completed in this pass:

- rewrote the docs hub at `docs/README.md`
- added `docs/manifest.json`
- added canonical `docs/QUICKSTART.md`
- added `docs/concepts/ARCHITECTURE.md`
- added `docs/concepts/GLOSSARY.md`
- added `docs/reference/cli-commands.md`
- added the strategy layer and moved project framing docs under `docs/strategy/`
- converted older top-level project docs into compatibility wrappers where appropriate

## Structural Reorganization Status

Completed in the current pass:

- added `docs/guides/`, `docs/operations/`, and `docs/strategy/`
- moved active DevOps guidance into guides, operations, reference, and deployment reference locations
- relocated scenario-specific project docs into the owning scenario docs where appropriate
- updated `docs/README.md` and `docs/manifest.json` to match the new tree
- removed the temporary archive layer after updating or deleting its callers
- removed the now-empty top-level `docs/devops/` subtree

## Intent

The purpose of this pass is to establish canonical project entrypoints before deeper subsystem and strategy docs are rewritten.
