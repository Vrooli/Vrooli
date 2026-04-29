# Scenario-Skill Template

Starter for a Claude Skill that wraps a single Vrooli scenario for external agent consumption. Copy this folder to `skills/<scenario-slug>/`, rename the `.template` files, and fill in.

## Files

- `SKILL.md.template` → rename to `SKILL.md`. Required. YAML frontmatter + markdown body.
- `scripts/install.sh.template` → rename to `scripts/install.sh`. Required if the scenario isn't already on the user's PATH. Pinned, checksummed, idempotent.

You may add:

- `references/*.md` — deeper docs the agent loads on demand. Useful for advanced workflows that bloat the SKILL.md if inlined.
- `assets/` — templates, configs, fixtures the install script copies into place.

You should NOT add:

- A `SKILL.md` at the template-folder level. Templates use `SKILL.md.template` to keep registry indexers from picking them up.
- Any committed signatures, SBOMs, or OCI manifests. Those are CI build outputs.
- Any secrets, API keys, or credentials. Skills should never embed these.

## How to fill in

1. **Pick a slug.** Match the scenario slug exactly (e.g., `git-control-tower`). Use kebab-case, ≤ 64 chars.
2. **Write the description first.** This is what registries index and what agents pattern-match against. One sentence, capability-focused, no marketing copy. Examples:
   - Good: *"Use when managing complex git workflows across multiple branches, PRs, or repos. Wraps the Vrooli GCT CLI to plan multi-PR sequences, enforce branch hygiene, and draft PR descriptions."*
   - Bad: *"Vrooli's revolutionary git control tower for the agentic era."* (no agent will load this for any task)
3. **Constrain `allowed-tools`** in the frontmatter to the minimum the skill needs. `Bash(gct:*)` is correct for a GCT skill; `Bash(*)` is not.
4. **Write the body short.** Pattern:
   - When to use this skill (1-3 lines)
   - Prerequisites (1-2 lines)
   - Setup (point to `./scripts/install.sh`, don't inline)
   - 3-5 most useful workflows with concrete commands
   - When NOT to use this skill (1-3 lines — agents respect this signal)
5. **Implement `scripts/install.sh`** with pinned commit SHAs + SHA-256 checksums for every download. No bare `curl https://example.com/install.sh | bash`. The script must be idempotent (running twice should be a no-op when already installed).
6. **No hidden unicode.** Run the file through a normalizer before committing (CI will reject otherwise).
7. **Test with a real agent runtime.** Load the folder into Claude Code (or equivalent), give it a task the skill should handle, verify it picks the skill up via the description and uses it correctly.

## Before opening the PR

Run through the [skills/SECURITY.md](../../SECURITY.md) checklist. CI catches some of it; reviewers catch the rest.

## After publish

The CI workflow signs (Cosign), provenances (SLSA L3 via slsa-github-generator), and pushes to `ghcr.io/vrooli/skill-<slug>:<version>`. From there, register with `anthropics/skills` (PR), `skills.sh` (Vercel CLI), and any other curated registries appropriate for the audience. See [docs/skills/publishing-guide.md](../../../docs/skills/publishing-guide.md).
