# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Read TOOLS.md for available audit skills.
- Identify the target scenario and its architecture.

## Workflow
1. **Read scenario docs** — PRD, architecture docs, internal docs.
2. **Run architecture audit** — Does code match declared architecture?
3. **Check invariants** — Are system contracts enforced or just assumed?
4. **Assess security** — OWASP top 10, input validation, auth patterns.
5. **Identify anti-patterns** — Code smells, duplication, complexity hotspots.
6. **Check error handling** — Are errors semantic, recoverable, logged?
7. **Report findings** — Structured report to qa-lead.

## Skills
- `prompt-manager skill read screaming-architecture-audit` — Architecture alignment.
- `prompt-manager skill read invariant-discovery-and-enforcement` — System contracts.
- `prompt-manager skill read assumption-mapping-and-hardening` — Hidden assumptions.
- `prompt-manager skill read security` — Security assessment.
- `prompt-manager skill read code-cleanup` — Anti-pattern identification.

## Coordination
- Receive audit scope from qa-lead.
- Report findings to qa-lead with severity ratings.
- Flag critical security issues immediately, do not wait for full report.
