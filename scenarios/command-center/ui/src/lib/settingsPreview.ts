// Preview helpers for the settings surface. They turn the editable catalogs into
// the exact inputs the live board engine consumes, so the settings preview is the
// same scene engine driven by authored signal samples — not a mock.
import type { CSSProperties } from "react";
import type { CatalogEntry, Coverage, Reading, Sample } from "./api";

const str = (value: unknown): string => (typeof value === "string" ? value : "");

const shapeToKind = (shape: unknown): Reading["kind"] => (shape === "rows" ? "panel" : "scalar");

/** One preview reading per signal, valued from its authored sample. Bindings resolve
 *  a slot to a signal id; the scene then reads that reading exactly as on the board. */
export function sampleReadings(signals: CatalogEntry[]): Reading[] {
  return signals.map((signal) => {
    const sample = (signal.sample && typeof signal.sample === "object" ? signal.sample : null) as Sample | null;
    const value = typeof sample?.value === "number" ? sample.value : typeof signal.value === "number" ? signal.value : null;
    return {
      id: signal.id,
      label: str(signal.label) || signal.id,
      description: str(signal.description) || undefined,
      unit: str(signal.unit) || undefined,
      format: str(signal.format) || undefined,
      source: {},
      coverage: (str(signal.coverage) || "NOW") as Coverage,
      trust: "VALID",
      empirical: "HIT",
      value,
      kind: shapeToKind(signal.shape),
      rows: sample?.rows,
      observedAt: null,
      ttlSeconds: 60,
      target: null,
      owner: null,
      whatIsNeeded: null,
      firstObservedMissing: null,
      gapOpenDays: null,
      sample,
      prediction: null,
      origin: "sample",
      origin_env: "local",
      origin_display: "Sample",
    } satisfies Reading;
  });
}

/** A theme's paint tokens as inline CSS variables, so the preview stage recolors to
 *  the edited theme without touching the global board (AmbientCanvas reads these vars). */
export function themeStyle(theme: CatalogEntry | undefined): CSSProperties {
  const tokens = theme?.tokens && typeof theme.tokens === "object" ? (theme.tokens as Record<string, unknown>) : {};
  const style: Record<string, string> = {};
  for (const [name, value] of Object.entries(tokens)) {
    if (name.startsWith("--") && typeof value === "string") style[name] = value;
  }
  return style as CSSProperties;
}
