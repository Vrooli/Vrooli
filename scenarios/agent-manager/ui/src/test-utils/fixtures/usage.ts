// Usage Intelligence fixtures built from the generated Measures schemas, so a
// contract change breaks these builders instead of silently passing tests.
import { create } from "@bufbuild/protobuf";
import {
  UsageBurnersResponseSchema,
  UsageCoverageResponseSchema,
  UsageFrictionResponseSchema,
  RunCostResponseSchema,
  RunCycleTimeResponseSchema,
  RunSuccessRateResponseSchema,
  RunStatusDistributionResponseSchema,
  ModelBreakdownResponseSchema,
  ProfileBreakdownResponseSchema,
  UsageForecastResponseSchema,
  UsageRecommendationsResponseSchema,
  type UsageBurnersResponse,
  type UsageCoverageResponse,
  type UsageForecastResponse,
  type UsageRecommendationsResponse,
} from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";

const HOUR = 3600_000;
// Relative to the test clock so "reading 3 min ago" is never stale by accident.
export const USAGE_NOW = Date.now();
const at = (offsetHours: number) => new Date(USAGE_NOW + offsetHours * HOUR).toISOString();

// Wire shapes as the API reports them; the UI names them "Codex · weekly" and "Claude · 5-hour".
export const CODEX_SCOPE = { provider: "openai", pool: "primary", window: "10080m" };
export const CLAUDE_SCOPE = { provider: "anthropic", pool: "five_hour", window: "300m" };
export const BURNER_RUN_ID = "2d230f9d-0000-4000-8000-000000000001";

/**
 * A window whose reset passed weeks ago with no reading since: the API may still
 * report it, and the UI must collapse it to one line.
 */
export function makeDormantWindow() {
  return {
    provider: "openai", pool: "primary", window: "300m", windowMinutes: 300n, status: "unknown",
    statusReason: `the window reset at ${at(-49 * 24)}`, hasPercent: true, usedPercent: 0,
    windowStart: at(-49 * 24 - 5), resetsAt: at(-49 * 24), observedAt: at(-49 * 24 - 1), sampleCount: 12n,
  };
}

export function makeUsageForecast(overrides: { observedAt?: string; status?: string; dormant?: boolean; idleClaude?: boolean } = {}): UsageForecastResponse {
  return create(UsageForecastResponseSchema, {
    windows: [
      {
        ...CODEX_SCOPE,
        windowMinutes: 10080n,
        status: overrides.status ?? "will_exhaust",
        hasPercent: true,
        usedPercent: 33,
        windowStart: at(-24),
        resetsAt: at(6 * 24),
        observedAt: overrides.observedAt ?? at(-0.05),
        sampleCount: 40n,
        burn: [
          { horizon: "1h", available: false, stale: true },
          { horizon: "6h", available: true, percentPerHour: 3.4, spanHours: 6 },
          { horizon: "24h", available: true, percentPerHour: 1.5, spanHours: 24 },
        ],
        centralPercentPerHour: 1.5,
        projectedExhaustionAt: at(30),
        exhaustionBandStart: at(18),
        exhaustionBandEnd: at(42),
        lastingPercentPerHour: 0.46,
        lastReadingPercent: 27,
        calibrationFactor: 1.12,
        calibrationWindows: 4n,
        series: [{ at: at(-24), usedPercent: 0 }, { at: at(-12), usedPercent: 10 }, { at: at(-0.05), usedPercent: 33 }],
      },
      overrides.idleClaude
        ? { ...CLAUDE_SCOPE, windowMinutes: 300n, status: "idle", hasPercent: true, usedPercent: 12, windowStart: at(-2), resetsAt: at(3), observedAt: at(-1), sampleCount: 5n, standing: "idle" }
        : { ...CLAUDE_SCOPE, status: "unknown", statusReason: "No status line reading yet.", sampleCount: 0n },
      ...(overrides.dormant ? [makeDormantWindow()] : []),
    ],
  });
}

export function makeUsageBurners(groupBy = "session"): UsageBurnersResponse {
  return create(UsageBurnersResponseSchema, {
    windows: [
      {
        ...CODEX_SCOPE,
        groupBy,
        usedPercent: 33,
        observedIncreasePoints: 33,
        attributedPoints: 30.3,
        unattributedPoints: 0.4,
        otherBurners: 77n,
        otherPoints: 0.3,
        burners: [
          { key: "1e5904ed", label: "Research orchestrator", points: 28, share: 0.84, sessions: 45n, runs: 0n, source: "indexed", model: "terra", modality: { imagesIn: 12n } },
          { key: groupBy === "run" ? BURNER_RUN_ID : "vega-e1", label: "Vega E1 Overview", points: 2.2, share: 0.07, sessions: 1n, runs: 1n, runIds: [BURNER_RUN_ID], source: "agent_manager", model: "terra", tokens: { input: 4_000_000n, cachedInput: 1_140_000_000n, cacheCreation: 600_000n, output: 850_000n, total: 1_145_450_000n }, requests: 3335n },
        ],
        calibration: { method: "fitted", rSquared: 0.91, slices: 48n, offDiskPointsPerHour: 0.02, weights: [{ runner: "codex", model: "terra", cachedInputTokensPerPoint: 40_000_000, inputWeight: 8, outputWeight: 40, source: "fitted" }] },
        pendingTokens: { total: 120_000n },
        meteredLines: [{ kind: "web_search", count: 3n, state: "unknown", note: "Codex does not report a charge" }],
      },
    ],
  });
}

export function makeUsageRecommendations(): UsageRecommendationsResponse {
  return create(UsageRecommendationsResponseSchema, {
    groups: [
      {
        rank: 1, targetKind: "session", targetId: "1e5904ed", targetLabel: "Research orchestrator", ...CODEX_SCOPE,
        combinedSavedPoints: 12, recoverablePoints: 9, usedSoFarPoints: 28, atThisRatePoints: 60, shareOfWindowBurn: 0.8, stillRunning: true,
        levers: [{
          rule: "R-context", lever: "context", savedPoints: 12, marginalPoints: 12, recoverablePoints: 9, confidence: "high", action: "Start a fresh session or compact the research orchestrator",
          rationale: "Each request re-reads about 185k tokens of context.",
          evidenceRefs: [
            { kind: "run", id: BURNER_RUN_ID, label: "Vega E1 Overview" },
            { kind: "session", id: "1e5904ed", label: "web-console 1e5904ed" },
            // Sibling sub-agents share a label; their ids tell them apart.
            { kind: "run", id: "00b65371-7359-42d6-b8f4-83218914f65f", label: "sub-agent" },
            { kind: "run", id: "1d804127-2a6e-445b-85ad-1817512a1c02", label: "sub-agent" },
          ],
        }],
      },
      {
        rank: 2, targetKind: "workload", targetId: "subagents", targetLabel: "Sub-agents", ...CODEX_SCOPE,
        combinedSavedPoints: 6, recoverablePoints: 0, usedSoFarPoints: 9, atThisRatePoints: 9,
        levers: [{ rule: "R-model", lever: "model", savedPoints: 6, marginalPoints: 6, confidence: "medium", action: "Run sub-agents on luna instead of terra", rationale: "luna succeeds on comparable work." }],
      },
    ],
    foldedGroups: 3n, foldedPoints: 0.2,
    rules: [
      { rule: "R-context", fired: true, rows: 1n },
      { rule: "R-idle", fired: false, reason: "no parked session consumed quota this window" },
    ],
  });
}

export function makeUsageCoverage(): UsageCoverageResponse {
  return create(UsageCoverageResponseSchema, {
    since: at(-7 * 24),
    method: "Compare with: find ~/.codex/sessions -name '*.jsonl' -newermt '7 days ago'.",
    roots: [
      { kind: "web-console-codex-home", harness: "codex", files: 64n, indexed: 62n, unreadable: 2n, capturedShare: 0.96, issues: [{ path: "/home/u/.codex/sessions/a.jsonl", reason: "still being written" }, { path: "/home/u/.codex/sessions/b.bin", reason: "unsupported format" }] },
      { kind: "agent-manager-run-home", harness: "mixed", files: 65n, owned: 65n, capturedShare: 1 },
    ],
    totals: { kind: "total", files: 129n, indexed: 62n, owned: 65n, unreadable: 2n, capturedShare: 0.97, turns: 9000n, usageReports: 8800n },
  });
}

export const FRICTION_FINGERPRINT = "f1".repeat(16);

export function makeUsageFriction(): ReturnType<typeof create<typeof UsageFrictionResponseSchema>> {
  const base = { owner: "agent-manager", ownerKind: "command", episodes: 18n, runs: 3n, wallClockMs: 185_000n, typicalWallClockMs: 10_000n, points: 0.4, timeKind: "redundant", episodeTurns: 0n, tokens: 0n, ...CODEX_SCOPE };
  return create(UsageFrictionResponseSchema, {
    open: 3n, reported: 1n, dismissed: 1n, runsAnalysed: 90n, runsTotal: 100n,
    problems: [
      { ...base, fingerprint: FRICTION_FINGERPRINT, pattern: "fallback_after_capability", description: "`agent-manager usage forecast` fails, then the agent falls back to another tool.", status: "open" },
      { ...base, fingerprint: "a2".repeat(16), pattern: "read_reread", description: "An agent reads a file, then reads the same file again.", owner: "Agent behaviour", ownerKind: "agent", status: "open", points: 9.3, estSavedPoints: 4.6, episodes: 1819n, runs: 26n },
      { ...base, fingerprint: "d5".repeat(16), pattern: "retry_after_failure", description: "`agent-manager usage burners` fails and the agent retries it straight away.", status: "open", wallClockMs: 9_000_000n, episodeTurns: 120n, tokens: 4_500_000n },
      { ...base, fingerprint: "b3".repeat(16), pattern: "stall", description: "A run sits idle between tool calls.", status: "reported", bugId: "knw-1793200000000000001" },
      { ...base, fingerprint: "c4".repeat(16), pattern: "wait_misuse", description: "An agent repeats a short wait.", status: "dismissed" },
    ],
  });
}

/** Answers for the six Details measures: 47 finished runs, 509.6 M tokens, no per-token billing. */
export function makeDetailsMeasures() {
  const validity = { state: "available", reason: "", sampleSize: 47n };
  return {
    runSuccessRate: create(RunSuccessRateResponseSchema, { rate: 0.957, successfulRuns: 45n, terminalRuns: 47n, validity }),
    runCost: create(RunCostResponseSchema, { totalTokens: 509_600_000n, cacheReadTokens: 496_000_000n, totalRuns: 47n, requests: 9000n, validity }),
    runCycleTime: create(RunCycleTimeResponseSchema, { p50DurationMs: 14_000, p90DurationMs: 110_000, completedDurationRuns: 47n, validity }),
    runStatusDistribution: create(RunStatusDistributionResponseSchema, { rows: [{ status: "pending", count: 2n }, { status: "starting", count: 1n }, { status: "running", count: 4n }, { status: "complete", count: 45n }], validity }),
    modelBreakdown: create(ModelBreakdownResponseSchema, { rows: [{ key: "gpt-6.1-terra", value: "gpt-6.1-terra", runCount: 24n, successCount: 24n, totalTokens: 346_800_000n }, { key: "gpt-6-luna", value: "gpt-6-luna", runCount: 23n, successCount: 21n, totalTokens: 162_800_000n }], validity }),
    profileBreakdown: create(ProfileBreakdownResponseSchema, { rows: [{ key: "", value: "unknown", runCount: 30n, successCount: 30n, totalTokens: 1_600_000n }, { key: "p1", value: "prompt-manager-delivery-orchestrator", runCount: 4n, successCount: 2n, totalTokens: 494_800_000n }], validity }),
  };
}
