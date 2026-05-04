# Research Packet: git-control-tower

## Uniqueness Check (repo-local)

Command:
`rg -l "git-control-tower" scenarios/`

Findings:
- References exist across a few scenarios/docs, but there is no other dedicated scenario providing a structured “git control plane” API+CLI+UI.
- A legacy `scenarios/git-control-tower/` implementation previously existed and was intentionally moved out of the repo to `/tmp` to restart from a modern template.

## Related Scenarios / Resources

- `scenarios/prd-control-tower/`: PRD lifecycle enforcement + operational target/requirements patterns to mirror.
- `scenarios/scenario-completeness-scoring/`: Objective completeness scoring used to guide future iterations.
- `scenarios/scenario-auditor/`: PRD structure validation and scenario hygiene checks.

## External References (Domain Baselines)

- Git porcelain commands and exit codes (status/diff/add/commit/branch).
- Conventional Commits specification for commit message validation.
- Common safety patterns for automation around git (explicit allowlists, repo-root path validation, “dry-run” preview before mutation).

