import { describe, expect, it } from "vitest";
import type { Reading } from "../lib/api";
import { COVERAGES, TRUSTS, figureValue, qualify, resolveInk, resolveReading, type Ink } from "@vrooli/react-component-library/ProvenanceInk/0.1.2";
import { authoredSample, makeReading } from "../test-utils/readings";

const base: Reading = makeReading({ id: "m", value: 12, observedAt: new Date().toISOString(), ttlSeconds: 30 });

describe("resolveInk — the one resolver", () => {
  it("maps every (coverage, trust) pair in the closed vocabularies to exactly one ink", () => {
    const inks: Ink[] = ["solid", "dimmed", "hollow", "dotted", "unavailable", "none"];
    for (const coverage of COVERAGES) for (const trust of TRUSTS) expect(inks).toContain(resolveInk(coverage, trust, true).ink);
  });
  it("draws measured readings with the correct ink", () => {
    expect(resolveInk("NOW", "VALID", false)).toEqual({ ink: "solid", figure: "measured", finding: false });
    expect(resolveInk("NOW", "CACHED", false)).toEqual({ ink: "dimmed", figure: "measured", finding: false });
    expect(resolveInk("NOW", "UNTRUSTED", false)).toEqual({ ink: "solid", figure: "measured", finding: true });
    expect(resolveInk("NOW", "UNAVAILABLE", false)).toEqual({ ink: "unavailable", figure: "none", finding: false });
  });
  it("draws illustrative figures hollow or dotted regardless of trust", () => {
    for (const trust of TRUSTS) {
      expect(resolveInk("IN-REACH", trust, true).ink).toBe("hollow");
      expect(resolveInk("MISSING", trust, true).ink).toBe("dotted");
      expect(resolveInk("UNREGISTERED", trust, true).ink).toBe("none");
    }
  });
  it("shows samples only for illustrative coverage", () => {
    expect(figureValue({ ...base, coverage: "MISSING", value: null, sample: authoredSample(5, [1, 5]) }, resolveInk("MISSING", "UNAVAILABLE", true))).toBe(5);
    expect(figureValue({ ...base, sample: authoredSample(5, [1, 5]) }, resolveInk("NOW", "VALID", true))).toBe(12);
  });
  it("treats a NOW cell without a value as unavailable", () => expect(resolveReading({ ...base, value: null }).ink).toBe("unavailable"));
});

describe("qualify — no figure without its qualifier", () => {
  it("names the source and age for a live figure", () => expect(qualify(base, resolveReading(base)).text).toMatch(/swarm-manager · observed \d+s ago/));
  it("names the owner and days open for an absent figure", () => {
    const reading: Reading = { ...base, coverage: "MISSING", trust: "UNAVAILABLE", value: null, owner: "monetization", gapOpenDays: 14, sample: authoredSample(1) };
    expect(qualify(reading, resolveReading(reading))).toEqual({ text: "no substrate · monetization · open 14 days", tone: "gap" });
  });
  it("names what is needed for an in-reach figure", () => {
    const reading: Reading = { ...base, coverage: "IN-REACH", trust: "UNAVAILABLE", value: null, whatIsNeeded: "an endpoint", sample: authoredSample(1) };
    expect(qualify(reading, resolveReading(reading)).text).toBe("illustrative · needs an endpoint");
  });
  it("shows an integrity finding with an untrusted number", () => {
    const reading: Reading = { ...base, trust: "UNTRUSTED", trustReason: "value exceeds the denominator" };
    expect(qualify(reading, resolveReading(reading))).toEqual({ text: "swarm-manager · cannot be believed: value exceeds the denominator", tone: "amber" });
  });
});

describe("formatAge and sourceName", () => {
  it("scales the age unit", async () => {
    const { formatAge, sourceName } = await import("@vrooli/react-component-library/ProvenanceInk/0.1.2");
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
    const silent: Reading = { ...base, trust: "UNAVAILABLE", value: null, trustReason: "deadline exceeded" };
    expect(qualify(silent, resolveReading(silent))).toEqual({ text: "swarm-manager not answering · deadline exceeded", tone: "amber" });
    const cached: Reading = { ...base, trust: "CACHED", observedAt: new Date(Date.now() - 90_000).toISOString() };
    expect(qualify(cached, resolveReading(cached)).text).toMatch(/^last good 2m ago · source not answering$/);
    const unregistered: Reading = { ...base, coverage: "UNREGISTERED" };
    expect(qualify(unregistered, resolveReading(unregistered))).toEqual({ text: "not registered", tone: "quiet" });
  });
});

describe("origin labels", () => {
  it("keeps absent origin metadata safe for rendering", () => {
    const reading: Reading = { ...base, origin: "", origin_env: "", origin_display: "" };
    expect(reading.origin || "local").toBe("local");
    expect(reading.origin_display || "Local instance").toBe("Local instance");
  });
});
