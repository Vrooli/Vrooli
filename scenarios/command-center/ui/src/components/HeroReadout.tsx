import { forwardRef } from "react";
import type { Reading } from "../lib/api";
import { InkMark, figureValue, isIllustrative, qualify, resolveReading } from "@vrooli/react-component-library/ProvenanceInk/0.1.2";
import { RollingNumber } from "@vrooli/react-component-library/RollingNumber/0.1.3";
import { FreshnessArc } from "@vrooli/react-component-library/FreshnessArc/0.1.2";

interface HeroReadoutProps {
  reading: Reading | null;
  /** In hide mode a room with nothing measured says so instead of drawing an illustration. */
  emptyReason?: string;
}

/** The one enormous number a room is composed around, with the qualifier that makes it honest. */
export const HeroReadout = forwardRef<HTMLDivElement, HeroReadoutProps>(function HeroReadout({ reading, emptyReason }, ref) {
  if (!reading) {
    return (
      <div ref={ref} className="cc-hero" data-reading data-ink="none" data-provenance="none">
        <RollingNumber value={null} ink="none" scale="wall" placeholder="—" />
        <span className="cc-hero-label">No measured reading</span>
        <span className="cc-qualifier" data-qualifier data-tone="quiet">{emptyReason ?? "Nothing in this room is measured yet."}</span>
      </div>
    );
  }
  const resolution = resolveReading(reading);
  const value = figureValue(reading, resolution);
  const qualifier = qualify(reading, resolution);
  const live = resolution.ink === "solid" || resolution.ink === "dimmed";
  return (
    <div ref={ref} className="cc-hero" data-reading data-metric-id={reading.id} data-ink={resolution.ink} data-provenance={isIllustrative(resolution) ? "sample" : resolution.figure === "measured" ? "measured" : "absent"} data-coverage={reading.coverage} data-trust={reading.trust} data-empirical={reading.empirical}>
      <RollingNumber value={value} format={reading.format} unit={reading.unit} ink={resolution.ink} scale="wall" placeholder={resolution.ink === "unavailable" ? "––" : "—"} />
      {live ? <FreshnessArc observedAt={reading.observedAt} ttlSeconds={reading.ttlSeconds} cached={resolution.ink === "dimmed"} /> : null}
      <span className="cc-hero-label">{reading.label}</span>
      <span className="cc-qualifier" data-qualifier data-tone={qualifier.tone}>
        {isIllustrative(resolution) && resolution.ink !== "none" ? <InkMark ink={resolution.ink}>{resolution.ink === "hollow" ? "in reach" : "missing"}</InkMark> : null}
        {qualifier.text}
      </span>
      {reading.empirical !== "NONE" ? <span className="cc-empirical" data-empirical={reading.empirical}>prediction {reading.empirical.toLowerCase()}</span> : null}
    </div>
  );
});
