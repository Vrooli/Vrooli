## Meta focus: Writing Standards

Map of which controlled-language industry standard governs which Vrooli
artifact. Named standards compress style rules the model already knows —
prefer "write this in EARS form" over hand-rolled wording rules. This skill
holds the placement map only; each owning skill carries the applied detail.

---

### 1. The placement map

| Artifact | Standard | Applied detail lives in |
|---|---|---|
| Plan procedural content (steps, validation, acceptance, boundaries) | ASD-STE100 Simplified Technical English | `implementation-plan-authoring` §3 |
| Plan flows with 3+ actors/steps | Mermaid diagram (fenced block), not prose arrow chains | `implementation-plan-authoring` §0 |
| Requirement `title`/`description` | EARS templates + RFC 2119 keywords | `requirements-traceability-steer` §5; `scenarios/test-genie/docs/reference/requirement-schema.md` |
| PRD operational-target one-liners | EARS shape; tier ↔ RFC 2119 (P0=MUST, P1=SHOULD, P2=MAY) | `prd-authoring` "Writing Standard" |
| Acceptance criteria, BAS/e2e case descriptions | Gherkin (Given/When/Then) | `e2e-testing` §4 |
| Backlog item `title`/`description` | Outcome statement + Gherkin done-condition | `swarm-manager-work-authoring` §"Writing Standard" |
| Goal / milestone `title` + intent statement | Outcome statement + Gherkin acceptance | `swarm-manager-work-authoring` §"Writing Standard: goals and milestones" |
| Test suites after feature removal | Positive validation of replacement; no tombstone/absence tests | `test` §6; `e2e-testing` §4 |
| Bug/friction reports (repro, expected/actual) | STE-100 steps; Given/When/Then shape | `report-bug` §2; `report-friction` §2 |
| Research evidence and implementation-plan Findings/Actions | STE-100 | `swarm-manager-initialize-research` |
| Skill procedural prose itself | STE-100 | `docs/agent-system/SKILL_AUTHORING.md` "Universal quality bars" |
| Workflow prompt contracts | STE-100 | `docs/agent-system/SKILL_AUTHORING.md` §"Contract skills: machine-invoked workflow prompts" |
| Procedural docs (operations/, guides/, runbooks) | STE-100 | this skill (no dedicated doc-style skill yet) |
| Operator-facing decision text / rationale | Plain-summary-first + terms-defined-at-first-use; rationale in normal prose (NOT STE) | `docs/agent-system/SWARM_MANAGER_WORK.md` §"Operator legibility" |
| Decision content placement (outcome vs evidence vs prescribed route) | Three content classes; a prescribed route forecloses | `docs/agent-system/SWARM_MANAGER_WORK.md` §"Durability"; classes defined in `swarm-manager-work-authoring` |

### 2. Where the standards do NOT apply

- **Rationale and "why" prose** — design tradeoffs, risks, limitations,
  confidence language. STE-100 strips nuance; keep these in normal
  explanatory prose even inside an otherwise STE artifact.
- **Narrative/concept canon** — `VISION.md`, `docs/concepts/`,
  `docs/narrative/`, idea-workshop and vision-walk output. Divergent and
  persuasive writing needs full English.
- **Marketing content** — owned by the brand voice canon
  (`docs/marketing/STRATEGY.md`), not by these standards.
- **Code comments** — follow the surrounding code's conventions; out of
  scope for controlled language.

### 3. The shared core (when no specific standard applies)

For any agent-facing procedural text without a named owner above:

- Follow the STE-100 procedural-prose rules in
  `docs/agent-system/SKILL_AUTHORING.md` §"Universal quality bars" — cite
  them, do not restate them.
- Replace decision-hiding words with the specific behavior meant. The
  canonical word list lives in `docs/agent-system/SKILL_AUTHORING.md`
  §"Universal quality bars" — cite it, do not copy it.
- Prohibitions are claims about observable responses, not absences.

---

### 4. Boundaries

This skill is a routing map: read it to find the owning skill, then follow
that skill. Do not restate or extend per-artifact rules here — extend the
owning skill and update the map row. No known operational edge cases for
standard usage.
