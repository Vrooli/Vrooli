# Action Entity and Memory Promotion RFC

Status: proposed.

This RFC records the documentation baseline for adding Actions to prompt-manager and for teaching agents how raw notebook knowledge should graduate into durable system forms.

## Problem

Prompt-manager currently has three primary entities:

- Skills: reusable prose guidance
- Agents: persistent identities that perform work
- Teams: groups of agents organized around a mission

This works well for judgment-heavy behavior, but some skills eventually become thin wrappers over deterministic CLI commands. Keeping those operations as prose has costs:

- Agents spend tokens rediscovering command syntax.
- Repeated operations are harder to validate.
- Operational details drift between skills, prompts, and CLI help.
- The system lacks a first-class discoverable object for "run this exact known operation."

Separately, team notebooks and knowledge logs need a consistent promotion model. Raw observations should not all become skills. Some are durable facts, some are missing implementation, some are executable operations, and some should remain unverified notes.

## Proposal

Add a proposed Action entity to the prompt-manager ontology.

```text
Truth lives in the Plan of Record.
Judgment lives in Skills.
Execution lives in Actions.
Implementation lives in CLIs.
Unbuilt work lives in the Backlog.
Raw learning starts in Notebooks.
```

An Action is a typed, discoverable wrapper over exactly one Vrooli-controlled CLI command. It declares input schema, output schema, permissions, examples, ownership, and validation. It does not implement business logic.

## Non-Goals

- Do not make Actions arbitrary shell scripts.
- Do not encode branching or routing in Action definitions.
- Do not replace skills with Actions.
- Do not rename the existing capability-matching model.
- Do not wrap raw external tools directly.

## Naming

Use "Action" for the executable wrapper. Avoid "Capability" because prompt-manager already uses capabilities for agent and skill requirement matching.

## Command Boundary

Actions may wrap:

- `vrooli ...`
- `prompt-manager ...`
- scenario CLIs managed by Vrooli lifecycle
- Vrooli-owned resource CLIs or resource commands exposed through Vrooli wrappers

Actions should not wrap raw `git`, `docker`, `psql`, `curl`, `grep`, scripts, command pipelines, or shell conditionals. If those tools are required, the owning scenario/resource/project should provide a controlled CLI command first.

## Promotion Classifier

Use this classifier when reviewing notebooks, knowledge logs, run lessons, and other accumulated observations:

```text
If it says what is true -> Plan of Record.
If it says how to decide -> Skill.
If it says what to run -> Action.
If it says how it works -> CLI implementation.
If it says what is missing -> Backlog/capability-gap.
If it is unverified or one-off -> Notebook.
```

## Documentation Changes in This RFC

This RFC is supported by:

- [DOC: docs/concepts/ACTIONS.md] - Action entity concept and intended contract
- [DOC: docs/concepts/MEMORY-PROMOTION.md] - notebook-to-durable-memory promotion model
- [DOC: docs/concepts/SWARM-MODEL.md] - updated entity relationships
- [DOC: docs/concepts/CAPABILITY-MATCHING.md] - naming boundary between capabilities and Actions
- [DOC: docs/concepts/GRAPH.md] - future graph edge model

## Implementation Plan Follow-Ups

After this documentation baseline is accepted, create two implementation plans:

1. Action entity implementation:
   - schema and storage
   - API
   - CLI
   - search/discover integration
   - UI
   - validation and execution safety

2. Action adoption and memory promotion:
   - meta-optimization prompts and skills
   - decision contexts
   - notebook classifier adoption
   - seed Actions
   - measurement loop

## Acceptance Criteria

- New agents can understand where Actions fit without this conversation.
- Existing docs distinguish current entities from proposed Action work.
- The word "capability" remains reserved for matching and requirements.
- The memory promotion classifier is documented in a canonical place.
- Future implementation plans can reference this RFC instead of restating the ontology.
