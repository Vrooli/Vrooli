/**
 * @libraryId react-component-library:HeroReadout
 * @displayName HeroReadout
 * @description Provenance-qualified dominant reading for ambient displays
 * @version 0.1.1
 * @tags ["data-display","ambient","provenance"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { forwardRef } from "react";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";
import type { ProvenanceReading } from "@vrooli/react-component-library/ProvenanceInk/0";
import {
  InkMark,
  figureValue,
  isIllustrative,
  qualify,
  resolveReading,
} from "@vrooli/react-component-library/ProvenanceInk/0";
import { RollingNumber } from "@vrooli/react-component-library/RollingNumber/0";
import { FreshnessArc } from "@vrooli/react-component-library/FreshnessArc/0";

export interface HeroReading extends ProvenanceReading {
  id?: string;
  label: string;
  format?: string;
  unit?: string;
  ttlSeconds?: number;
  empirical?: string;
}

interface HeroReadoutProps {
  reading: HeroReading | null;
  /** In hide mode a room with nothing measured says so instead of drawing an illustration. */
  emptyReason?: string;
}

const styles = `
[data-rcl-hero-readout] { display: grid; justify-items: center; gap: var(--space-xs, .75rem); min-inline-size: 0; color: var(--color-foreground, #e8ecf3); text-align: center; }
[data-rcl-hero-label] { font: var(--text-title, 700 1.5rem/1.2 var(--font-sans, sans-serif)); letter-spacing: .08em; text-transform: uppercase; }
[data-rcl-hero-qualifier] { max-inline-size: 70ch; color: var(--color-muted-foreground, #94a3b8); font: var(--text-label, 500 .8rem/1.25 var(--font-sans, sans-serif)); }
[data-rcl-hero-empirical] { color: var(--color-muted-foreground, #94a3b8); font: var(--text-caption, 600 .7rem/1.3 var(--font-mono, monospace)); letter-spacing: .08em; text-transform: uppercase; }
`;

/** The one enormous number a room is composed around, with the qualifier that makes it honest. */
export const HeroReadout = withClassName(
  forwardRef<HTMLDivElement, HeroReadoutProps>(function HeroReadout({ reading, emptyReason }, ref) {
    if (!reading) {
      return (
        <>
          <StyleSheet name="hero-readout-0-1-0" css={styles} />
          <div ref={ref} data-rcl-hero-readout data-reading data-ink="none" data-provenance="none">
            <RollingNumber value={null} ink="none" scale="wall" placeholder="—" />
            <span data-rcl-hero-label>No measured reading</span>
            <span data-rcl-hero-qualifier data-qualifier data-tone="quiet">
              {emptyReason ?? "Nothing in this room is measured yet."}
            </span>
          </div>
        </>
      );
    }
    const resolution = resolveReading(reading);
    const value = figureValue(reading, resolution);
    const qualifier = qualify(reading, resolution);
    const live = resolution.ink === "solid" || resolution.ink === "dimmed";
    return (
      <>
        <StyleSheet name="hero-readout-0-1-0" css={styles} />
        <div
          ref={ref}
          data-rcl-hero-readout
          data-reading
          data-metric-id={reading.id ?? undefined}
          data-ink={resolution.ink}
          data-provenance={
            isIllustrative(resolution)
              ? "sample"
              : resolution.figure === "measured"
                ? "measured"
                : "absent"
          }
          data-coverage={reading.coverage}
          data-trust={reading.trust}
          data-empirical={reading.empirical}
        >
          <RollingNumber
            value={value}
            format={reading.format}
            unit={reading.unit}
            ink={resolution.ink}
            scale="wall"
            placeholder={resolution.ink === "unavailable" ? "––" : "—"}
          />
          {live ? (
            <FreshnessArc
              observedAt={reading.observedAt}
              ttlSeconds={reading.ttlSeconds}
              cached={resolution.ink === "dimmed"}
            />
          ) : null}
          <span data-rcl-hero-label>{reading.label}</span>
          <span data-rcl-hero-qualifier data-qualifier data-tone={qualifier.tone}>
            {isIllustrative(resolution) && resolution.ink !== "none" ? (
              <InkMark ink={resolution.ink}>
                {resolution.ink === "hollow" ? "in reach" : "missing"}
              </InkMark>
            ) : null}
            {qualifier.text}
          </span>
          {reading.empirical && reading.empirical !== "NONE" ? (
            <span data-rcl-hero-empirical data-empirical={reading.empirical}>
              prediction {reading.empirical.toLowerCase()}
            </span>
          ) : null}
        </div>
      </>
    );
  }),
);
