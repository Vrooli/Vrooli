---
name: "program-discovery"
description: "Find an owner skill and exact Program Runtime contract before repeating multi-step CLI work."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["search"]
  status: "active"
  revision: 5
  createdAt: "2026-11-23T00:00:00Z"
  updatedAt: "2026-11-23T23:36:00Z"
  requires: {scenarios: ["search-hub", "prompt-manager", "program-runtime"], commands: ["search-hub query", "prompt-manager skill read", "program-runtime library search", "program-runtime library run", "program-runtime bindings describe"]}
  origin: {kind: "authored"}
---
## Search focus: Program Discovery
Before repeating a multi-step scenario workflow, find its owner skill and an exact Program Runtime contract.
1. Find the owner skill with `program-runtime library run search-hub.research --input query="<task>" --input 'types=["skill"]' --input include_skill=true`; add `--input include_discover=true` when the repeated path includes Prompt Manager discover. Read `signals.skill_lookup.content` and honor its status and truncation flag.
2. Search `program-runtime library search "<task>"`; inspect candidates in result order with `program-runtime library run program-runtime.prepare-operation --input name=<scenario>.<program>` to read inputs, called bindings and owner skill.
3. Select the first candidate whose inputs and called bindings cover every repeated operation. Reject partial matches; check unclear binding effects with `program-runtime bindings describe <id>`.
4. Run an exact match with `program-runtime library run <scenario>.<program> --input k=v`; branch on envelope status. In agent sessions, leave provenance and caller fields unset so the harness supplies verified identity.
If no exact match exists, use the owner's documented CLI and name the uncovered operation. Discovery is read-only; check effects before execution, then report owner skill, program/status, or gap. Do not dump the full catalog at startup.
### Troubleshooting & Edge Cases
If Search Hub fails, read the known owner skill and search the library directly. If the skill is unavailable, use owner docs and CLI help. If library search fails, use the owner CLI and report the discovery gap.
