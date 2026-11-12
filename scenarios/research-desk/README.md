# Research Desk

Configure, launch, watch, steer and review supervised research from a phone, without being
in the research loop.

Supervised research runs as Codex supervisor sessions in Web Console that staff
lower-powered owners over topics in `~/.vrooli/research`. The desk is a launcher, a lens
and a courier over those files and sessions. It builds the best-known packet (charter,
stance message, goal kernel) and launches it in one tap. It shows what is moving and what
changed since you last looked. It carries decisions to Notification Hub and answers back
into a file the supervisor reads, and it can pause, resume, wind down or stop a commission
without a chat message.

## Invariants

1. **Stop test.** Stopping the desk changes nothing for a running commission; no
   supervisor or owner step calls it.
2. **Files are the state.** The research home and Codex session records are
   authoritative; the desk keeps no database.
3. **Manual always works.** A packet folder launches by pasting two messages; the desk
   recognises commissions it did not launch.
4. **One writer per file.** The desk never writes `STATE.md`, `LEDGER.md`, checkpoints or
   `MISSION.md`.
5. **Optional lenses.** Web Console, Vrooli Events, Notification Hub (the capability
   registry's entries) and python3 (the skill scripts) may be down; affected fields read
   `unknown`.
6. **Unknown stays unknown.** All older commission layouts display; nothing is guessed.
7. **No new supervisor burden.** Read sets have size ceilings that a test enforces.

## Reading order

1. [`docs/concepts/SYSTEM.md`](docs/concepts/SYSTEM.md) — domains, surface, states, launch,
   control, courier, proofs.
2. [`docs/reference/FILE-CONTRACT.md`](docs/reference/FILE-CONTRACT.md) — the files,
   `packet.py`, the two grammars and the launch record.
3. [`docs/DECISIONS.md`](docs/DECISIONS.md) — why, when you need it.
4. [`docs/guides/skills-move.md`](docs/guides/skills-move.md) — for the research skills.
5. [`docs/internal/SEAMS.md`](docs/internal/SEAMS.md) — test boundaries.

Charter and operator words: `.vrooli/work/research-desk/` at the repository root. PRD and
requirements: `PRD.md`, `requirements/`. Pointer documents in `docs/` (marked
`not-applicable` in `docs/manifest.json`) only link to these.

## Running

```bash
make setup && make start     # lifecycle; never run binaries directly
make status
vrooli scenario test research-desk --phases structure,unit,docs
make real-home-check         # read-only invariants over ~/.vrooli/research
```

## Status

2026-11-12: API and CLI built for all three domains (activity, launch, operator) and the
courier, after review 3 (`.vrooli/work/research-desk/REVIEW-3-RULINGS.md`). The phone UI is
built by its own owner. Not built: the remaining-quota lens (B3) and the desk's own Web
Console credential client (D20). Requirements and how each is validated:
`requirements/01-must-ship/module.json`.
