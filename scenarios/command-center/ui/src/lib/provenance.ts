import type { Coverage, Reading, Trust } from "./api";

/**
 * The one place that turns a (coverage, trust) pair into a rendering.
 * Every figure on every surface goes through here. A second resolver anywhere
 * in this codebase is a defect: two sources of truth about honesty diverge.
 */
export type Ink = "solid" | "dimmed" | "hollow" | "dotted" | "unavailable" | "none";

export interface Resolution {
  ink: Ink;
  /** Which number the figure shows, if any. */
  figure: "measured" | "sample" | "none";
  /** Set when the number is shown alongside an integrity finding. */
  finding: boolean;
}

export function resolveInk(coverage: Coverage, trust: Trust, hasSample: boolean): Resolution {
  switch (coverage) {
    case "NOW":
      switch (trust) {
        case "VALID":
          return { ink: "solid", figure: "measured", finding: false };
        case "CACHED":
          return { ink: "dimmed", figure: "measured", finding: false };
        case "UNTRUSTED":
          return { ink: "solid", figure: "measured", finding: true };
        case "UNAVAILABLE":
          return { ink: "unavailable", figure: "none", finding: false };
      }
      break;
    case "IN-REACH":
      return { ink: "hollow", figure: hasSample ? "sample" : "none", finding: false };
    case "MISSING":
      return { ink: "dotted", figure: hasSample ? "sample" : "none", finding: false };
    case "UNREGISTERED":
      return { ink: "none", figure: "none", finding: false };
  }
  return { ink: "none", figure: "none", finding: false };
}

export const COVERAGES: Coverage[] = ["NOW", "IN-REACH", "MISSING", "UNREGISTERED"];
export const TRUSTS: Trust[] = ["VALID", "CACHED", "UNAVAILABLE", "UNTRUSTED"];

export function resolveReading(reading: Reading): Resolution {
  const measured = typeof reading.value === "number";
  const resolution = resolveInk(reading.coverage, reading.trust, reading.sample !== null);
  if (resolution.figure === "measured" && !measured) {
    return { ink: "unavailable", figure: "none", finding: false };
  }
  return resolution;
}

export function figureValue(reading: Reading, resolution: Resolution): number | null {
  if (resolution.figure === "measured") return reading.value;
  if (resolution.figure === "sample") return reading.sample?.value ?? null;
  return null;
}

/** Whether the figure being shown is an authored illustration rather than a measurement. */
export const isIllustrative = (resolution: Resolution): boolean => resolution.figure === "sample";

export interface Qualifier {
  /** The one line every figure carries. */
  text: string;
  /** Tone reinforces material; it never carries the state alone. */
  tone: "live" | "amber" | "gap" | "quiet";
}

export function formatAge(observedAt: string | null, now = Date.now()): string {
  if (!observedAt) return "unknown age";
  const seconds = Math.max(0, Math.round((now - Date.parse(observedAt)) / 1000));
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 48) return `${hours}h ago`;
  return `${Math.round(hours / 24)}d ago`;
}

export function sourceName(reading: Reading): string {
  const binding = reading.source.binding ?? "";
  return binding.replace(/^scenario:/, "") || reading.source.team || "unknown source";
}

export function qualify(reading: Reading, resolution: Resolution, now = Date.now()): Qualifier {
  const owner = reading.owner ?? reading.source.team ?? "owner unknown";
  const days = reading.gapOpenDays ?? 0;
  switch (resolution.ink) {
    case "solid":
      if (resolution.finding) {
        return { text: `${sourceName(reading)} · cannot be believed: ${reading.trustReason ?? "integrity finding"}`, tone: "amber" };
      }
      return { text: `${sourceName(reading)} · observed ${formatAge(reading.observedAt, now)}`, tone: "live" };
    case "dimmed":
      return { text: `last good ${formatAge(reading.observedAt, now)} · ${reading.trustReason ?? "source not answering"}`, tone: "amber" };
    case "unavailable":
      return { text: `${sourceName(reading)} not answering · ${reading.trustReason ?? "no value asserted"}`, tone: "amber" };
    case "hollow":
      return { text: `illustrative · needs ${reading.whatIsNeeded ?? "a pipeline"}`, tone: "gap" };
    case "dotted":
      return { text: `no substrate · ${owner} · open ${days} ${days === 1 ? "day" : "days"}`, tone: "gap" };
    default:
      return { text: "not registered", tone: "quiet" };
  }
}

/** Short label for a legend or a chip. */
export const INK_LABELS: Record<Exclude<Ink, "none">, string> = {
  solid: "measured",
  dimmed: "cached",
  hollow: "illustrative · in reach",
  dotted: "illustrative · missing",
  unavailable: "not answering",
};
