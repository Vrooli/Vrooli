# Vrooli Skills

This folder is **publication source** for Claude Skills (the [open SKILL.md standard](https://github.com/anthropics/skills)) that teach external agents — Claude Code, Codex CLI, Antigravity CLI, Cursor, Windsurf, etc. — how to use specific Vrooli capabilities standalone.

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

## Scenario-to-skill readiness checklist

A scenario is ready to become a published skill only when the answer to each question is yes:

1. **Standalone value:** Would an external agent choose this capability without needing the rest of Vrooli?
2. **Standalone install:** Is there a one-command install path that avoids dragging in the full Vrooli runtime by stealth?
3. **Stable interface:** Is the CLI/API surface stable enough that the skill will not drift every few days?
4. **Security posture:** Can the skill pass the full security baseline with minimal `allowed-tools`, pinned downloads, scanner clearance, signature, provenance, and SBOM?
5. **Telemetry hook:** Can installs, registry referrers, scanner status, and downstream conversion be attributed to this skill separately?
6. **Subscription path:** Is the eventual convenience-layer path clear for users who want managed gateway access, hosted infrastructure, or the broader bundle?

Publishing is valid before direct monetization exists. Free agent usage is a validation channel: it proves standalone task value, exposes integration failures, and builds registry trust. It is not a reason to weaken curation. A low-value or insecure skill damages the channel more than no skill.

## How to add a skill

1. Read [SECURITY-POSTURE.md](../internal/SECURITY-POSTURE.md) and [SECURITY-POLICY.md](../internal/SECURITY-POLICY.md). Both are non-negotiable.
2. Copy `.templates/scenario-skill/` to `<scenario-slug>/`. Rename `SKILL.md.template` → `SKILL.md` and `scripts/install.sh.template` → `scripts/install.sh`.
3. Fill in YAML frontmatter (`name`, `description`, `allowed-tools`). Keep `allowed-tools` minimal — only what's strictly needed.
4. Write the markdown body. Be terse. Pattern: when to use, prerequisites, install (point to script), 3-5 most useful workflows. The description is what registries index; the body is what an agent reads after deciding to load the skill.
5. Implement `scripts/install.sh` with pinned commit SHAs and checksums. No bare `curl | bash` to mutable URLs.
6. Sanitize for hidden unicode (zero-width chars, bidi marks, non-NFC normalization).
7. Run [build-and-publishing.md](../guides/build-and-publishing.md) — local scanner pass + Cosign sign + SLSA provenance + SBOM.
8. Open a PR. Review checklist in [SECURITY.md](SECURITY.md) is mandatory.

## Folder conventions

```
skills/
├── README.md                     ← this file
├── SECURITY.md                   ← disclosure + signing conventions
├── .templates/
│   └── scenario-skill/           ← starter template for a scenario-wrapping skill
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

The scenario-owned source is what humans edit. CI (see [build-and-publishing.md](../guides/build-and-publishing.md)) builds it into signed OCI artifacts published to `ghcr.io/vrooli/plugin-<slug>:<version>` and registered with curated registries. The published artifact has the signature, provenance, and SBOM attached; source does not.

So if you're an agent reading this folder directly: you're looking at editable source. The trust artifact for any specific version of a skill lives at the OCI registry, not here.

## Related docs

- [build-and-publishing.md](../guides/build-and-publishing.md) — full publish pipeline (Cosign, SLSA L3, SBOM, OCI, registry submission)
- [SECURITY-POSTURE.md](../internal/SECURITY-POSTURE.md) — 13-point security checklist + OWASP Agentic Skills Top 10 mapping
- [docs/monetization/catalogs/channels/skill-registries.md](../docs/monetization/catalogs/channels/skill-registries.md) — why this folder exists (channel discipline, activation triggers)
