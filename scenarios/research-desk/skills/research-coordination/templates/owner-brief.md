# Owner brief template

The research manager sends this to each owner with `fork_turns=none`, filled from the card and the
charter; owners never read the commission folder.

```text
You own [topic] for this commission. Land [the card's workstreams, in its order if it gives one] toward their showcases. [The card's Read first line and the operator's notes for this card, verbatim. Include the manager's concrete application of each relevant operator feedback item and its source; include image paths after inspecting them. If the topic has work/<ID>/BRIEF.md for a workstream, name its path: start from it.]

The operator's words: [the stance quotes, then any direction for this topic, verbatim].

Your charter is [abs MISSION.md path]. Re-read it at each checkpoint and after any compaction. Read the adaptive-research skill first ([its path]); it wins over any other skill's stop rule. Mode [mode]: [the mission's approval flow line]. Real boundaries and Not this: [the mission's sections, unchanged]. [The charter's Shared machine section.]

Build big. Everything in the repository is in scope unless your mission reserves it: deletions, migrations, whole APIs and scenarios, plus other agents' uncommitted work (integrate it, never discard it). Size changes to the destination. Prove them on real software and targets. For parallel work, give each worker an hours-long outcome and fresh brief (use `work/<ID>/BRIEF.md` when present); it need not read the mission or checkpoint. Spawn native Codex sub-agents (`fork_turns=none`); if unavailable, use `vrooli agent launch --runner codex --prompt "Read <brief path> and complete its assignment"`. Never use Agent Manager.

**Window allocation:** [destination estimate from the mission; this commission's allowance; bounded slice and its hours; one primary owner for the outcome; how helpers return integration to that owner]. Never assign a whole destination whose estimate exceeds the available window. If the topic has an allocation command, include and run its exact invocation before work, and save the receipt in the run's `evidence/` folder. For design-to-implementation, use `tools/delivery-gate/delivery_gate.py allocate`; its output names the bounded slice and single-owner integration path.

Keep the mission outcome and evidence standard when you switch tools. If part of the proof is unavailable, advance the independent work and name the missing evidence.

Optional services fail fast: when one fails, take the direct path and log one line in your checkpoint. Message the manager for four things only: your plan (once, naming what you start in the first hour, while you build), a showcase claim, an item only the operator can decide, a bright line or a direction (an ID, two plain sentences, your recommendation, and the path of a ready-to-run brief for it), and a need you cannot meet. If a showcase can only wait, say so in one message and take the next workstream.

Keep [run folder]/CHECKPOINT.md under 30 KB and current: changed, verified, remaining, unverified, measured changed paths, dimensions moved, and everything you started under `Started`. Stop what you start when its job ends. Run `python3 -I [F] diff --baseline [B] --keep-file [K]` before every message to the manager and before GPU work; stop only your own lines. [If the card names list files: re-read them each checkpoint and assess every new Wanted line.] You work for hours; return with a showcase claim or after about six hours.
```
