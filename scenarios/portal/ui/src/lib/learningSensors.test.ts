import { describe, expect, it } from "vitest";

import { LearningSensor } from "./learningSensors";

describe("LearningSensor", () => {
  it("keeps bounded typed events and reports explicit duration denominators", () => {
    const sensor = new LearningSensor({ maxEvents: 2, now: () => 42 });
    sensor.record({ kind: "orientation", temperature: "cold", topology: "local" });
    sensor.record({ kind: "first_action", temperature: "warm", topology: "remote", durationMs: 100, targetClass: "chat" });
    sensor.record({ kind: "verified_completion", temperature: "warm", topology: "remote", durationMs: 300, outcome: "verified_success", targetClass: "chat" });

    expect(sensor.events()).toHaveLength(2);
    expect(sensor.snapshot()).toMatchObject({
      eventCount: 2,
      durationSamples: 2,
      p50DurationMs: 200,
      p95DurationMs: 290,
      missingDurationDenominator: 0,
    });
  });

  it("does not retain raw content or invalid durations", () => {
    const sensor = new LearningSensor();
    sensor.record({ kind: "failure", temperature: "cold", topology: "local", durationMs: -1 });
    sensor.record({ kind: "failure", temperature: "cold", topology: "local", durationMs: Number.NaN });
    expect(sensor.events()).toEqual([]);
  });

  it("excludes synthetic attempts and marks capped or small samples unreliable", () => {
    const missing = new LearningSensor({ minEvents: 2, now: () => 42 });
    missing.record({ kind: "orientation", temperature: "cold", topology: "local" });
    expect(missing.snapshot()).toMatchObject({ missingDurationDenominator: 1, reliability: "unreliable" });

    const sensor = new LearningSensor({ minEvents: 3, now: () => 42 });
    sensor.record({ kind: "orientation", temperature: "cold", topology: "local", provenance: "test" });
    sensor.record({ kind: "first_action", temperature: "warm", topology: "local", durationMs: 10 });
    sensor.record({ kind: "failure", temperature: "warm", topology: "local", durationMs: 20 });

    expect(sensor.snapshot()).toMatchObject({
      eventCount: 2,
      testEventCount: 1,
      reliability: "unreliable",
      reliabilityReason: "sample_too_small",
    });

    const capped = new LearningSensor({ maxEvents: 2, minEvents: 1 });
    capped.record({ kind: "first_action", temperature: "warm", topology: "local", durationMs: 10 });
    capped.record({ kind: "failure", temperature: "warm", topology: "local", durationMs: 20 });
    capped.record({ kind: "verified_completion", temperature: "warm", topology: "local", durationMs: 30 });
    expect(capped.snapshot()).toMatchObject({ reliability: "unreliable", reliabilityReason: "sample_capped" });
  });

  it("compares fresh and reused attempts only within the same acceptance context", () => { // LEARN-03
    const sensor = new LearningSensor({ now: () => 42 });
    sensor.record({ kind: "first_action", temperature: "cold", topology: "local", durationMs: 500, contextKey: "tv-home", acceptanceKey: "volume-up" });
    sensor.record({ kind: "failure", temperature: "cold", topology: "local", outcome: "failed", contextKey: "tv-home", acceptanceKey: "volume-up" });
    sensor.record({ kind: "first_action", temperature: "warm", topology: "local", durationMs: 200, contextKey: "tv-home", acceptanceKey: "volume-up" });
    sensor.record({ kind: "failure", temperature: "warm", topology: "local", outcome: "unavailable", contextKey: "tv-home", acceptanceKey: "volume-up" });

    expect(sensor.compare("tv-home", "volume-up")).toMatchObject({
      comparable: true,
      reason: "comparable",
      fresh: { eventCount: 2, durationSamples: 1, missingDurationDenominator: 1, failureCount: 1 },
      reused: { eventCount: 2, durationSamples: 1, missingDurationDenominator: 1, failureCount: 1 },
    });
    expect(sensor.compare("other-context", "volume-up")).toMatchObject({ comparable: false, reason: "context_mismatch" });
  });

  it("flags a reused speedup that drops required assertions", () => { // LEARN-05
    const sensor = new LearningSensor();
    sensor.record({ kind: "verified_completion", temperature: "cold", topology: "local", outcome: "verified_success", contextKey: "mail", acceptanceKey: "send", assertionsCompleted: 3, assertionsRequired: 3 });
    sensor.record({ kind: "verified_completion", temperature: "warm", topology: "local", outcome: "verified_success", contextKey: "mail", acceptanceKey: "send", assertionsCompleted: 2, assertionsRequired: 3, durationMs: 10 });
    expect(sensor.compare("mail", "send")).toMatchObject({ comparable: false, reason: "assertion_regression", assertionIntegrity: "regressed" });
    expect(sensor.snapshot().assertionViolations).toBe(1);
  });
});
