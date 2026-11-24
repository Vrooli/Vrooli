import type { PanelRow, Reading } from "../lib/api";
import { RollingNumber } from "@vrooli/react-component-library/RollingNumber/0.1.6";
import { AutoScroll } from "./AutoScroll";
import { resolveReading, figureValue } from "@vrooli/react-component-library/ProvenanceInk/0.1.3";
import { panelEmptyMessage } from "../lib/panelEmptyState";
import { ReadingTrustLine } from "./ReadingTrustLine";

type Props = { reading: Reading; maxRows?: number };

/** A panel longer than this is a table, not a readout; the rest belong to the Focus view. */
const MAX_ROWS = 24;

/** Compact ranked figure for panel readings. Material, not hue, carries row provenance. */
export function PanelReadout({ reading, maxRows = MAX_ROWS }: Props) {
  const resolution = resolveReading(reading);
  const sampled = resolution.figure === "sample";
  const untrustedCurrent = reading.coverage === "NOW" && reading.trust === "UNTRUSTED";
  const sourceRows: PanelRow[] = untrustedCurrent ? [] : resolution.figure === "measured" ? reading.rows ?? [] : sampled ? reading.sample?.rows ?? [] : [];
  const rows = sourceRows
    .slice(0, maxRows)
    .map((row): PanelRow => sampled ? { ...row, ink: row.ink ?? "dotted" } : reading.trust === "CACHED" ? { ...row, ink: row.ink ?? "reduced" } : row);
  return (
    <section className="cc-panel-readout" aria-label={reading.label} data-kind={reading.kind ?? "panel"} data-reading data-coverage={reading.coverage} data-trust={reading.trust} data-ink={resolution.ink} data-provenance={resolution.figure}>
      <div className="cc-panel-readout__heading">{reading.label}</div>
      {rows.length === 0 ? (
        <div className="cc-panel-readout__empty" aria-label="No observations">
          <span className="cc-panel-readout__empty-mark" aria-hidden="true">∅</span>
          <span>{panelEmptyMessage(reading.id, untrustedCurrent)}</span>
          {figureValue(reading, resolution) === null && reading.value !== null ? <small>The last good breakdown is withheld after its observation expires.</small> : null}
        </div>
      ) : (
        <AutoScroll className="cc-panel-readout__rows" rowSelector=".cc-panel-readout__row" label={`${reading.label} rows`}>
          {rows.map((row: PanelRow) => (
            <div className={`cc-panel-readout__row cc-panel-readout__row--${row.ink ?? "solid"}`} key={row.key}>
              <div className="cc-panel-readout__label">{row.label}</div>
              <div className="cc-panel-readout__value"><RollingNumber value={row.value} format="compact" ink={row.ink === "hollow" ? "hollow" : row.ink === "dotted" ? "dotted" : row.ink === "reduced" ? "dimmed" : "solid"} scale="display" /></div>
                {row.detail ? <small className="cc-panel-readout__detail">{row.detail}</small> : null}
              <div className="cc-panel-readout__bar" style={{ width: `${Math.max(0, Math.min(100, row.share * 100))}%` }} />
            </div>
          ))}
        </AutoScroll>
      )}
      <ReadingTrustLine reading={reading} resolution={resolution} />
    </section>
  );
}
