import { describe, expect, it } from "vitest";
import {
  COVERAGES,
  TRUSTS,
  figureValue,
  formatAge,
  qualify,
  resolveInk,
  resolveReading,
  sourceName,
  type Ink,
  type ProvenanceReading,
} from "./ProvenanceInk";

// Minimal reading fixtures for the resolver's structural contract. Any richer
// reading (e.g. an adopter's own Reading) satisfies ProvenanceReading, so these
// assertions belong beside the resolver rather than in any one adopter.
const authoredSample = (value: number, series: number[] = [value]) => ({
  value,
  series,
  basis: "hand-authored, reviewed 2026-09-01",
});

const makeReading = (overrides: Partial<ProvenanceReading> = {}): ProvenanceReading => ({
  coverage: "NOW",
  trust: "VALID",
  value: null,
  observedAt: null,
  owner: null,
  whatIsNeeded: null,
  gapOpenDays: null,
  sample: null,
  source: { team: "director-swarm", binding: "scenario:swarm-manager" },
  ...overrides,
});

const base: ProvenanceReading = makeReading({ value: 12, observedAt: new Date().toISOString() });

describe("resolveInk — the one resolver", () => {
  it("maps every (coverage, trust) pair in the closed vocabularies to exactly one ink", () => {
    const inks: Ink[] = ["solid", "dimmed", "hollow", "dotted", "unavailable", "none"];
    for (const coverage of COVERAGES)
      for (const trust of TRUSTS) expect(inks).toContain(resolveInk(coverage, trust, true).ink);
  });
  it("draws measured readings with the correct ink", () => {
    expect(resolveInk("NOW", "VALID", false)).toEqual({
      ink: "solid",
      figure: "measured",
      finding: false,
    });
    expect(resolveInk("NOW", "CACHED", false)).toEqual({
      ink: "dimmed",
      figure: "measured",
      finding: false,
    });
    expect(resolveInk("NOW", "UNTRUSTED", false)).toEqual({
      ink: "solid",
      figure: "measured",
      finding: true,
    });
    expect(resolveInk("NOW", "UNAVAILABLE", false)).toEqual({
      ink: "unavailable",
      figure: "none",
      finding: false,
    });
  });
  it("draws illustrative figures hollow or dotted regardless of trust", () => {
    for (const trust of TRUSTS) {
      expect(resolveInk("IN-REACH", trust, true).ink).toBe("hollow");
      expect(resolveInk("MISSING", trust, true).ink).toBe("dotted");
      expect(resolveInk("UNREGISTERED", trust, true).ink).toBe("none");
    }
  });
  it("shows samples only for illustrative coverage", () => {
    expect(
      figureValue(
        { ...base, coverage: "MISSING", value: null, sample: authoredSample(5, [1, 5]) },
        resolveInk("MISSING", "UNAVAILABLE", true),
      ),
    ).toBe(5);
    expect(
      figureValue({ ...base, sample: authoredSample(5, [1, 5]) }, resolveInk("NOW", "VALID", true)),
    ).toBe(12);
  });
  it("treats a NOW cell without a value as unavailable", () =>
    expect(resolveReading({ ...base, value: null }).ink).toBe("unavailable"));
});

describe("qualify — no figure without its qualifier", () => {
  it("names the source and age for a live figure", () =>
    expect(qualify(base, resolveReading(base)).text).toMatch(/swarm-manager · observed \d+s ago/));
  it("names the owner and days open for an absent figure", () => {
    const reading: ProvenanceReading = {
      ...base,
      coverage: "MISSING",
      trust: "UNAVAILABLE",
      value: null,
      owner: "monetization",
      gapOpenDays: 14,
      sample: authoredSample(1),
    };
    expect(qualify(reading, resolveReading(reading))).toEqual({
      text: "no substrate · monetization · open 14 days",
      tone: "gap",
    });
  });
  it("names what is needed for an in-reach figure", () => {
    const reading: ProvenanceReading = {
      ...base,
      coverage: "IN-REACH",
      trust: "UNAVAILABLE",
      value: null,
      whatIsNeeded: "an endpoint",
      sample: authoredSample(1),
    };
    expect(qualify(reading, resolveReading(reading)).text).toBe("illustrative · needs an endpoint");
  });
  it("shows an integrity finding with an untrusted number", () => {
    const reading: ProvenanceReading = {
      ...base,
      trust: "UNTRUSTED",
      trustReason: "value exceeds the denominator",
    };
    expect(qualify(reading, resolveReading(reading))).toEqual({
      text: "swarm-manager · cannot be believed: value exceeds the denominator",
      tone: "amber",
    });
  });
});

describe("formatAge and sourceName", () => {
  it("scales the age unit", () => {
    const now = Date.parse("2026-09-01T12:00:00Z");
    expect(formatAge("2026-09-01T11:59:30Z", now)).toBe("30s ago");
    expect(formatAge("2026-09-01T11:20:00Z", now)).toBe("40m ago");
    expect(formatAge("2026-09-01T02:00:00Z", now)).toBe("10h ago");
    expect(formatAge("2026-08-20T12:00:00Z", now)).toBe("12d ago");
    expect(formatAge(null, now)).toBe("unknown age");
    expect(sourceName({ ...base, source: { team: "monetization" } })).toBe("monetization");
    expect(sourceName({ ...base, source: {} })).toBe("unknown source");
  });
  it("frames unavailable, cached, and unregistered readings", () => {
    const silent: ProvenanceReading = {
      ...base,
      trust: "UNAVAILABLE",
      value: null,
      trustReason: "deadline exceeded",
    };
    expect(qualify(silent, resolveReading(silent))).toEqual({
      text: "swarm-manager not answering · deadline exceeded",
      tone: "amber",
    });
    const cached: ProvenanceReading = {
      ...base,
      trust: "CACHED",
      observedAt: new Date(Date.now() - 90_000).toISOString(),
    };
    expect(qualify(cached, resolveReading(cached)).text).toMatch(
      /^last good 2m ago · source not answering$/,
    );
    const unregistered: ProvenanceReading = { ...base, coverage: "UNREGISTERED" };
    expect(qualify(unregistered, resolveReading(unregistered))).toEqual({
      text: "not registered",
      tone: "quiet",
    });
  });
});
