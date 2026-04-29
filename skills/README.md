# Vrooli Skills

This folder is **publication source** for Claude Skills (the [open SKILL.md standard](https://github.com/anthropics/skills)) that teach external agents — Claude Code, Codex CLI, Gemini CLI, Cursor, Windsurf, etc. — how to use specific Vrooli capabilities standalone.

It is **not** a runtime skills directory. Internal skills used while working *on* Vrooli (vision walks, planning, debugging, etc.) live in [prompt-manager](../scenarios/prompt-manager/), invoked via `prompt-manager skill ...`. Don't confuse the two:

| Surface | Audience | Storage | Access |
|---|---|---|---|
| Internal skills | Agents working *on* Vrooli (you, right now) | `prompt-manager` | `prompt-manager skill ...` CLI |
| External skills | Agents in other runtimes using Vrooli capabilities | This folder (`skills/`) | Published via OCI / curated registries |

Different audience, different content shape, different lifecycle. Don't try to unify them.

## What lives here

- One subfolder per published skill (folder name = skill slug, kebab-case).
- `SECURITY.md` — disclosure policy, signing/scanning conventions, kill-switch process. Read this before publishing.
- `.templates/scenario-skill/` — starter template for a new scenario-wrapping skill. Copy, fill in, publish.

The leading dot on `.templates/` keeps it out of skill-shaped scans. Templates use `SKILL.md.template` (not `SKILL.md`) so registry indexers don't mistake them for real skills.

## When to add a skill here

Most scenarios should NOT publish a skill. Default answer is no. Add one only when **all** of these are true:

1. The capability has sharp standalone value — an external agent could plausibly want this without the rest of Vrooli.
2. The capability is **standalone-installable** — there's a one-command install path that doesn't drag in the full Vrooli runtime. (This is usually the gating prerequisite. If your scenario isn't standalone-installable, that's the work to do first; the skill is downstream.)
3. The CLI surface is stable enough that skill ↔ CLI drift won't be a constant maintenance burden.

Strong fits today: Git Control Tower, Prompt Manager, Workspace Sandbox.
Weak fits today: anything tightly coupled to Swarm Manager / Agent Manager / scenarios that assume other Vrooli scenarios are running.
Never fits: scenarios where `--help` + a README would already do the job (don't publish a "use the CLI's help command" skill — registries will reject it as low-value).

## How to add a skill

1. Read [SECURITY.md](SECURITY.md) and [docs/skills/security-baseline.md](../docs/skills/security-baseline.md). Both are non-negotiable.
2. Copy `.templates/scenario-skill/` to `<scenario-slug>/`. Rename `SKILL.md.template` → `SKILL.md` and `scripts/install.sh.template` → `scripts/install.sh`.
3. Fill in YAML frontmatter (`name`, `description`, `allowed-tools`). Keep `allowed-tools` minimal — only what's strictly needed.
4. Write the markdown body. Be terse. Pattern: when to use, prerequisites, install (point to script), 3-5 most useful workflows. The description is what registries index; the body is what an agent reads after deciding to load the skill.
5. Implement `scripts/install.sh` with pinned commit SHAs and checksums. No bare `curl | bash` to mutable URLs.
6. Sanitize for hidden unicode (zero-width chars, bidi marks, non-NFC normalization).
7. Run [docs/skills/publishing-guide.md](../docs/skills/publishing-guide.md) — local scanner pass + Cosign sign + SLSA provenance + SBOM.
8. Open a PR. Review checklist in [SECURITY.md](SECURITY.md) is mandatory.

## Folder conventions

```
skills/
├── README.md                     ← this file
├── SECURITY.md                   ← disclosure + signing conventions
├── .templates/
│   └── scenario-skill/           ← starter for a scenario-wrapping skill
│       ├── README.md
│       ├── SKILL.md.template
│       └── scripts/
│           └── install.sh.template
├── git-control-tower/            ← real skill (when added)
│   ├── SKILL.md
│   ├── scripts/
│   │   └── install.sh
│   └── references/               ← optional, agent-readable deep-dive docs
└── ...
```

- Skill folder name = slug = scenario slug (1:1). If a scenario has multiple skills (rare), use suffixes: `git-control-tower-pr-flow/`.
- No top-level `SKILL.md`. Each skill is a folder.
- Publication artifacts (signatures, SBOMs, OCI manifests) are CI outputs, not committed here.

## Why these are publication source rather than the published artifact itself

The folder you see here is what humans edit. CI (see [docs/skills/publishing-guide.md](../docs/skills/publishing-guide.md)) builds these into signed OCI artifacts published to `ghcr.io/vrooli/skill-<slug>:<version>` and registered with curated registries. The published artifact has the signature, SLSA provenance, and SBOM attached; the source folder doesn't (and shouldn't — those are build-time and tied to a release tag).

So if you're an agent reading this folder directly: you're looking at editable source. The trust artifact for any specific version of a skill lives at the OCI registry, not here.

## Related docs

- [docs/skills/publishing-guide.md](../docs/skills/publishing-guide.md) — full publish pipeline (Cosign, SLSA L3, SBOM, OCI, registry submission)
- [docs/skills/security-baseline.md](../docs/skills/security-baseline.md) — 13-point security checklist + OWASP Agentic Skills Top 10 mapping
- [docs/monetization/channels/skill-registries.md](../docs/monetization/channels/skill-registries.md) — why this folder exists (channel discipline, activation triggers)
