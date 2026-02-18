# Enhance: Synthesize Refined Plan

## Purpose

Synthesize clarifications, accepted suggestions, research findings, and archive materials into a refined, actionable plan. Prepare staging artifacts alongside the plan so the process step can incorporate them into the target scenario.

## Input Context

**Required reading:** `prompt-manager skill read swarm-manager-backlog-tools` — folder structure, artifact schemas, and CLI commands for reading/writing backlog files.

**Optional reading:** `prompt-manager skill read swarm-manager-process-idea` — understanding how the process step consumes enhance outputs helps produce better staging materials.

## Scope

**In scope:**
- Reading and synthesizing all item context (spec, clarify, suggest, research, archive, user files)
- Producing `enhance/summary.md` — the refined plan
- Producing staging artifacts in `enhance/` — prepared materials for the process step to incorporate into the scenario (e.g., PRD context brief, requirements context, doc outlines)
- Updating `spec.json` with enhanced description if significantly changed

**Out of scope:**
- Generating questions (see `swarm-manager-clarify-idea`)
- Proposing improvements (see `swarm-manager-suggest-idea`)
- Processing/implementing the idea (see `swarm-manager-process-idea`)
- Modifying `archive/` — it contains user-provided materials and must not be altered by agents

## Output Requirements

**Primary outputs**:
1. `enhance/summary.md` — Refined plan document (the source of truth for what to implement)
2. `enhance/prd-context.md` — Synthesized PRD context brief ready for `prd-control-tower` consumption
3. `enhance/requirements-context.md` — Requirements context ready for `prd-control-tower requirements generate`
4. `spec.json` — Updated with enhanced description (if significant changes)

**Conditional outputs** (when relevant materials exist in archive or research):
5. `enhance/doc-outlines.md` — Outlines for scenario documentation (README sections, RESEARCH findings, PROBLEMS entries)

Write all outputs via CLI:
```bash
swarm-manager backlog file-upload --kind <kind> --name <name> --path <relative-path> --content '<content>'
```

### The Staging Role of `enhance/`

The `enhance/` folder serves dual purposes:
1. **`summary.md`** is the plan — it describes *what* to implement and *why*
2. **Staging artifacts** are prepared materials — they contain *content ready to be incorporated* into the scenario by the process step

The process step reads `enhance/` staging artifacts when available, falling back to raw `archive/` materials if enhance hasn't run. By producing well-structured staging materials, the enhance step ensures the process step doesn't have to re-synthesize from raw sources.

## Success Criteria

- [ ] All available context read (spec, clarify, suggest, research, archive, user files)
- [ ] All answered clarifying questions incorporated as definitive statements
- [ ] All accepted suggestions integrated into the plan
- [ ] Rejected suggestions acknowledged (not included in plan)
- [ ] Archive materials reviewed and relevant content incorporated into staging artifacts
- [ ] Scope clearly bounded (included, excluded, deferred)
- [ ] `summary.md` is implementation-ready with no remaining ambiguities
- [ ] `prd-context.md` produced with all sources synthesized
- [ ] `requirements-context.md` produced if requirements information exists
- [ ] All outputs uploaded via CLI and verified
- [ ] Readiness gate evaluated honestly

## Instructions

You are creating the enhanced specification for a Swarm Manager backlog item. Your goal is to synthesize all gathered information into a clear, implementation-ready plan and prepare staging materials that the process step will incorporate into the scenario.

**Context from spec.json:**
- Kind: {{ITEM_KIND}}
- Title: {{ITEM_TITLE}}
- Description: {{ITEM_DESCRIPTION}}
- Priority: {{ITEM_PRIORITY}}
- Tags: {{ITEM_TAGS}}

### Processing Steps

1. **Read all available context**

   ```bash
   swarm-manager backlog get --kind {{ITEM_KIND}} --name {{ITEM_NAME}}
   swarm-manager backlog files --kind {{ITEM_KIND}} --name {{ITEM_NAME}}
   ```

   Read artifacts in refinement order — start with the most refined source:

   - **`enhance/` (prior run)** — if `enhance/summary.md` and staging artifacts exist from a prior run, read them first. These are your **foundation** — they represent the previously synthesized state of all inputs. Your job is to update and improve this foundation, not discard it.
   - `spec.json` — item metadata and description (superseded by prior enhance/ if it exists)
   - `clarify/questions.json` — answered questions (definitive authority). Pay special attention to answers added **since the last enhance run** — these are new input that must be layered onto the foundation.
   - `suggest/suggestions.json` — accepted/rejected suggestions. Same as above: look for decisions made since the last enhance run.
   - `research/summary.md` — feasibility findings (advisory)
   - `archive/` — user-provided materials (requirements, docs, designs, prior scenario artifacts). Only use for content not already captured in the prior enhance/ output.
   - User-uploaded files in the item root

   > **Re-run principle:** Each enhance run builds on the last. If `enhance/` already exists, treat it as the accumulated knowledge base. Layer new clarify answers, new suggestion decisions, and new research on top. Don't start from scratch — refine what's there.

2. **Resolve conflicts using the decision hierarchy**

   When sources disagree, apply this precedence:

   ```
   Conflict detected between sources
     → Is one source an answered question?
        → Yes → Answered question wins (definitive)
     → Is one source an accepted suggestion and the other the original spec?
        → Yes → Accepted suggestion wins (user approved it)
     → Is one source enhance/summary.md from a prior run and the other spec.json?
        → Yes → enhance/summary.md wins (synthesized later with more context)
     → Is one source research and the other a user decision?
        → Yes → User decision wins (research is advisory)
     → Neither is clearly authoritative?
        → Flag as an unresolved ambiguity in the readiness gate
   ```

3. **Synthesize the plan (`enhance/summary.md`)**

   Transform gathered context into definitive statements:
   - Convert Q&A pairs into specifications: "Q: What auth method? A: OAuth" → "Uses OAuth 2.0 for authentication"
   - Integrate each accepted suggestion, explaining how it changes the approach
   - Acknowledge rejected suggestions (note what was rejected and why)
   - Define explicit scope boundaries
   - Capture technical approach, integration points, dependencies, and risks

4. **Prepare staging artifacts**

   These are the materials the process step will incorporate into the scenario. The goal: make it so the process step can use these directly without re-reading raw sources.

   #### 4a. `enhance/prd-context.md` — PRD context brief

   Synthesize a free-form brief suitable for `prd-control-tower prd generate`. Map sources to PRD sections:

   | Source | Maps to PRD Section |
   |--------|-------------------|
   | Summary overview & refined scope | Overview, value proposition, target users |
   | Summary success criteria | P0 operational targets (core capabilities) |
   | Accepted high-impact suggestions | P1 targets (important enhancements) |
   | Accepted medium/low-impact suggestions | P2 targets (nice-to-have polish) |
   | Summary implementation notes | Tech direction snapshot |
   | Research dependency analysis | Dependencies & launch plan |
   | Archive: requirements docs | Extract and refine into P0/P1/P2 targets |
   | Archive: design mockups / wireframes | UX & branding section |
   | Archive: API specs / schemas | Tech direction snapshot |
   | Archive: prior PRD | Update and refine (don't copy verbatim — incorporate new decisions) |

   > **Decision hierarchy reminder:** Answered questions are definitive. Accepted suggestions must appear. Rejected suggestions must NOT appear. Research is advisory.

   #### 4b. `enhance/requirements-context.md` — Requirements context

   If archive materials contain requirements documents, testing strategies, or technical constraints, synthesize them into a context file suitable for `prd-control-tower requirements generate`:
   - Validation approach (testing strategy)
   - Technical constraints and dependency relationships
   - Specific operational target details that need elaboration

   Skip this file if no requirements-related source material exists.

   #### 4c. `enhance/doc-outlines.md` — Documentation outlines (conditional)

   If archive or research materials contain documentation, research findings, or known issues worth preserving, prepare outlines:
   - README sections to include
   - RESEARCH.md findings to carry forward
   - PROBLEMS.md entries for known risks or deferred issues
   - PROGRESS.md initial entry

   Skip this file if no documentation-related source material exists.

5. **Evaluate readiness gate**

   Before declaring the plan ready, check:

   | Gate | Check | If Failed |
   |------|-------|-----------|
   | Critical questions answered | All `importance: critical` questions in clarify have non-empty answers | **Block** — plan is not ready |
   | Scope defined | Summary has explicit included/excluded/deferred sections | **Block** — add scope boundaries |
   | No unresolved conflicts | No sources contradict each other without resolution | **Block** — resolve or flag |
   | Technical approach viable | Research (if available) doesn't flag showstoppers | **Block** — address feasibility |
   | Important questions answered | All `importance: important` questions have answers | **Warn** — plan can proceed but flag gaps |
   | Archive reviewed | All archive files have been read and relevant content incorporated | **Warn** — may miss user intent |

6. **Write all outputs**

   ```bash
   # Primary plan
   swarm-manager backlog file-upload --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --path enhance/summary.md --content '<content>'

   # Staging artifacts
   swarm-manager backlog file-upload --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --path enhance/prd-context.md --content '<content>'
   swarm-manager backlog file-upload --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --path enhance/requirements-context.md --content '<content>'  # if applicable
   swarm-manager backlog file-upload --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --path enhance/doc-outlines.md --content '<content>'  # if applicable

   # Update spec.json if enhanced description differs significantly
   swarm-manager backlog update --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --data '{"enhanced_description": "...", "enhanced_at": "ISO-8601"}'
   ```

7. **Verify**

   ```bash
   swarm-manager backlog files --kind {{ITEM_KIND}} --name {{ITEM_NAME}}
   swarm-manager backlog file-get --kind {{ITEM_KIND}} --name {{ITEM_NAME}} --path enhance/summary.md
   ```

   Confirm all expected files exist in `enhance/` and the summary is coherent.

### Output Format for `enhance/summary.md`

```markdown
# Enhanced Plan: {{ITEM_TITLE}}

## Overview
[2-3 sentence enhanced description that incorporates clarifications]

## Clarifications Applied

| Question | Answer | Impact |
|----------|--------|--------|
| [Question] | [Answer] | [How it affects the plan] |

## Suggestions Integrated

### Accepted
| Suggestion | Integration |
|------------|-------------|
| [Suggestion] | [How it's incorporated] |

### Not Accepted
| Suggestion | Reason |
|------------|--------|
| [Suggestion] | [Why not included] |

## Refined Scope

### Included (Must Have)
- [Feature/capability 1]
- [Feature/capability 2]

### Included (Should Have)
- [Feature/capability 3]

### Excluded (Out of Scope)
- [Not included 1] - [Reason]
- [Not included 2] - [Reason]

### Deferred (Future)
- [Future feature 1] - Target: v2
- [Future feature 2] - Target: when needed

## Implementation Notes

### Technical Approach
[Key technical decisions and patterns]

### Integration Points
- [Scenario/Resource]: [How it integrates]

### Dependencies
- [Dependency]: [Why needed]

### Risks & Mitigations
| Risk | Mitigation |
|------|------------|
| [Risk] | [How to handle] |

## Success Criteria
- [ ] [Measurable criterion 1]
- [ ] [Measurable criterion 2]
- [ ] [Measurable criterion 3]

## Readiness Gate
- [ ] All critical questions answered
- [ ] Scope clearly defined
- [ ] Technical approach validated
- [ ] Dependencies available
- [ ] Success criteria measurable
- [ ] Archive materials incorporated into staging artifacts

**Ready for processing:** [Yes / No — if no, explain what's blocking]

## Staging Artifacts Produced
- `enhance/prd-context.md` — [brief description of what it covers]
- `enhance/requirements-context.md` — [brief description, or "Not produced (no requirements source material)"]
- `enhance/doc-outlines.md` — [brief description, or "Not produced (no documentation source material)"]
```

## Quality Guidelines

**Good enhancement:**
- Transforms Q&A into definitive statements
- Shows clear decision trail
- Removes ambiguity
- Produces staging artifacts that the process step can use directly
- Incorporates archive materials without losing user intent
- Scope is clear and bounded
- Readiness gate is evaluated honestly

**Poor enhancement:**
- Leaves questions unanswered
- Ignores suggestions without explanation
- Vague scope boundaries
- Skips archive materials
- Produces summary.md but no staging artifacts
- Declares "ready" despite unresolved blockers

## Anti-Patterns

- **Don't** ignore unanswered questions — flag them as blockers if critical
- **Don't** silently drop rejected suggestions — acknowledge them
- **Don't** add new scope not from clarifications/suggestions
- **Don't** create implementation details that contradict answers
- **Don't** leave the plan in an ambiguous state
- **Don't** modify `archive/` — it's user-provided and sacred
- **Don't** copy archive materials verbatim into staging — synthesize and refine them
- **Don't** write files directly to disk — use the backlog CLI

## Troubleshooting & Edge Cases

| Problem | Solution |
|---------|----------|
| `clarify/questions.json` doesn't exist | Enhance can still run — use spec.json and archive as primary context |
| `suggest/suggestions.json` doesn't exist | Enhance can still run — fewer inputs to synthesize |
| Archive contains a complete PRD from a prior scenario | Don't copy it verbatim — synthesize a new `prd-context.md` that incorporates answered questions and accepted suggestions |
| Archive is empty | Normal — produce staging artifacts from spec + clarify + suggest + research |
| Research flags a showstopper | Mark readiness gate as blocked and explain the issue in summary.md |
| Answered questions conflict with accepted suggestions | Answered questions win (decision hierarchy). Note the conflict in the summary. |
| Prior `enhance/summary.md` exists | This is a re-run. Read the prior version, then generate a fresh synthesis from all current context. The old summary is superseded. |
