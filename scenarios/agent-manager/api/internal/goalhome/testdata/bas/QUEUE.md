# BAS queue

Full history through 2026-10-06 (handoffs, censuses, candidate evidence, the
long Forecast) is in `archive/QUEUE-through-2026-10-06.md`.

## Needs operator

(none; the operator answered BAS-FB-063 on 2026-10-06)

## Handoff

- 2026-10-07 operator: act on BAS-FB-064 now, before anything else.
  1. Accept E26 under the FB-064 gate amendment. Generated `ui/dist` identity is
     not a gate.
  2. Rewrite `## Forecast` for the quality destination in GOAL.md.
  3. Plan and admit the first journey slice (journeys 12 → 24). Spawn its worker.
  4. Keep this section short and replace it each check-in.
- The census notes from the previous handoff are in `archive/handoff-2026-10-07.md`.

## Forecast

- Destination: runtime ≤205,000. Current 253,452 (2026-10-07), gap 48,452.
- Last five: E21 −4,081/≤0; E22 +379/≤750 (feature); E23 −328/≤0;
  E24 −66/≤0; E25 −332/≤0 (brief's conservative estimate 0).

## Next

Each item is a redesign slice (`large-effort-orchestration` §2.1).

1. **RD-WF — one workflow model and pipeline (E23 accepted).** Remeasure
   those owners after E23 and admit ordered follow-on tranches only when an
   epoch-scale replacement and exact deletion list are evidenced.
2. **RD-REC — recording UI on one owner.** Return if a wider caller map finds a
   substantial owner overlap.
3. **RD-EXP — export on the replay spec.** E26 moves standalone HTML playback
   to the existing ReplaySpec-driven UI renderer.
4. **RD-WSUI and RD-AI** — plan after RD-WF from the same kind of owner map.
5. **DESK — desktop portability** (D22). Run it when a native machine is
   available.

## Waiting on others

These never hold acceptance or other slices.

- **E22-R1** — intended-player review of capture. Returns when QA bug
  `knw-1791087389154714359` is fixed.
- **DET-SIGNIN-R1** — real sign-ins with persistent profiles. Returns when the
  operator supplies dedicated, authorized test-only sites and accounts.

## Done

- E25 — RD-REC: one recording/execution workspace timeline controller. (`epochs/E25.md`)
- E24 — RD-REC first tranche: canonical `TimelineEntry` owns UI recording state. (`epochs/E24.md`)
- E23 — RD-WF first tranche: one generated-V2-backed UI workflow codec. (`epochs/E23.md`)
