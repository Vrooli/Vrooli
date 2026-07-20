# `security` phase

The `security` phase is a **thin delegating phase**: it calls the
`security-health` scenario's shared `ScenarioValidationService` and maps its
normalized findings into the shared `FINDING_SOURCE_SECURITY` channel.
test-genie itself contains **no security scanners** — all stack-specific
scanning (secrets, Go SAST, Go vuln-DB, JS dependency CVEs) lives in
`security-health`, which is the only component that knows how to scan each
substrate. This mirrors the other shared health-provider phases.

This phase declares a [Phase Capability Contract](../../concepts/phase-capability-contract.md); the sections below follow the required remediation-doc skeleton.

## North Star

Every scenario has a **clean, continuously-monitorable security posture**: its
scanners (secrets, Go SAST, Go vuln-DB, JS dependency CVEs) run and complete, no
scanner reports a leaked credential, reachable vulnerability, or production
dependency CVE, its first-party API stamps baseline security headers at a central
boundary with no unsafe credentialed-wildcard CORS, and only clean advisory
vulnerability intelligence remains. At maximum maturity the deepest capability —
`security_findings` — is L3 (clean for continuous fleet monitoring) and the
supporting ladders (`target_resolution`, `substrate_coverage`, `scanner_coverage`,
`response_header_policy`) are each at their top rung.

## The rungs and their gates

The phase aggregates five per-capability ladders; the load-bearing one is
`security_findings` (unmapped scanner findings fall back to it as ERROR safety
blockers). The rungs are monotone.

| Rung (security_findings) | Gate | Next unlock |
|---|---|---|
| L0 Unavailable | Scanner findings cannot be evaluated for the target. | Scanner findings are normalized into the shared security report. |
| L1 Findings normalized | Findings carry severity, location, and remediation. | Clear every ERROR-class secret, SAST, reachable-vuln, and production dependency vuln. |
| L2 Safety blockers absent | No ERROR-class safety blocker remains. | Advisory vulnerability intelligence is clean. |
| L3 Posture clean | Findings and advisory dependency intelligence are clean for continuous fleet monitoring. | Maximum security-findings maturity reached. |

The other capabilities share the same L0→top shape: scanners must be discoverable
(`scanner_coverage`), substrates classified with explicit gaps
(`substrate_coverage`), and baseline headers centralized with safe CORS
(`response_header_policy`).

## What each finding means

Each finding caps the capability it names at a rung; only ERROR/BLOCKER
severities fail the phase, so absent-scanner and degraded observations stay
advisory.

| Code | Capability | Caps at | Severity | Fails phase? |
|---|---|---|---|---|
| _(scanner secret/SAST/vuln finding — fallback)_ | security_findings | L2 | ERROR | Yes |
| `security-health.security-headers-missing` | response_header_policy | L2 | ERROR | Yes |
| `security-health.insecure-cors` | response_header_policy | L3 | ERROR | Yes |
| `security-health.security-headers-legacy-xss` | response_header_policy | L3 | WARNING | No |
| `security-health.security-headers-degraded` | response_header_policy | L1 | INFO | No |
| `security-health.substrate-unsupported` | substrate_coverage | L2 | INFO | No |
| `security-health.scanner-absent` | scanner_coverage | L2 | INFO | No |
| `security-health.scanner-degraded` | scanner_coverage | L3 | INFO | No |
| `security-health.scenario-not-found` | target_resolution | L1 | WARNING | No |

Real vulnerability findings (leaked secrets, reachable CVEs, high/critical SAST)
arrive as normalized scanner findings routed through the descriptor `fallback`
(`security_findings`, ERROR) — they are the blocking core; the named codes above
cover scanner-infrastructure and response-header posture.

## The canonical fix

- **Scanner secret/SAST/vulnerability findings** → remediate the specific finding: rotate/remove the leaked credential, fix the SAST defect, or upgrade the vulnerable dependency past the fixed version. These hard-gate the swarm-manager `security` dimension (R1 "Safe").
- **`security-health.security-headers-missing`** → auto-fixable (`fixer_status: implemented`): centralize baseline security headers in one API-router middleware. Run `security-health fix preview|apply <scenario>`.
- **`security-health.insecure-cors`** → manual: replace credentialed wildcard CORS with a scenario-specific origin + credential policy; a generic rewrite would be unsafe.
- **`security-health.security-headers-legacy-xss`** → auto-fixable: drop the legacy `X-XSS-Protection` header.
- **`scanner-absent` / `scanner-degraded` / `substrate-unsupported`** → advisory: install the missing scanner binary (e.g. `osv-scanner`) or accept the documented coverage gap; these never fail the phase.

## How to verify

```bash
# See the current rung, gaps, and next move for every capability:
security-health validate scenario <scenario>

# Or drive it through Test Genie and read the per-phase scorecard:
test-genie execute <scenario> --phases security
test-genie runs findings --scenario <scenario>
```

## What it runs

```text
scenario-validation/v1.ScenarioValidationService.ValidateScenario
```

Test Genie reads the shared `status` and `assessment.findings` fields. Each
assessment finding is mapped to an `ArchitectureFinding{Source:
FINDING_SOURCE_SECURITY}`, so it carries a deterministic stable ID, normalized
severity, and the per-source effort default (`MEDIUM`).

## Severity contract (load-bearing)

security-health performs the normalization; this phase routes its severity
strings through the same `normalizeFindingSeverity` table every producer uses:

| security-health severity | normalized | gates R1? |
|---|---|---|
| `SEVERITY_ERROR` (critical/high) | ERROR | **yes** |
| `SEVERITY_WARNING` (moderate/medium) | WARNING | no |
| `SEVERITY_INFO` (low/info/degraded) | INFO | no |

Only ERROR findings fail the phase. They flow into the swarm-manager
`security` dimension, which hard-gates the **R1 ("Safe")** ladder rung — so a
leaked credential or a reachable CVE holds a scenario at R1 until it is
resolved.

## Classification

- **Optional**, auto-joins the `comprehensive` preset (anti-drift guard).
- **180s timeout** — scanners (govulncheck/osv-scanner) hit network vuln DBs
  and are slower than the other delegating phases.
- Unreachable `security-health` API → advisory skip because the phase is
  optional. Start it via `vrooli scenario start security-health`.
- `TEST_GENIE_SKIP_SECURITY=1` skips the phase entirely (matches the
  `ui-health` escape hatch).

## Degraded scanners

When an applicable scanner's binary is absent (e.g. `osv-scanner` not
installed) or cannot complete, security-health emits an **INFO** observation
rather than a failure. The phase stays green; the gap is visible but advisory.

## See also

- `scenarios/security-health/` — the producer scenario.
- `packages/proto/schemas/architecture/v1/findings.proto` —
  `FINDING_SOURCE_SECURITY = 9`.
- `scenarios/security-health/.vrooli/test-genie.json` — the descriptor-owned
  `security` dimension and phase coverage metadata.
