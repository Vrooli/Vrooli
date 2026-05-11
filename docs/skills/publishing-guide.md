# Skills Publishing Guide

End-to-end pipeline for publishing a Claude Skill from this repo to curated registries with full supply-chain attestation. This is the **how**; for the **what** and **why**, see [skills/README.md](../../skills/README.md) and [docs/monetization/catalogs/channels/skill-registries.md](../monetization/catalogs/channels/skill-registries.md). For the security checklist that gates publishing, see [security-baseline.md](security-baseline.md).

## What we publish

For each released skill version, the published artifact is a signed OCI image at:

```
ghcr.io/vrooli/skill-<slug>:<version>
```

with three referrers attached:

1. **Cosign signature** (Sigstore keyless via Fulcio) — proves the artifact came from a Vrooli GitHub Action invocation.
2. **SLSA Level 3 provenance attestation** — identifies the source commit, build steps, and CI environment.
3. **SBOM (CycloneDX)** — lists every dependency and transitive package in the skill bundle.

Consumers (registries, scanners, end users) verify all three before trusting the artifact.

## The pipeline (CI workflow)

The workflow lives in `.github/workflows/publish-skill.yml` (to be created when the first skill is ready to publish). It triggers on tags matching `skill-<slug>-v*.*.*`. Pseudo-shape:

```yaml
name: Publish Skill
on:
  push:
    tags:
      - 'skill-*-v*.*.*'

permissions:
  contents: read
  id-token: write       # required for keyless cosign
  packages: write       # required to push to ghcr.io
  attestations: write   # required for SLSA provenance

jobs:
  build:
    runs-on: ubuntu-latest
    outputs:
      digest: ${{ steps.push.outputs.digest }}
    steps:
      - uses: actions/checkout@v4
      - name: Parse tag → slug + version
        id: parse
        run: |
          # 'skill-git-control-tower-v1.2.3' → slug=git-control-tower version=v1.2.3
          ...
      - name: Sanitize SKILL.md (no hidden unicode)
        run: scripts/skill-sanitize.sh skills/${{ steps.parse.outputs.slug }}
      - name: Run Snyk Agent Scan
        uses: snyk/agent-scan-action@<pinned-sha>
        with:
          path: skills/${{ steps.parse.outputs.slug }}
          fail-on: critical,high
      - name: Run Cisco Skill Scanner
        uses: cisco-ai-defense/skill-scanner-action@<pinned-sha>
        with:
          path: skills/${{ steps.parse.outputs.slug }}
          fail-on: critical
      - name: Build OCI artifact
        run: scripts/skill-build.sh skills/${{ steps.parse.outputs.slug }} ${{ steps.parse.outputs.version }}
      - name: Generate SBOM with Syft
        uses: anchore/sbom-action@<pinned-sha>
        with:
          path: skills/${{ steps.parse.outputs.slug }}
          format: cyclonedx-json
          output-file: sbom.cdx.json
      - name: Push to ghcr.io
        id: push
        run: |
          oras push ghcr.io/vrooli/skill-${{ steps.parse.outputs.slug }}:${{ steps.parse.outputs.version }} \
            ./skill.tar.gz:application/vnd.vrooli.skill.tar+gzip
      - name: Attach SBOM as referrer
        run: oras attach --artifact-type application/vnd.cyclonedx+json \
          ghcr.io/vrooli/skill-${{ steps.parse.outputs.slug }}:${{ steps.parse.outputs.version }} \
          sbom.cdx.json
      - name: Sign with Cosign (keyless)
        uses: sigstore/cosign-installer@<pinned-sha>
      - run: |
          cosign sign --yes ghcr.io/vrooli/skill-${{ steps.parse.outputs.slug }}@${{ steps.push.outputs.digest }}
  provenance:
    needs: build
    permissions:
      id-token: write
      contents: read
      actions: read
    uses: slsa-framework/slsa-github-generator/.github/workflows/generator_container_slsa3.yml@<pinned-tag>
    with:
      image: ghcr.io/vrooli/skill-<slug>
      digest: ${{ needs.build.outputs.digest }}
      registry-username: ${{ github.actor }}
    secrets:
      registry-password: ${{ secrets.GITHUB_TOKEN }}
```

The actual workflow lives in the repo when implemented; the above is the documented shape.

## Local dry run before tagging a release

Before pushing a `skill-<slug>-v<version>` tag, run the pipeline locally as far as possible:

```bash
# 1. Sanitize for hidden unicode
scripts/skill-sanitize.sh skills/<slug>

# 2. Run scanners
docker run --rm -v "$PWD/skills/<slug>:/skill" snyk/agent-scan:latest /skill
docker run --rm -v "$PWD/skills/<slug>:/skill" cisco/skill-scanner:latest /skill

# 3. Build the bundle locally
scripts/skill-build.sh skills/<slug> v0.0.0-dryrun

# 4. Test loading in a real agent runtime
cp -r skills/<slug> ~/.claude/skills/<slug>
# Open Claude Code, give it a task the skill should handle, verify it loads + runs.
```

If any of (1)–(4) fails, do not tag.

## Registry submission

Once `ghcr.io/vrooli/skill-<slug>:<version>` is published and signed, register it with the curated discovery surfaces:

### 1. Anthropic curated marketplace (primary)

Open a PR to [`literal:anthropics/skills`](https://github.com/anthropics/skills) adding an entry pointing to the OCI artifact. The curation tier is the highest-trust surface; agents weight it more heavily than self-published listings.

### 2. skills.sh (Vercel)

```bash
gh skill publish ghcr.io/vrooli/skill-<slug>:<version>
```

Vercel's `gh skill` extension handles registry indexing and discovery for skills.sh.

### 3. Other registries (per audience fit)

- **ClawHub** (OpenClaw audience) — uses VirusTotal scanning at submission. Submit via [their submission flow](https://clawhub.example).
- **LobeHub Skills** — large general-purpose index. Submit if relevant.
- **Skills Directory** — curated index with stricter security review. Worth submitting for the badge.

For each registry, the source of truth is still `ghcr.io/vrooli/skill-<slug>:<version>` — registries link out, they don't host. This means a single release tag updates every registry simultaneously, and signature/provenance/SBOM are uniformly verifiable.

## Versioning and immutability

- Versions follow semver: `skill-<slug>-v<major>.<minor>.<patch>`.
- **Versions are immutable.** Once `skill-<slug>-v1.2.3` is published, never overwrite it. Bug fixes and security patches go to `v1.2.4` or higher. This is supply-chain hygiene, not a convenience.
- Skill version is independent of scenario version. A scenario at `gct-v3.4.5` may correspond to `skill-gct-v1.0.0`. Don't try to keep them in lockstep.

## Deprecation and revocation

When a skill version needs to be pulled (security finding, scenario API broken, anything):

1. **Cosign key revocation** via the Sigstore transparency log if the issue is signing-trust-related.
2. **Registry takedown** via each registry's removal flow. Some registries accept revocation lists; others require manual delisting.
3. **Public security advisory** in the GitHub Security Advisories tab on the source repo, with the affected version range and remediation guidance.
4. **Updated release** at the next version with the fix and a clear changelog entry referencing the advisory.

The kill-switch procedure must be runnable in under an hour from "vulnerability confirmed" to "registries delisted." Practice it before you need it.

## Observability and channel telemetry

Per [skill-registries.md](../monetization/catalogs/channels/skill-registries.md), `financial-tracker` ingests these signals when this channel activates:

- Install count by registry × time
- Referrer traffic from skill registries to Vrooli landing pages
- Install → free-self-host → subscription conversion
- Scanner pass rate at publish time
- Registry curation tier per skill version

The CI workflow emits structured publish events; the consumer-side install scripts emit anonymous install pings to a Vrooli-hosted endpoint with version, registry, and host-runtime (Claude Code / Codex / Cursor / etc.). The install ping has no PII. Document this in the SKILL.md transparently — agents and users have a right to see it.

## Workspace-sandbox integration

Where the wrapped scenario can run under [workspace-sandbox](../../scenarios/workspace-sandbox/), the install script should configure that as the default execution path. This converts Vrooli's per-run accountability substrate into a publish-time differentiator — see [security-baseline.md](security-baseline.md) for the full framing. Mention "runs under workspace-sandbox" in the SKILL.md description; it's a real trust signal.

## Common failure modes and recoveries

| Failure | Cause | Recovery |
|---|---|---|
| Snyk Agent Scan flags `curl ... \| bash` | Install script downloads from a non-pinned URL | Rewrite to use a release-tag URL with a SHA-256 checksum |
| Cisco Skill Scanner flags `Bash(*)` | `allowed-tools` is too broad | Constrain to `Bash(<cli-name>:*)` |
| Hidden-unicode CI step fails | Zero-width or bidi chars in SKILL.md | Run `scripts/skill-sanitize.sh` locally; commit the sanitized file |
| Cosign signing step fails with "OIDC token expired" | GitHub Actions OIDC token timeout | Retry the workflow; if persistent, check `permissions: id-token: write` |
| SLSA generator can't find the artifact | Image push hasn't propagated | Add a `sleep 30` between push and provenance steps, or use the digest output directly |
| Agent never loads the skill | YAML frontmatter description doesn't match agent task language | Rewrite description in agent-task terms, not marketing terms |

## Reference

- [Anthropic Agent Skills overview](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview)
- [SKILL.md format guide](https://code.claude.com/docs/en/skills)
- [Cosign / Sigstore documentation](https://docs.sigstore.dev/)
- [SLSA framework](https://slsa.dev/)
- [Syft SBOM generator](https://github.com/anchore/syft)
- [skills-oci pattern (Salaboy)](https://www.salaboy.com/2026/04/19/manage-and-distribute-skills-with-skills-oci/)
- [security-baseline.md](security-baseline.md) — what the publishing pipeline is enforcing
