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

This file, then [`docs/concepts/SYSTEM.md`](docs/concepts/SYSTEM.md) (how it works),
[`docs/reference/FILE-CONTRACT.md`](docs/reference/FILE-CONTRACT.md) (the files and
grammars) and [`docs/DECISIONS.md`](docs/DECISIONS.md) (why). What is not built or not
proven: [`docs/internal/PROBLEMS.md`](docs/internal/PROBLEMS.md). Requirements:
`requirements/01-must-ship/module.json`.

Outside the reading order: why the research method works, [`docs/concepts/METHOD.md`](docs/concepts/METHOD.md).

## Running

```bash
make setup && make start     # lifecycle; never run binaries directly
vrooli scenario test research-desk --phases structure,unit,docs
make real-home-check         # read-only invariants over ~/.vrooli/research
```

## Status

2026-11-12: seen live once: automatic launch, notice delivery and a supersede stop.
Not yet seen live: a decision ask with its answer, operator actions on a real commission, a
relaunch. Details and what is not built: [`docs/internal/PROBLEMS.md`](docs/internal/PROBLEMS.md).
