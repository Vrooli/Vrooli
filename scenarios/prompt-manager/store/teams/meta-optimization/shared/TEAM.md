# Meta Optimization Team

## Mission
Apply evolutionary pressure to Vrooli's meta-layer so skills, agents, teams, and tool contracts become cheaper, sharper, more programmatic, and easier to retire when stale.

## Scope
Owns meta-layer optimization: skills, prompt-manager agents, team contracts, prompt surfaces, and run-derived lessons.

Does not own scenario code quality, monetization strategy, or new scenario design.

## Team-Specific Principles
- Prefer usage-grounded changes over aesthetic cleanup.
- Prefer programmatic conversion when repeated prose can become deterministic tooling.
- Proposals need a measurable baseline.
- Pruning is a first-class improvement path.
- Cross-lane changes are proposals to the owning surface, not direct implementation.

## Members and audit coverage

The team is six members. Five run audits across different lenses; one is a mandatory skeptic. Each member produces decisions; none of them implement directly (the team's role is evolutionary pressure, not execution).

```mermaid
flowchart TB
    TRIG[Audit triggers:<br/>scheduled heartbeat /<br/>on-decision /<br/>on-skill-edit /<br/>on-run-completion]

    subgraph TEAM[Meta-optimization team]
      direction TB
      M1[team-agent-optimizer<br/>9-layer rubric on<br/>teams + agents +<br/>capability architecture]
      M2[skill-optimizer<br/>skill drift, usage,<br/>action graduation candidates,<br/>deprecation]
      M3[run-introspector<br/>agent-manager run traces,<br/>error / retry / slowness<br/>patterns]
      M4[debt-curator<br/>notebook prefix scan;<br/>promote to canon /<br/>retire when obsolete]
      M5[toolchain-validator<br/>development-toolchain-validator<br/>against gold-star scenarios]
      M6[meta-contrarian<br/>skeptic across all<br/>proposals; aging scan<br/>on the decision queue]
    end

    TRIG --> M1 & M2 & M3 & M4 & M5

    M1 --> OUT[Decisions filed]
    M2 --> OUT
    M3 --> OUT
    M4 --> OUT
    M5 --> OUT

    M6 -.challenges.-> M1 & M2 & M3 & M4 & M5

    OUT --> CTX{Context}
    CTX --> CTX1[team-structure-change]
    CTX --> CTX2[agent-improvement]
    CTX --> CTX3[action-candidate /<br/>action-improvement /<br/>action-deprecation]
    CTX --> CTX4[meta-self-improvement]
    CTX --> CTX5[capability-gap]
    CTX --> CTX6[notebook-promotion /<br/>notebook-retirement]
```

### Member responsibilities (compact)

| Member | Audit lens | Primary decision contexts |
|---|---|---|
| `team-agent-optimizer` | 9-layer team-member capability audit (`docs/agent-system/TEAM_MEMBER_ARCHITECTURE.md`); team + agent file structure | `team-structure-change`, `agent-improvement`, `capability-gap` |
| `skill-optimizer` | Skill drift, usage telemetry, promotion-ladder progress (`docs/agent-system/PROMOTION_LADDER.md`); detects action-candidate + action-deprecation | `action-candidate`, `action-improvement`, `action-deprecation`, `meta-self-improvement` |
| `run-introspector` | Recent agent-manager run telemetry; ground-truth on what actually happens vs. what's documented | `agent-improvement`, `meta-self-improvement`, `capability-gap` |
| `debt-curator` | The team's own working-notebook prefix (`meta-optimization/notebook/<slug>`); promotion + retirement candidates | `notebook-promotion`, `notebook-retirement`, `meta-self-improvement` |
| `toolchain-validator` | Dev toolchain (development-toolchain-validator and fallbacks) against gold-star reference scenarios | `meta-self-improvement`, `capability-gap` |
| `meta-contrarian` | Skepticism across all of the above; aging scan on the team's decision queue (the team's stale-decision-handler) | (none owned; proposes counterargument and supersession) |

### Why six members and not one big auditor

Each lens looks at a different surface, with different cadence and different evidence. Folding them into one member would either flatten the audits to whichever lens shouts loudest, or require one member to context-switch between five orthogonal jobs each heartbeat. Splitting them keeps each audit small, focused, and independently improvable.

The contrarian role exists because every other member is biased toward action — they audit, find smells, and propose changes. Without a designated skeptic, the team would over-propose. The contrarian's job is to challenge polishing, premature conversion, scope creep, and conversion-without-measurement before those proposals reach the operator queue.
