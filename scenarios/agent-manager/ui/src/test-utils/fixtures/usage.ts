// Usage Intelligence fixtures built from the generated Measures schemas, so a
// contract change breaks these builders instead of silently passing tests.
import { create } from "@bufbuild/protobuf";
import {
  UsageBurnersResponseSchema,
  UsageCoverageResponseSchema,
  UsageFrictionResponseSchema,
  UsageInterventionsSchema,
  RunCostResponseSchema,
  RunCycleTimeResponseSchema,
  RunSuccessRateResponseSchema,
  RunStatusDistributionResponseSchema,
  ModelBreakdownResponseSchema,
  ProfileBreakdownResponseSchema,
  UsageForecastResponseSchema,
  UsageOutcomesResponseSchema,
  UsageParityResponseSchema,
  UsageCompareResponseSchema,
  type UsageOutcomesResponse,
  UsageExperimentResponseSchema,
  type UsageExperimentResponse,
  ListUsageModesResponseSchema,
  type ListUsageModesResponse,
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
    dormant: true,
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
        // The API's marks: the pool's longest window, and the most urgent live one.
        primary: true,
        focus: true,
      },
      overrides.idleClaude
        ? { ...CLAUDE_SCOPE, windowMinutes: 300n, status: "idle", hasPercent: true, usedPercent: 12, windowStart: at(-2), resetsAt: at(3), observedAt: at(-1), sampleCount: 5n, standing: "idle", primary: true }
        : { ...CLAUDE_SCOPE, status: "unknown", statusReason: "No status line reading yet.", sampleCount: 0n, primary: true },
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
        foldBelowPoints: 0.25,
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
    // The API's projection of the Codex weekly window (reset in 6 days) with its one lever applied: the run-out moves from 30 h to 48 h, 4 days short.
    outcome: { recoverablePoints: 9, levers: 1, projected: true, endPercent: 250, reachesReset: false, runOutAt: at(48), shortfallSeconds: 96n * 3600n, cutNeeded: 0.69 },
    actions: [
      {
        rank: 1, source: "lever", kind: "R-context", ownerKind: "harness", owner: "codex session context", title: "Compact codex sessions after ~60 requests of growth", ...CODEX_SCOPE,
        action: "Compact codex sessions about every 60 requests (lower the auto-compact threshold) instead of letting context grow",
        rationale: "70% of codex session quota was requests above 200k tokens.", confidence: "medium",
        effect: { value: 12, unit: "points", basis: "projected" }, score: 0.267, scoreBasis: "26.7% of the openai 10080m window's 45.0 observed points", recoverablePoints: 9,
        drilldown: [
          { kind: "session", id: "1e5904ed", label: "Research orchestrator", detail: "projected to save ≈8.0 pts (still running)", value: 8, stillRunning: true },
          { kind: "session", id: "76d970de", label: "Token usage analysis", detail: "would have saved ≈4.0 pts (finished); compacted 6×", value: 4 },
        ],
        evidenceRefs: [
          { kind: "run", id: BURNER_RUN_ID, label: "Vega E1 Overview" },
          // Sibling sub-agents share a label; their ids tell them apart.
          { kind: "run", id: "00b65371-7359-42d6-b8f4-83218914f65f", label: "sub-agent" },
          { kind: "run", id: "1d804127-2a6e-445b-85ad-1817512a1c02", label: "sub-agent" },
        ],
        verify: "after the change: `agent-manager usage compare …` should read improved",
      },
      {
        rank: 2, source: "tool", kind: "failure", ownerKind: "command", owner: "test-genie runs wait", title: "Fix failing `test-genie runs wait`",
        action: "Fix `test-genie runs wait`: it failed 161 times in 5 runs", rationale: "161 failures.", confidence: "high", fingerprints: ["fp-wait"],
        effect: { value: 161, unit: "calls", basis: "observed" }, score: 0.075, scoreBasis: "7.5% of the window's 2149 failed calls",
        evidenceRefs: [{ kind: "run", id: BURNER_RUN_ID }], verify: "after the fix, `agent-manager usage friction` should no longer list finding fp-wait",
      },
      {
        rank: 3, source: "lever", kind: "R-model-mix", ownerKind: "harness", owner: "codex sub-agent model", title: "Try luna for codex sub-agents", ...CODEX_SCOPE,
        action: "Run lighter codex sub-agent work on luna instead of terra", rationale: "a tier prior", confidence: "unproven",
        effect: { upper: 6, unit: "points", basis: "range" }, scoreBasis: "0–13.3% of the openai 10080m window's 45.0 observed points", verify: "compare",
      },
    ],
    belowMateriality: 3n, materialityScore: 0.01,
    coverage: ["operator interventions: 300 messages classified (90% coverage); no category has passed the accuracy gate, so none is recommended: cannot tell, not none"],
    scoring: ["assumption: kinds of harm weigh equally"],
    rules: [
      { rule: "R-context", fired: true, rows: 1n },
      { rule: "R-idle", fired: false, reason: "no session made wait-only requests this window" },
    ],
  });
}

export function makeUsageCoverage(): UsageCoverageResponse {
  return create(UsageCoverageResponseSchema, {
    since: at(-7 * 24),
    method: "Compare with: find ~/.codex/sessions -name '*.jsonl' -newermt '7 days ago'.",
    roots: [
      { kind: "web-console-codex-home", harness: "codex", files: 64n, indexed: 62n, unreadable: 2n, bytes: 150n, capturedBytes: 144n, capturedShare: 0.96, issues: [{ path: "/home/u/.codex/sessions/a.jsonl", reason: "still being written" }, { path: "/home/u/.codex/sessions/b.bin", reason: "unsupported format" }] },
      { kind: "agent-manager-run-home", harness: "mixed", files: 65n, owned: 65n, bytes: 50n, capturedBytes: 50n, capturedShare: 1 },
    ],
    totals: { kind: "total", files: 129n, indexed: 62n, owned: 65n, unreadable: 2n, bytes: 200n, capturedBytes: 194n, capturedShare: 0.97, turns: 9000n, usageReports: 8800n },
  });
}

/** The interventions section once the gate admits quality_miss: one category, one example repeating an earlier report. */
export function makeAdmittedInterventions() {
  return create(UsageInterventionsSchema, {
    operatorMessages: 120n, classified: 108n, coverage: 0.9, agentHours: 40, shown: 20n,
    categories: [{ category: "quality_miss", owner: "proof before claim: scientific-debugging", interventions: 20n, perAgentHour: 0.5, repeats: 3n, workRequests: 80n, workToolCalls: 55n, workTokens: 1_200_000n }],
    examples: [{ eventId: "evt-2", runId: "run-7", category: "quality_miss", text: "Still broken on mobile", repeatOfEventId: "evt-1" }],
  });
}

export const FRICTION_FINGERPRINT = "f1".repeat(16);

export function makeUsageFriction(): ReturnType<typeof create<typeof UsageFrictionResponseSchema>> {
  const tool = { ownerKind: "scenario", runs: 3n, costUnit: "calls", signature: "", exampleCommands: [] as string[], commandPath: "", bugId: "" };
  return create(UsageFrictionResponseSchema, {
    open: 3n, reported: 1n, dismissed: 1n, windowStart: "2026-10-22T06:00:00Z", belowFloor: 371n, materialityScore: 0.01,
    coverage: { commands: 175_706n, outcomeKnown: 70_282n, outcomeKnownShare: 0.4, vrooliCommands: 24_041n, vrooliOutcomeKnown: 8_400n, sentence: "175706 commands, outcome known for 40% (24041 Vrooli CLI calls, 35% known)" },
    toolFindings: [
      { ...tool, fingerprint: FRICTION_FINGERPRINT, kind: "failure", owner: "vrooli", ownerKind: "control-plane", description: "`search-hub query`, `prompt-manager skill read` and 6 other commands failed 128 times in 20 runs with: Error: agent identity identity_refresh_unavailable", calls: 128n, runs: 20n, cost: 128n, costShare: 0.158, signature: "identity_refresh_unavailable", exampleCommands: ["search-hub query recall notes"], status: "open",
        shares: [
          { kind: "failure", unit: "calls", cost: 101n, share: 0.124, base: "the window's 812 Vrooli CLI failed calls", sentence: "failure 101 calls = 12.4% of the window's 812 Vrooli CLI failed calls" },
          { kind: "degraded", unit: "calls", cost: 63n, share: 0.0025, base: "the window's 24041 Vrooli CLI calls", sentence: "degraded 63 calls = 0.3% of the window's 24041 Vrooli CLI calls" },
        ] },
      { ...tool, fingerprint: "d5".repeat(16), kind: "output-weight", owner: "prompt-manager", commandPath: "prompt-manager skill read", description: "`prompt-manager skill read` returns heavy output: median 6.1k, p90 24k, max 70k result tokens per call", calls: 1455n, costUnit: "tokens", cost: 19_300_000n, costShare: 0.031, status: "open",
        shares: [{ kind: "output-weight", unit: "tokens", cost: 19_300_000n, share: 0.031, base: "the window's context tax (622.6M re-read tokens)", sentence: "output-weight 19.3M tokens = 3.1% of the window's context tax (622.6M re-read tokens)" }] },
      { ...tool, fingerprint: "b3".repeat(16), kind: "degraded", owner: "agent-manager", commandPath: "test-genie runs", description: "`test-genie runs` reported \"agent-manager park unavailable … waiting inline instead\" 161 times in 34 runs", calls: 161n, cost: 126n, costShare: 0.0052, status: "reported", bugId: "knw-1793200000000000001",
        shares: [{ kind: "degraded", unit: "calls", cost: 126n, share: 0.0052, base: "the window's 24041 Vrooli CLI calls", sentence: "degraded 126 calls = 0.5% of the window's 24041 Vrooli CLI calls" }] },
      { ...tool, fingerprint: "c4".repeat(16), kind: "polling", owner: "device-control", commandPath: "device-control session list", description: "Agents poll `device-control session list`", calls: 208n, cost: 208n, costShare: 0.066, status: "dismissed",
        shares: [{ kind: "polling", unit: "calls", cost: 208n, share: 0.066, base: "the window's 3132 polled Vrooli CLI calls", sentence: "polling 208 calls = 6.6% of the window's 3132 polled Vrooli CLI calls" }] },
    ],
    interventions: { operatorMessages: 120n, classified: 108n, coverage: 0.9, agentHours: 40, unshownReason: "no category has passed the accuracy gate yet, so the classified messages are not shown" },
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

/** The pooled bucket: friction by mass, with per-category hints that are not proven. */
export function makePooledInterventions() {
  return create(UsageInterventionsSchema, {
    operatorMessages: 120n, classified: 108n, coverage: 0.9, agentHours: 40, shown: 18n,
    categories: [{ category: "operator_friction", pooled: true, interventions: 18n, perAgentHour: 0.45, workRequests: 40n, workToolCalls: 22n, workTokens: 500_000n, hints: [{ category: "quality_miss", owner: "scientific-debugging", count: 11n }, { category: "early_stop", owner: "harness-goal-authoring", count: 4n }] }],
    examples: [{ eventId: "evt-9", runId: "run-9", category: "quality_miss", pooled: true, text: "It is not fixed" }],
  });
}

/** One active Codex mode ending at the Codex window reset, with burn since it started. */
export function makeUsageModes(active = true, periods: Array<{ id: string; label: string; runner: string; startedAt: string; endedAt?: string; expiresAt?: string }> = []): ListUsageModesResponse {
  return create(ListUsageModesResponseSchema, {
    generatedAt: at(0),
    periods: periods.map((period) => ({ endedAt: "", expiresAt: "", ...period })),
    modes: active ? [{
      mode: {
        id: "um-test-1", label: "Lower tier for codex", kind: "save", scope: { runner: "codex" }, startedAt: at(-1), startedBy: "operator@example.com", source: "agent-manager",
        untilKind: "window_reset", untilProvider: "openai", untilWindowMinutes: 10080n, expiresAt: at(6 * 24),
      },
      secondsLeft: BigInt(6 * 24 * 3600),
      burn: [{ provider: "openai", pool: "primary", window: "10080m", observedIncreasePoints: 2.4 }],
    }] : [],
    reach: [{ provider: "openai", pool: "primary", window: "10080m", resetsAt: at(6 * 24), observedIncreasePoints: 10, operatorPoints: 5, profilePoints: 3, unattributedPoints: 2, operatorShare: 0.5, profileShare: 0.3, unattributedShare: 0.2 }],
    reachNotes: ["Desktop apps and harnesses started outside the shim cannot be reached."],
  });
}

const range = (value: number, low: number, high: number, samples = 20n) => ({ available: true, value, low, high, samples });

function outcomeRow(key: string, extra: Record<string, unknown> = {}) {
  return { key, runner: "codex", stratum: "operator", status: "enough", units: 40n, runs: 60n, accepted: 30n, completed: 0n, corrected: 4n, unknown: 6n, unknownRate: 0.28, verdict: "baseline", priceSource: "tier_prior", ...extra };
}

export type OutcomesKind = "none" | "early" | "observational" | "randomized" | "inconclusive";

/** `usage outcomes` as the API reports it, in the states the evidence view tells apart. */
export function makeUsageOutcomes(kind: OutcomesKind, by = "model"): UsageOutcomesResponse {
  const day = (offset: number) => new Date(Date.UTC(2026, 9, 1 + offset)).toISOString();
  const base = { generatedAt: "2026-10-30T12:00:00Z", windowStart: "2026-10-01T00:00:00Z", windowEnd: "2026-10-30T00:00:00Z", by, units: 97n, unknownUnits: 27n, completedUnits: 12n, design: kind === "randomized" ? "randomized" : "observational" };
  const mode = { id: "um-test-1", label: "Save usage", runner: "codex", startedAt: day(10), endedAt: day(14) };
  if (by === "mode") {
    return create(UsageOutcomesResponseSchema, { ...base, modePeriods: kind === "none" ? [] : [mode], rows: [
      outcomeRow("no_mode", { reworkRate: range(0.14, 0.09, 0.2) }),
      ...(kind === "none" ? [] : [outcomeRow("um-test-1", { reworkRate: range(0.18, 0.1, 0.27), corrected: 6n, units: 36n })]),
    ] });
  }
  const series = [
    { model: "terra", runner: "codex", buckets: [0, 1, 2, 3, 12, 13].map((d, i) => ({ at: day(d), points: 2 + i, runs: 3n })) },
    { model: "luna", runner: "codex", buckets: [10, 11, 12, 13].map((d, i) => ({ at: day(d), points: 0.5 + i / 2, runs: 4n })) },
  ];
  if (kind === "none") return create(UsageOutcomesResponseSchema, { ...base, rows: [outcomeRow("terra", { status: "not_enough_data", accepted: 2n, verdict: "not_enough_data" })], series });
  if (kind === "early") return create(UsageOutcomesResponseSchema, { ...base, modePeriods: [mode], rows: [outcomeRow("terra", { status: "not_enough_data", accepted: 3n, verdict: "not_enough_data" }), outcomeRow("luna", { status: "not_enough_data", accepted: 1n, verdict: "not_enough_data" })], series });
  return create(UsageOutcomesResponseSchema, {
    ...base, modePeriods: [mode], series,
    rows: [
      outcomeRow("terra", { pointsPerAccepted: range(9, 7, 11), reworkRate: range(0.14, 0.09, 0.2), activeMinutesPerRun: range(12, 9, 15) }),
      outcomeRow("luna", { verdict: kind === "inconclusive" ? "inconclusive" : "cheaper", baseline: "terra", pointsPerAccepted: range(2.5, 2, 3.4), cheaperBy: kind === "inconclusive" ? range(0.97, 0.13, 49.5) : range(3.6, 2.6, 4.5), consumptionRatio: range(1.4, 1.1, 1.9), breakEvenRatio: 5, modelPriceRatio: 4, modelPriceSource: kind === "randomized" ? "price_book" : "tier_prior", priceSource: kind === "randomized" ? "price_book" : "tier_prior", reworkRate: range(0.18, 0.1, 0.27), activeMinutesPerRun: range(9, 6, 12), pointsPerCompleted: range(1.8, 1.4, 2.2) }),
    ],
  });
}

export function makeUsageParity() {
  const measures = (denominator: number, usage: number, outcome: number) => [
    { name: "usage", numerator: BigInt(usage), denominator: BigInt(denominator), share: usage / denominator },
    { name: "outcome", numerator: BigInt(outcome), denominator: BigInt(denominator), share: outcome / denominator },
  ];
  return create(UsageParityResponseSchema, { generatedAt: "2026-10-30T12:00:00Z", rows: [
    { path: "am_headless", harness: "codex", runs: 100n, eligibleRuns: 100n, measures: measures(100, 92, 80) },
    { path: "am_interactive", harness: "codex", runs: 50n, eligibleRuns: 50n, measures: measures(50, 25, 20) },
    { path: "imported", harness: "claude", runs: 20n, eligibleRuns: 20n, measures: measures(20, 7, 2) },
  ] });
}

export function makeUsageCompare() {
  return create(UsageCompareResponseSchema, {
    left: { runs: 30n, acceptedUnits: 10n, pointsPerAccepted: 6, unknownUnits: 3n },
    right: { runs: 20n, acceptedUnits: 8n, pointsPerAccepted: 3, unknownUnits: 2n },
    pointsPerAcceptedChange: -0.5, verdict: "improved", caveats: ["Unknown outcomes are left out of the ratio."],
  });
}

const rng = (value: number, low: number, high: number) => ({ available: true, value, low, high, samples: 90n });

/** One Learn comparison as `usage experiment status` reports it. */
export function makeUsageExperiment(verdict: "cheaper_looks_better" | "not_enough_data" | "about_the_same" | "inconclusive" | "withheld" = "cheaper_looks_better", extra: Record<string, unknown> = {}): UsageExperimentResponse {
  const decided = verdict !== "not_enough_data" && verdict !== "inconclusive" && verdict !== "withheld";
  return create(UsageExperimentResponseSchema, {
    generatedAt: "2026-10-30T12:00:00Z",
    experiments: [{
      modeId: "um-learn-1", label: "Learn: code.default", role: "code.default", runner: "codex", incumbentModel: "gpt-6.1-terra", challengerModel: "gpt-6-luna", design: "randomized",
      splitPercent: 50, active: true, startedAt: "2026-10-01T00:00:00Z",
      incumbent: { arm: "incumbent", units: 130n, known: 120n, accepted: 100n, pointsPerAccepted: rng(10, 8, 12), reworkRate: rng(0.2, 0.1, 0.3) },
      challenger: { arm: "challenger", units: 128n, known: 118n, accepted: 99n, exceptions: 2n, pointsPerAccepted: rng(decided ? 5 : 9, decided ? 4 : 5, decided ? 6 : 14), reworkRate: rng(0.2, 0.1, 0.3) },
      costRatio: decided ? rng(0.5, 0.42, 0.6) : { available: false }, costPValue: decided ? 0.001 : 1, reworkDifference: rng(0, -0.08, 0.08), reworkPValue: 0.9,
      verdict, reason: decided ? "the challenger cost 50% less per accepted outcome" : "12 accepted outcomes on the incumbent and 9 on the challenger; each arm needs 75.",
      needAccepted: decided ? 0n : 66n,
      plan: { unitsPerDay: 0.2, costCv: 1.84, varianceAssumed: true, pilotUnits: 6n, effect: 0.5, power: 0.8, minAcceptedPerArm: 75n, minEffect: 0.3, reworkMargin: 0.2,
        detect: { reached: true, days: 1200, lowDays: 460, highDays: 3300, perArm: 120n }, same: { reached: false } },
      recommendation: {
        available: decided, role: "code.default", runner: "codex", fromModel: "gpt-6.1-terra", toModel: "gpt-6-luna", verdict, evidence: "the challenger cost 50% less per accepted outcome",
        policyFile: "resources/codex/model-policy.json", policyOps: ['replace /roles/code.default/model: "gpt-6.1-terra" -> "gpt-6-luna"'], decisionId: "decision-1",
      },
      ...extra,
    }],
  });
}
