# Skills Publishing Guide

End-to-end pipeline for publishing an Agent Plugin from this repo to OCI and curated registries with full supply-chain attestation. This is the **how**; for the **what** and **why**, see [SKILL-PUBLICATION.md](../concepts/SKILL-PUBLICATION.md) and [skill-registries.md](../../../../docs/monetization/catalogs/channels/skill-registries.md). For the security checklist that gates publishing, see [SECURITY-POSTURE.md](../internal/SECURITY-POSTURE.md).

## What we publish

For each released plugin version, the published artifact is a signed OCI artifact at:

```
ghcr.io/vrooli/plugin-<slug>:<version>
```

with three referrers attached:

1. **Cosign signature** (Sigstore keyless via Fulcio) — proves the artifact came from a Vrooli GitHub Action invocation.
2. **SLSA Level 3 provenance attestation** — identifies the source commit, build steps, and CI environment.
3. **SBOM (CycloneDX)** — lists every dependency and transitive package in the skill bundle.

Consumers (registries, scanners, end users) verify all three before trusting the artifact.

## The pipeline (CI workflow)

The repository workflow is `.github/workflows/publish-plugin.yml`. It validates the
scenario-owned declaration and comprehensive delivery suite on plugin release tags;
the API remains the authority for composition, managed evidence, governance, and
registry confirmation. It triggers on tags matching `plugin-<slug>-v*` or manually:

```yaml
name: Publish Agent Plugin
on:
  push:
    tags:
      - 'plugin-*-v*'

permissions:
  contents: read
  id-token: write       # required for keyless cosign
  packages: write       # required to push to ghcr.io
  attestations: write   # required for SLSA provenance

jobs:
  validate:
    runs-on: ubuntu-latest
    outputs:
      digest: ${{ steps.push.outputs.digest }}
    steps:
      - uses: actions/checkout@v4
      - name: Run the server-owned delivery suite
        run: vrooli scenario test scenario-to-plugin --preset comprehensive
      - name: Verify the governed source revision
        env:
          SOURCE_REVISION: ${{ github.sha }}
        run: test "${SOURCE_REVISION}" != ""
```

Publication itself is deliberately not a shell-side upload. An operator composes the
package through the scenario API, supplies managed Cosign/SLSA/CycloneDX evidence,
receives a passing deployment-manager decision for the exact source revision, and
invokes `scenario-to-plugin publish run <package-id> <channel>`. The distributor pushes
the artifact and all three OCI referrers, retrieves the manifest and referrer index, and
only then records the channel as published.

## Local dry run before tagging a release

Before pushing a `plugin-<slug>-v<version>` tag, run the pipeline locally as far as possible:

```bash
vrooli scenario test scenario-to-plugin --preset comprehensive
scenario-to-plugin readiness list
scenario-to-plugin package compose hello-plugin "$SOURCE_REVISION"
# Run `package show` and the governed check/rehearsal operations through the API.
# Publication is refused unless managed evidence and deployment-manager approval exist.
```

If any of (1)–(4) fails, do not tag.

## Registry submission

Once `ghcr.io/vrooli/plugin-<slug>:<version>` is published and signed, register it with the curated discovery surfaces:

### 1. Anthropic curated marketplace (primary)

Open a PR to [`literal:anthropics/skills`](https://github.com/anthropics/skills) adding an entry pointing to the OCI artifact. The curation tier is the highest-trust surface; agents weight it more heavily than self-published listings.

### 2. skills.sh (Vercel)

```bash
gh skill publish ghcr.io/vrooli/plugin-<slug>:<version>
```

Vercel's `gh skill` extension handles registry indexing and discovery for skills.sh.

### 3. Other registries (per audience fit)

- **ClawHub** (OpenClaw audience) — uses VirusTotal scanning at submission. Submit via [their submission flow](https://clawhub.example).
- **LobeHub Skills** — large general-purpose index. Submit if relevant.
- **Skills Directory** — curated index with stricter security review. Worth submitting for the badge.

For each registry, the source of truth is still `ghcr.io/vrooli/plugin-<slug>:<version>` — registries link out, they don't host. This means a single release tag updates every registry simultaneously, and signature/provenance/SBOM are uniformly verifiable.

## Versioning and immutability

- Versions follow semver: `plugin-<slug>-v<major>.<minor>.<patch>`.
- **Versions are immutable.** Once `plugin-<slug>-v1.2.3` is published, never overwrite it. Bug fixes and security patches go to `v1.2.4` or higher. This is supply-chain hygiene, not a convenience.
- Skill version is independent of scenario version. A scenario at `gct-v3.4.5` may correspond to `skill-gct-v1.0.0`. Don't try to keep them in lockstep.

## Deprecation and revocation

When a skill version needs to be pulled (security finding, scenario API broken, anything):

1. **Cosign key revocation** via the Sigstore transparency log if the issue is signing-trust-related.
2. **Registry takedown** via each registry's removal flow. Some registries accept revocation lists; others require manual delisting.
3. **Public security advisory** in the GitHub Security Advisories tab on the source repo, with the affected version range and remediation guidance.
4. **Updated release** at the next version with the fix and a clear changelog entry referencing the advisory.

The kill-switch procedure must be runnable in under an hour from "vulnerability confirmed" to "registries delisted." Practice it before you need it.

## Observability and channel telemetry

Per [skill-registries.md](../../../../docs/monetization/catalogs/channels/skill-registries.md), `financial-tracker` ingests these signals when this channel activates:

- Install count by registry × time
- Referrer traffic from skill registries to Vrooli landing pages
- Install → free-self-host → subscription conversion
- Scanner pass rate at publish time
- Registry curation tier per skill version

The CI workflow emits structured publish events; the consumer-side install scripts emit anonymous install pings to a Vrooli-hosted endpoint with version, registry, and host-runtime (Claude Code / Codex / Cursor / etc.). The install ping has no PII. Document this in the SKILL.md transparently — agents and users have a right to see it.

## Workspace-sandbox integration

Where the wrapped scenario can run under [workspace-sandbox](../../../workspace-sandbox/), the install script should configure that as the default execution path. This converts Vrooli's per-run accountability substrate into a publish-time differentiator — see [SECURITY-POSTURE.md](../internal/SECURITY-POSTURE.md) for the full framing. Mention "runs under workspace-sandbox" in the SKILL.md description; it's a real trust signal.

## Common failure modes and recoveries

| Failure | Cause | Recovery |
|---|---|---|
| Snyk Agent Scan flags `curl ... \| bash` | Install script downloads from a non-pinned URL | Rewrite to use a release-tag URL with a SHA-256 checksum |
| Cisco Skill Scanner flags `Bash(*)` | `allowed-tools` is too broad | Constrain to `Bash(<cli-name>:*)` |
| Hidden-unicode conformance fails | Zero-width, bidi, or non-NFC characters in SKILL.md | Fix the scenario-owned skill and rerun `scenario-to-plugin package compose` plus conformance |
| Cosign signing step fails with "OIDC token expired" | GitHub Actions OIDC token timeout | Retry the workflow; if persistent, check `permissions: id-token: write` |
| SLSA generator can't find the artifact | Image push hasn't propagated | Add a `sleep 30` between push and provenance steps, or use the digest output directly |
| Agent never loads the skill | YAML frontmatter description doesn't match agent task language | Rewrite description in agent-task terms, not marketing terms |

## Scenario-to-plugin delivery ramp

The repository's implemented publisher is `scenarios/scenario-to-plugin`.
It composes one canonical Agent Plugins 1.0.0 tree from a governed scenario
declaration:

```text
plugin-root/
├── plugin.json                 # closed Agent Plugins metadata manifest
├── skills/<name>/SKILL.md      # copied from the wrapped scenario
├── mcp.json                    # present only when an MCP declaration exists
└── cli/                        # standalone installer and runtime artifacts
```

The same composed tree is projected to two channels: a local/user-scoped
folder for clean-room installation and an OCI artifact for registries. The
operator sequence is readiness → compose → conformance → attestation →
workspace-sandbox rehearsal → deployment-manager release decision → publish.
The release decision must match the exact source revision; the ramp never
authorizes itself. Cosign, SLSA provenance, and CycloneDX references are bound
to the artifact digest, and publication is recorded only after retrieval.

Use `vrooli scenario test scenario-to-plugin` for the owned suite. For a
local dry run, the API's attestation operation produces reference-only evidence
without pushing to a registry. The permanent negative fixtures live under
`scenarios/scenario-to-plugin/testdata/conformance/`.

## Reference

- [Anthropic Agent Skills overview](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview)
- [SKILL.md format guide](https://code.claude.com/docs/en/skills)
- [Cosign / Sigstore documentation](https://docs.sigstore.dev/)
- [SLSA framework](https://slsa.dev/)
- [Syft SBOM generator](https://github.com/anchore/syft)
- [skills-oci pattern (Salaboy)](https://www.salaboy.com/2026/04/19/manage-and-distribute-skills-with-skills-oci/)
- [SECURITY-POSTURE.md](../internal/SECURITY-POSTURE.md) — what the publishing pipeline is enforcing
