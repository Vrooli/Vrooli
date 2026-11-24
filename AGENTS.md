# Working in Vrooli

Deliver the operator’s requested outcome at the stated scale. Build on prior work when it helps. Skills carry judgment, governed programs run repeatable workflows, and scenarios own durable state and supported operations.

How agents are set up is defined by the [agent rules](docs/agent-system/RULES.md): one frame, ten rules and four bright lines. Where any other doc or skill disagrees with them, the rules win; report the conflict as debt.

Preserve the goal when a route fails. Switch to another supported route while keeping the outcome and acceptance criteria. If part of the work is blocked, advance what you can and state exactly what remains and why.

## Find a better path when it matters

Find Vrooli knowledge, skills, and reusable programs with `search-hub query "<question or task>"` when a better route or prior result could change the work.

```bash
search-hub query "Design a UX for a fitness tracker and find a workflow for building a clickable prototype"
```

## Use the method that fits

Follow a method named by the operator or brief. Reuse guidance already in context. Read a skill with `prompt-manager skill read <name>` or open its canonical source when the service is unavailable.

- For commissioned research, start with the brief or mission. See the [research entry point](docs/internal/research-experimentation/README.md).
- For an existing plan, use `implementation-plan-execution`.
- Ambitious or multi-hour outcomes go to a research topic, not a delivery team. For a long goal inside one session, use `harness-goal-authoring`. Give each delegate a coherent outcome, relevant constraints, and acceptance evidence. Carry the operator’s verbatim stance into sustained handoffs and re-read the charter after compaction.
- To check an agent, team or workspace you manage, use `loop-audit`.
- For new scenarios, use `ecosystem-fit`. For substantial UX work, use `experiential-ui-design`. Use `scenario-work-ladder` when the broken contract or evidence layer is unclear. Start clear, localized fixes with the relevant code and evidence.
- For publishing or deploying a scenario, use `deployment-coordinator`.

## Work in the shared repository

- Preserve shared work. Inspect overlapping changes and coordinate with active writers. A dirty tree alone does not require approval.
- Use [Scenario Dependency Analyzer](docs/package-governance.md) for dependency changes.
- Launch and manage scenarios through Make targets or [`vrooli scenario`](docs/configuration/scenarios.md). For Agent Manager maintenance, use the [restart and recovery procedure](scenarios/agent-manager/docs/runner-transcript-recovery.md). Host detection and repair belong in the control plane.
- Verify in proportion to the change. Use [Test Genie](docs/TESTING.md) for scenario suites and the reported wait command for server-owned runs. Write tests against expected behavior.
- For a build failure involving a generated symbol, run the owning verifier first. For protobufs, use `make -C packages/proto verify-committed-gen`. Regenerate through the owner if drift is confirmed.
- Follow the supported operation’s [authorization rules](docs/concepts/AUTHORIZATION.md). Use existing authority where it applies, preserve human-only decisions, and evaluate safety conditions independently.
- Edit component library drafts only. Open one with `react-component-library components draft-begin <asset>`.

## Leave work others can build on

After nontrivial work, leave a brief searchable record of the outcome and the evidence or lesson worth carrying forward. Use `vrooli-memory journal note --kind work-record` unless the active workflow already retains equivalent searchable evidence. Use `report-bug` for consequential outside-scope defects. Improve Vrooli when commissioned or when delivery depends on the fix.

```bash
vrooli-memory journal note --kind work-record \
  --trigger "What prompted the work" --approach "What you did" \
  --evidence "Verification or artifact location" --outcome "What changed" \
  "Short searchable summary of the result or lesson."
```

Use the [documentation index](docs/README.md) when you need an operating reference.
