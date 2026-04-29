# Cross-Platform Ledger

Linux-only assumptions in Vrooli's internal code, tracked against the deployment tier where they matter. Modeled on a debt ledger: each entry has a target tier, an owning surface, a current state, and a resolution path.

This file is intentionally a **ledger** — not a build queue. Most entries live here for months while tier-2+ deployment is still speculative. The ledger keeps the debt visible without forcing premature work.

## Active deployment tiers

Per [docs/deployment/](../deployment/) and `CLAUDE.md`:

| Tier | Status | Cross-platform pressure |
|---|---|---|
| **1** — local Vrooli stack | live | Linux-only is acceptable; current infrastructure |
| **2** — desktop (Electron) | speculative | Highest pressure — packaging will surface most assumptions |
| **3** — mobile (iOS/Android) | speculative | Sandboxing pressure |
| **4** — cloud / SaaS | speculative | Container isolation; managed services |
| **5** — enterprise / appliance | speculative | Air-gapped; custom resource mapping |

**Rule:** Until a tier is on the active deployment roadmap, ledger entries targeting it stay ledger-only. Do not propose blocking swarm-manager work for speculative tiers.

## Honesty flags

Every entry's "current state" carries one:
- **`measured`** — confirmed by running on the target platform (or in CI matrix once Gap 5 ships)
- **`inferred`** — read in code; assumed to break based on the platform's known constraints
- **`fixed`** — resolution shipped; entry retained for history until the next quarterly cleanup

## Entries

> Initial pass is empty by design. The platform-code-auditor populates this file via `cross-platform-debt` decisions as it rotates through audit slices. Entries that land in this file before any audit has run would be unsourced speculation.

The first audit pass will likely surface entries in these areas (named here for orientation, not as ledger entries):

- **Path separators** — Linux `/` vs Windows `\`. Watch for hardcoded path concatenation in lifecycle, setup, log resolution.
- **Home directory layout** — `~/.vrooli/` is XDG-style on Linux; macOS prefers `~/Library/Application Support/`; Windows uses `%APPDATA%`.
- **Daemon styles** — autoheal's watchdog uses systemd on Linux. macOS uses launchd, Windows uses services. The CLI install command already references all three; verify they're tested.
- **Process signaling** — `SIGUSR1` / `SIGTERM` semantics differ on Windows; lifecycle's stop flow may need a Windows-specific path.
- **Shell assumptions** — `bash` vs `sh` vs `pwsh`. infra/scripts may carry bash-isms.
- **TCP port discovery** — ephemeral-range hazards differ across platforms; `docs/reference/port-allocation.md` already names this for Linux but needs a tier-2/3 review.
- **Filesystem case sensitivity** — Linux is case-sensitive; macOS HFS+ default is case-insensitive; can mask bugs that surface only on Linux CI or Windows.
- **Symlink behavior** — different on Windows.

## Entry template

When platform-code-auditor proposes a new entry via `cross-platform-debt`, the operator appends here using this shape:

```markdown
### CPL-<NNN>: <one-line title>

- **Target tier(s):** [2 / 3 / 4 / 5 — list any that the assumption breaks on]
- **Owning surface:** [exact path, e.g., `cli/scenarios/cmd_start.go:142`]
- **Pattern:** [1-2 sentences describing the assumption]
- **Current state:** [`measured` | `inferred` | `fixed`]
- **Resolution path:** [ledger-only until tier <N> activates | swarm-manager item <id>]
- **Source decision:** `<decision-id>` raised by platform-code-auditor on YYYY-MM-DD
```

## Update protocol

1. platform-code-auditor raises `cross-platform-debt` decision → operator approves at vision walk → entry appended here.
2. When a tier activates on the deployment roadmap, `infra-health` does a one-time review of all ledger entries targeting that tier and proposes which become swarm-manager items vs stay ledger-only.
3. When an entry's resolution ships, mark `current state: fixed` with the shipping decision id; do not delete (history matters for cross-platform regression hunting).

## Change log

- `2026-04-28` — File created with the active-tier matrix, honesty-flag legend, entry template, and known-area orientation list. No ledger entries yet — first audit pass will populate.
