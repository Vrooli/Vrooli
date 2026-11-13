# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario research-desk`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Give the operator one place, usable from a phone, to configure, launch, watch, steer and review supervised research commissions, while the research itself keeps running exactly as it does today when this scenario is stopped, broken or absent. The desk is a launcher, a lens and a courier; it is never in the loop.
- **Primary users/verticals**: The Vrooli operator who commissions long supervised research windows (Codex supervisor sessions in Web Console driving lower-powered owner agents over `~/.vrooli/research` topics); secondarily, agents that build commission packets or run operator review sessions through the research skills this scenario owns.
- **Deployment surfaces**: Go API (read-mostly projection of the research home plus three narrow writers), `research-desk` CLI with the same operations, phone-first React UI on the tunnel hostname, and the scenario-owned research skills (`research-coordination`, `adaptive-research`, `local-experimentation`, `research-review`).
- **Value promise**: Launching a commission drops from about five minutes of agent conversation to one action with the best-known setup every time; the operator can answer 'what research is moving, when was this topic last active, which supervisors are running' at a glance; decisions reach the operator and answers get back without a single message entering a supervisor's conversation; and nothing about research becomes more fragile.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | One-action launch | Choose topics, allowance and emphasis and receive a complete packet (charter, stance message, goal kernel) with size gates, launched into a Web Console Codex session automatically or shown as two messages to paste.
- [x] OT-P0-002 | Clean-start verdict | Before launch, state per selected topic whether an earlier commission touches it, its observed state (live, paused, limited, stopped, ended, prepared, unknown), and what this launch does about it; refuse to double-supervise a live topic unless the operator explicitly runs alongside.
- [x] OT-P0-003 | Discard and relaunch | A bad start is stopped (never deleted), verified inactive, marked discarded with the operator's reason, and relaunched as a new commission rebuilt from the same packet inputs, superseding the old one.
- [x] OT-P0-004 | Activity projection | Show active research, last activity per topic, running supervisors, the topics each supervises and commissions overlapping in time, derived only from research-home files and harness session records, with unknown where a value cannot be derived.
- [x] OT-P0-005 | File-only steering | Operator direction is appended verbatim to a desk-owned operator file the supervisor already re-reads at unit start; nothing is sent into a supervisor conversation.
- [x] OT-P0-006 | Decision courier | Needs-you items written by supervisors become one feed, are delivered to Notification Hub as decisions or notices once the operator turns the courier on (on by default only for desk-launched commissions), and answers are read back from the Hub into the operator file, so a desk outage loses no answer; the UI answers directly when the Hub is down.
- [x] OT-P0-007 | Focus measurement | Measure and display each supervisor and owner role's read set, as listed in the research-coordination skill's limits.json, in bytes and estimated tokens, and flag sets over their ceilings, an oversized charter and appended amendments.
- [x] OT-P0-008 | Review support | A research-review skill and desk support for an operator review session over selected topics: evidence of accomplishment, tidy-up proposals, verdicts recorded verbatim and missions realigned only by explicit operator action.
- [x] OT-P0-009 | Scenario-owned research skills | Research skills move into scenarios/research-desk/skills through the supported ownership route with prompt-manager skill read and the direct-file fallback still working, and the supervisor procedure gains written route discipline for failing optional routes.
- [x] OT-P0-010 | Detached by construction | Stopping the desk mid-commission changes nothing for supervisors or owners; every packet launches by hand from its folder; commissions launched by hand are recognised.
- [x] OT-P0-011 | Phone-first interface | API and CLI first, then a UI designed for phone width on the tunnel hostname that replaces the generated placeholder completely.
- [x] OT-P0-012 | Control without chat | Pause, resume, wind down, stop and discard a commission from the desk without writing a message in its conversation; wind down is an operator-file entry, the others act on the session only on an operator tap and are recorded.
- [x] OT-P0-013 | Since last look | Per commission and per topic, show what changed since the operator last looked: new ledger text, changed checkpoints, showcase lines and workstream status changes.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Commission timeline | A time-axis view of commissions and owner threads that makes overlap, stalls and idle topics visible over the last days.
- [ ] OT-P1-002 | Stall notice | When a commission the desk can map shows no rollout writes past its stall threshold, raise one informational Notification Hub item (deduplicated by commission).
- [ ] OT-P1-003 | Packet diff | Show what changed between a new packet and the previous packet for the same topics (procedure version, cards, operator words) before launch.
- [ ] OT-P1-004 | Focus trend | Track start-up read-set size per commission over successive units so growth in charter, state or mission files is visible before it hurts.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Other harness supervisors | Recognise and project supervisors run in Claude Code or other harnesses whose session records differ from Codex homes.
- [ ] OT-P2-002 | Commission cost accounting | Per-commission token and time accounting from rollouts, compared with what each commission landed.
- [ ] OT-P2-003 | Second research home | Project a research home on another fleet node read-only through the same file contract.

## 🧱 Tech Direction Snapshot

- Preferred stacks / frameworks: Go API from the default scenario template (proto-first contracts, one service per domain), cli-core CLI generated from the same descriptors, React + Vite UI from the React Component Library on the phone-first archetype.
- Data + storage expectations: No database. The research home (`~/.vrooli/research`) and Web Console Codex session homes are authoritative; the desk writes only the files assigned to it in `docs/reference/FILE-CONTRACT.md` §1 and keeps an in-memory cache keyed by path, size and mtime that can be dropped and rebuilt. The template SQLite store and notes example are removed.
- Integration strategy: Read files directly; packets are rendered by the skill's stdlib `packet.py` (one renderer for the desk and for agents) and missions parsed by `mission_check.py`, both as subprocesses; use Web Console's HTTP API with an idempotency key for session create and terminal input, only on an operator tap (launch and control); call Notification Hub directly (`Ask`, `Send`, `ListAsks`, `Withdraw`) for decisions.
- Non-goals / guardrails: The desk never writes supervisor-owned files, never sends a message into a running supervisor conversation, never becomes a required step in any supervisor or owner procedure, does not score or alarm research inside Agent Manager, and does not schedule or orchestrate owners.

## 🤝 Dependencies & Launch Plan

- Required resources: None. No database or shared resource is required.
- Scenario dependencies: All optional lenses: web-console (automatic launch and control; manual launch works without it), notification-hub (decision courier; the UI answers directly when it is down), prompt-manager (skill registry projection of the owned skills; direct file paths always work). Agent Manager, Program Runtime and Search Hub are not used.
- Operational risks: Codex session-home layout changes (mitigated by one reader module and unknown fallbacks); a desk bug writing a supervisor-owned file (mitigated by a single write allow-list checked in tests); duplicate skill IDs during the skills move taking the registry offline (mitigated by an atomic move with a compatibility path that the registry skips); session delete in Web Console destroying transcripts (the desk never deletes sessions).
- Launch sequencing: 1) design and file contract; 2) API and CLI proven read-only against the real research home and the 25 existing commissions; 3) packet builder and manual launch; 4) automatic launch proven on one throwaway session; 5) courier; 6) phone UI; 7) skills move and research-review (a second owner may run this in parallel from step 1); 8) stop test during a real commission.

## 🎨 UX & Branding

- Look & feel: Calm operations desk for a phone: a single scrolling Now view (needs-you count, live supervisors, stalled items), then Commissions, Topics, Launch and Review. Dense but legible cards, status as words plus colour, never colour alone.
- Accessibility: WCAG 2.2 AA; every action reachable by touch at 44px targets; status never conveyed by colour alone; unknown values shown as the word unknown; works at 360px width on the tunnel hostname with no localhost links.
- Voice & messaging: Plain, factual, short. Say what is known and how it was derived (claimed by the supervisor vs observed from files). Operator words are shown verbatim, never paraphrased.
- Branding hooks: Default vrooli design kit expression tuned to a quiet desk palette; mark and icons chosen later through brand-manager.

## 📎 Appendix

- Design: `docs/concepts/SYSTEM.md` (proofs in §11), `docs/reference/FILE-CONTRACT.md`, `docs/DECISIONS.md`, `docs/concepts/EXPERIENCE.md`.
- Method sources: `docs/agent-system/LONG_EFFORT_DESIGN.md` (its sections on stance messages, goal kernels and compaction, §4a).
