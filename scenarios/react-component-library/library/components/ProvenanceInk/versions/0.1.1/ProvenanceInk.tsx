/**
 * @libraryId react-component-library:ProvenanceInk
 * @displayName ProvenanceInk
 * @description The honesty system for ambient displays: one resolver turns a reading's coverage and trust into one of four material inks — solid, dimmed, hollow, dotted — plus the ink stylesheet, chip and legend swatch.
 * @version 0.1.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:ProvenanceInk */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import type { ReactNode } from "react";

/**
 * The honesty system for ambient displays: one resolver turns a reading's
 * coverage and trust into an ink, and the inks are materials — solid,
 * dimmed, hollow, dotted — never colours. A second resolver anywhere is a
 * second source of truth about honesty, and they will diverge.
 */
export type Coverage = "NOW" | "IN-REACH" | "MISSING" | "UNREGISTERED";
export type Trust = "VALID" | "CACHED" | "UNAVAILABLE" | "UNTRUSTED";
export type Ink = "solid" | "dimmed" | "hollow" | "dotted" | "unavailable" | "none";

/** The fields the resolver reads; any richer reading satisfies it structurally. */
export interface ProvenanceReading {
  coverage: Coverage;
  trust: Trust;
  trustReason?: string;
  value: number | null;
  observedAt: string | null;
  owner?: string | null;
  whatIsNeeded?: string | null;
  gapOpenDays?: number | null;
  sample: { value: number; series: number[]; basis: string } | null;
  source?: { team?: string; binding?: string };
}

export interface Resolution {
  ink: Ink;
  /** Which number the figure shows, if any. */
  figure: "measured" | "sample" | "none";
  /** Set when the number is shown alongside an integrity finding. */
  finding: boolean;
}

export const COVERAGES: Coverage[] = ["NOW", "IN-REACH", "MISSING", "UNREGISTERED"];
export const TRUSTS: Trust[] = ["VALID", "CACHED", "UNAVAILABLE", "UNTRUSTED"];

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

export function resolveReading(reading: ProvenanceReading): Resolution {
  const resolution = resolveInk(reading.coverage, reading.trust, reading.sample !== null);
  if (resolution.figure === "measured" && typeof reading.value !== "number") {
    return { ink: "unavailable", figure: "none", finding: false };
  }
  return resolution;
}

export function figureValue(reading: ProvenanceReading, resolution: Resolution): number | null {
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

export function sourceName(reading: ProvenanceReading): string {
  const binding = reading.source?.binding ?? "";
  return binding.replace(/^scenario:/, "") || reading.source?.team || "unknown source";
}

export function qualify(
  reading: ProvenanceReading,
  resolution: Resolution,
  now = Date.now(),
): Qualifier {
  const owner = reading.owner ?? reading.source?.team ?? "owner unknown";
  const days = reading.gapOpenDays ?? 0;
  switch (resolution.ink) {
    case "solid":
      if (resolution.finding) {
        return {
          text: `${sourceName(reading)} · cannot be believed: ${reading.trustReason ?? "integrity finding"}`,
          tone: "amber",
        };
      }
      return {
        text: `${sourceName(reading)} · observed ${formatAge(reading.observedAt, now)}`,
        tone: "live",
      };
    case "dimmed":
      return {
        text: `last good ${formatAge(reading.observedAt, now)} · ${reading.trustReason ?? "source not answering"}`,
        tone: "amber",
      };
    case "unavailable":
      return {
        text: `${sourceName(reading)} not answering · ${reading.trustReason ?? "no value asserted"}`,
        tone: "amber",
      };
    case "hollow":
      return { text: `illustrative · needs ${reading.whatIsNeeded ?? "a pipeline"}`, tone: "gap" };
    case "dotted":
      return {
        text: `no substrate · ${owner} · open ${days} ${days === 1 ? "day" : "days"}`,
        tone: "gap",
      };
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

const styles = `
  [data-figure][data-ink="solid"] { color: var(--color-foreground, #e8ecf3); text-shadow: 0 0 40px color-mix(in srgb, var(--glow-primary, rgba(51,214,255,.5)) 40%, transparent); }
  [data-figure][data-ink="dimmed"] { color: color-mix(in srgb, var(--color-foreground, #e8ecf3) 55%, var(--color-background, #05070e)); text-shadow: none; }
  [data-figure][data-ink="hollow"] [data-rcl-figure-digits],
  [data-figure][data-ink="hollow"] [data-rcl-figure-affix] { color: transparent; -webkit-text-stroke: max(1.5px, 0.014em) color-mix(in srgb, var(--color-foreground, #e8ecf3) 78%, var(--provenance-sample, #b7a6ff)); }
  [data-figure][data-ink="dotted"] [data-rcl-figure-digits],
  [data-figure][data-ink="dotted"] [data-rcl-figure-affix] { color: transparent; -webkit-text-stroke: 0.7px color-mix(in srgb, var(--color-foreground, #e8ecf3) 40%, var(--provenance-sample, #b7a6ff)); background-image: radial-gradient(circle, color-mix(in srgb, var(--color-foreground, #e8ecf3) 92%, var(--provenance-sample, #b7a6ff)) 40%, transparent 44%); background-size: clamp(5px, 0.048em, 16px) clamp(5px, 0.048em, 16px); -webkit-background-clip: text; background-clip: text; }
  [data-figure][data-ink="unavailable"] [data-rcl-figure-digits],
  [data-figure][data-ink="none"] [data-rcl-figure-digits] { color: transparent; -webkit-text-stroke: max(1px, 0.01em) color-mix(in srgb, var(--color-warning, #f5b544) 55%, var(--color-muted-foreground, #94a3b8)); }
  [data-figure][data-ink="none"] [data-rcl-figure-digits] { -webkit-text-stroke-color: var(--color-muted-foreground, #94a3b8); }
  [data-rcl-ink-mark] { display: inline-block; margin-right: 0.6em; padding: 0.18em 0.55em; border: 1px solid currentColor; border-radius: var(--radius-pill, 9999px); font: 600 var(--text-caption, 600 0.7rem/1.3 ui-monospace, monospace); letter-spacing: 0.14em; text-transform: uppercase; vertical-align: middle; }
  [data-rcl-ink-mark][data-ink="solid"] { background: currentColor; color: var(--color-background, #05070e); }
  [data-rcl-ink-mark][data-ink="hollow"] { border-style: solid; color: color-mix(in srgb, var(--color-foreground, #e8ecf3) 70%, var(--provenance-sample, #b7a6ff)); }
  [data-rcl-ink-mark][data-ink="dotted"] { border-style: dotted; border-width: 1.5px; color: color-mix(in srgb, var(--color-foreground, #e8ecf3) 70%, var(--provenance-sample, #b7a6ff)); }
  [data-rcl-ink-mark][data-ink="unavailable"] { border-style: dashed; color: var(--color-warning, #f5b544); }
  [data-rcl-ink-swatch] { display: inline-flex; width: 1.6em; height: 1.6em; align-items: center; justify-content: center; margin-right: 0.45em; font: 700 1.05em system-ui, sans-serif; font-variant-numeric: tabular-nums; }
  [data-rcl-ink-swatch][data-ink="solid"] { color: var(--color-foreground, #e8ecf3); }
  [data-rcl-ink-swatch][data-ink="dimmed"] { color: color-mix(in srgb, var(--color-foreground, #e8ecf3) 55%, var(--color-background, #05070e)); }
  [data-rcl-ink-swatch][data-ink="hollow"] { color: transparent; -webkit-text-stroke: 1.2px color-mix(in srgb, var(--color-foreground, #e8ecf3) 78%, var(--provenance-sample, #b7a6ff)); }
  [data-rcl-ink-swatch][data-ink="dotted"] { color: transparent; -webkit-text-stroke: 0.5px color-mix(in srgb, var(--color-foreground, #e8ecf3) 34%, var(--provenance-sample, #b7a6ff)); background-image: radial-gradient(circle, var(--color-foreground, #e8ecf3) 34%, transparent 38%); background-size: 3px 3px; -webkit-background-clip: text; background-clip: text; }
`;

/** Mount once near any figure; the ink rules are keyed on data attributes so they compose with any layout. */
export function ProvenanceInkStyles() {
  return <StyleSheet name="provenance-ink-1" css={styles} />;
}

/** A small chip naming an ink, drawn in that ink's material. */
export const InkMark = withClassName(function InkMark({
  ink,
  children,
  className,
}: {
  ink: Exclude<Ink, "none">;
  children?: ReactNode;
  className?: string;
}) {
  return (
    <>
      <ProvenanceInkStyles />
      <span data-rcl-ink-mark data-ink={ink} className={className}>
        {children ?? INK_LABELS[ink]}
      </span>
    </>
  );
});

/** A legend glyph: the digit 8 in one ink, so a legend shows the materials themselves. */
export const InkSwatch = withClassName(function InkSwatch({
  ink,
  className,
}: {
  ink: Exclude<Ink, "none">;
  className?: string;
}) {
  return (
    <>
      <ProvenanceInkStyles />
      <span data-rcl-ink-swatch data-ink={ink} className={className} aria-hidden="true">
        8
      </span>
    </>
  );
});
