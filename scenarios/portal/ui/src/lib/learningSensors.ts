export type LearningEventKind =
  | "orientation"
  | "discovery"
  | "route_selection"
  | "first_action"
  | "verified_completion"
  | "failure";
export type LearningTemperature = "cold" | "warm";
export type LearningTopology = "local" | "remote";
export type LearningProvenance = "operator" | "test";
export type LearningReliability = "reliable" | "unreliable";
export type LearningReliabilityReason = "reliable" | "sample_too_small" | "sample_capped";

export interface LearningEvent {
  readonly kind: LearningEventKind;
  readonly at: number;
  readonly temperature: LearningTemperature;
  readonly topology: LearningTopology;
  readonly provenance?: LearningProvenance;
  readonly durationMs?: number;
  readonly outcome?: "verified_success" | "failed" | "unavailable" | "unknown";
  readonly targetClass?: "chat" | "desktop" | "terminal" | "device-panel";
  /** Stable acceptance/context keys used to form comparable cohorts. */
  readonly contextKey?: string;
  readonly acceptanceKey?: string;
  readonly assertionsCompleted?: number;
  readonly assertionsRequired?: number;
}

export interface LearningCohort {
  readonly eventCount: number;
  readonly durationSamples: number;
  readonly missingDurationDenominator: number;
  readonly failureCount: number;
  readonly p50DurationMs?: number;
  readonly p95DurationMs?: number;
}

export interface LearningComparison {
  readonly comparable: boolean;
  readonly reason: "comparable" | "context_mismatch" | "no_samples" | "assertion_regression";
  readonly fresh: LearningCohort;
  readonly reused: LearningCohort;
  readonly assertionIntegrity: "intact" | "regressed" | "unmeasured";
}

export interface LearningSnapshot {
  readonly eventCount: number;
  readonly byKind: Readonly<Record<LearningEventKind, number>>;
  readonly durationSamples: number;
  readonly p50DurationMs?: number;
  readonly p95DurationMs?: number;
  readonly missingDurationDenominator: number;
  readonly testEventCount: number;
  readonly reliability: LearningReliability;
  readonly reliabilityReason: LearningReliabilityReason;
  readonly cohorts: Readonly<Record<LearningTemperature, LearningCohort>>;
  readonly assertionViolations: number;
}

const EVENT_KINDS: readonly LearningEventKind[] = [
  "orientation",
  "discovery",
  "route_selection",
  "first_action",
  "verified_completion",
  "failure",
];

function quantile(values: readonly number[], q: number): number | undefined {
  if (values.length === 0) return undefined;
  const sorted = [...values].sort((a, b) => a - b);
  const index = (sorted.length - 1) * q;
  const lower = Math.floor(index);
  const upper = Math.ceil(index);
  const left = sorted[lower];
  const right = sorted[upper];
  if (left === undefined || right === undefined) return undefined;
  return left + (right - left) * (index - lower);
}

function cohort(events: readonly LearningEvent[]): LearningCohort {
  const durations = events.flatMap((event) => event.durationMs === undefined ? [] : [event.durationMs]);
  return {
    eventCount: events.length,
    durationSamples: durations.length,
    missingDurationDenominator: events.length - durations.length,
    failureCount: events.filter((event) => event.outcome === "failed" || event.outcome === "unavailable").length,
    p50DurationMs: quantile(durations, 0.5),
    p95DurationMs: quantile(durations, 0.95),
  };
}

function assertionViolations(events: readonly LearningEvent[]): number {
  return events.filter((event) =>
    event.assertionsRequired !== undefined &&
    event.assertionsCompleted !== undefined &&
    Number.isInteger(event.assertionsRequired) && event.assertionsRequired >= 0 &&
    Number.isInteger(event.assertionsCompleted) && event.assertionsCompleted >= 0 &&
    event.assertionsCompleted < event.assertionsRequired,
  ).length;
}

export class LearningSensor {
  private readonly eventsBuffer: LearningEvent[] = [];
  private readonly maxEvents: number;
  private readonly minEvents: number;
  private readonly now: () => number;
  private droppedEvents = 0;

  constructor(options: { maxEvents?: number; minEvents?: number; now?: () => number } = {}) {
    this.maxEvents = Math.max(1, Math.min(options.maxEvents ?? 512, 4096));
    this.minEvents = Math.max(1, Math.min(options.minEvents ?? 5, this.maxEvents));
    this.now = options.now ?? Date.now;
  }

  record(event: Omit<LearningEvent, "at"> & { at?: number }): void {
    if (!Number.isFinite(event.durationMs ?? 0) || (event.durationMs ?? 0) < 0) return;
    const normalized: LearningEvent = { ...event, provenance: event.provenance ?? "operator", at: event.at ?? this.now() };
    this.eventsBuffer.push(normalized);
    if (this.eventsBuffer.length > this.maxEvents) {
      this.droppedEvents += this.eventsBuffer.length - this.maxEvents;
      this.eventsBuffer.splice(0, this.eventsBuffer.length - this.maxEvents);
    }
  }

  events(): readonly LearningEvent[] {
    return this.eventsBuffer.slice();
  }

  snapshot(): LearningSnapshot {
    const byKind = Object.fromEntries(EVENT_KINDS.map((kind) => [kind, 0])) as Record<LearningEventKind, number>;
    const durations: number[] = [];
    let testEventCount = 0;
    for (const event of this.eventsBuffer) {
      if (event.provenance === "test") {
        testEventCount += 1;
        continue;
      }
      byKind[event.kind] += 1;
      if (event.durationMs !== undefined) durations.push(event.durationMs);
    }
    const eventCount = this.eventsBuffer.length - testEventCount;
    const operatorEvents = this.eventsBuffer.filter((event) => event.provenance !== "test");
    const cohorts = {
      cold: cohort(operatorEvents.filter((event) => event.temperature === "cold")),
      warm: cohort(operatorEvents.filter((event) => event.temperature === "warm")),
    } satisfies Record<LearningTemperature, LearningCohort>;
    const reliabilityReason: LearningReliabilityReason = this.droppedEvents > 0
      ? "sample_capped"
      : eventCount < this.minEvents
        ? "sample_too_small"
        : "reliable";
    return {
      eventCount,
      byKind,
      durationSamples: durations.length,
      p50DurationMs: quantile(durations, 0.5),
      p95DurationMs: quantile(durations, 0.95),
      missingDurationDenominator: eventCount - durations.length,
      testEventCount,
      reliability: reliabilityReason === "reliable" ? "reliable" : "unreliable",
      reliabilityReason,
      cohorts,
      assertionViolations: assertionViolations(operatorEvents),
    };
  }

  /**
   * Compare fresh (cold) and reused (warm) attempts only when they share the
   * same acceptance/context keys. Missing durations and failures stay in each
   * denominator, so an apparent speedup cannot be manufactured by omission.
   */
  compare(contextKey?: string, acceptanceKey?: string): LearningComparison {
    const events = this.eventsBuffer.filter((event) => event.provenance !== "test");
    const matching = events.filter((event) =>
      (contextKey === undefined || event.contextKey === contextKey) &&
      (acceptanceKey === undefined || event.acceptanceKey === acceptanceKey));
    const fresh = cohort(matching.filter((event) => event.temperature === "cold"));
    const reused = cohort(matching.filter((event) => event.temperature === "warm"));
    const freshViolations = assertionViolations(matching.filter((event) => event.temperature === "cold"));
    const reusedViolations = assertionViolations(matching.filter((event) => event.temperature === "warm"));
    const hasContext = contextKey !== undefined || acceptanceKey !== undefined;
    const contextMismatch = hasContext && matching.length === 0;
    const measuredAssertions = matching.some((event) => event.assertionsRequired !== undefined || event.assertionsCompleted !== undefined);
    const assertionRegression = reusedViolations > freshViolations;
    return {
      comparable: !contextMismatch && !assertionRegression && fresh.eventCount > 0 && reused.eventCount > 0,
      reason: contextMismatch ? "context_mismatch" : assertionRegression ? "assertion_regression" : fresh.eventCount > 0 && reused.eventCount > 0 ? "comparable" : "no_samples",
      fresh,
      reused,
      assertionIntegrity: !measuredAssertions ? "unmeasured" : assertionRegression ? "regressed" : "intact",
    };
  }
}

export const portalLearningSensor = new LearningSensor();
