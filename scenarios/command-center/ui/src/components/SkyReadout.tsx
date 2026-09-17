import { forwardRef } from "react";
import { InkMark } from "@vrooli/react-component-library/ProvenanceInk/0.1.2";
import { RollingNumber } from "@vrooli/react-component-library/RollingNumber/0.1.6";
import type { Constellation } from "../lib/api";
import { displayFormat } from "../lib/format";
import { tallySky } from "../lib/sky";

/**
 * Panorama's hero: how much of the whole board is measured, counted by the
 * board from its own readings. Two counts and the reasons for the rest, never
 * a percentage.
 */
export const SkyReadout = forwardRef<HTMLDivElement, { constellations: Constellation[] }>(function SkyReadout({ constellations }, ref) {
  const tally = tallySky(constellations);
  return (
    <div ref={ref} className="cc-hero cc-sky" data-testid="sky-readout" data-reading data-ink="solid" data-provenance="measured">
      <RollingNumber value={tally.measured} format={displayFormat(tally.measured, "integer")} unit="count" ink="solid" scale="wall" placeholder="—" />
      <span className="cc-hero-label">Signals measured</span>
      <span className="cc-qualifier" data-qualifier data-tone="live">
        of {tally.total} registered across {tally.rooms} {tally.rooms === 1 ? "room" : "rooms"}
        {tally.cached > 0 ? ` · ${tally.cached.toString()} from cache` : ""}
      </span>
      <ul className="cc-sky-tally" data-testid="sky-tally" aria-label="Signals not yet measured">
        <li data-state="failing"><InkMark ink="unavailable">{tally.failing}</InkMark>sensor failing</li>
        <li data-state="in-reach"><InkMark ink="hollow">{tally.inReach}</InkMark>in reach</li>
        <li data-state="missing"><InkMark ink="dotted">{tally.missing}</InkMark>no substrate</li>
      </ul>
    </div>
  );
});
