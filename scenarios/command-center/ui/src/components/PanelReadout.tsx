import type { PanelRow, Reading } from "../lib/api";
import { RollingNumber } from "@vrooli/react-component-library/RollingNumber/0.1.5";

type Props = { reading: Reading; maxRows?: number };

/** Compact ranked figure for panel readings. Material, not hue, carries row provenance. */
export function PanelReadout({ reading, maxRows = 6 }: Props) {
  // A panel can have measured rows or an explicitly authored sample. The API
  // keeps those separate so provenance remains honest, but the display should
  // still show the sample's actual table instead of falling through to an
  // empty placeholder.
  const sampled = !reading.rows?.length && Boolean(reading.sample?.rows?.length);
  const rows = (reading.rows ?? reading.sample?.rows ?? [])
    .slice(0, maxRows)
    .map((row) => sampled ? { ...row, ink: row.ink ?? "dotted" as const } : row);
  return (
    <section className="cc-panel-readout" aria-label={reading.label} data-kind="panel">
      <div className="cc-panel-readout__heading">{reading.label}</div>
      {rows.length === 0 ? (
        <div className="cc-panel-readout__empty" aria-label="No observations">No rows available</div>
      ) : rows.map((row: PanelRow) => (
        <div className={`cc-panel-readout__row cc-panel-readout__row--${row.ink ?? "solid"}`} key={row.key}>
          <div className="cc-panel-readout__label">{row.label}</div>
          <div className="cc-panel-readout__value"><RollingNumber value={row.value} format="compact" ink={row.ink === "hollow" ? "hollow" : row.ink === "dotted" ? "dotted" : row.ink === "reduced" ? "dimmed" : "solid"} scale="display" /></div>
          <div className="cc-panel-readout__bar" style={{ width: `${Math.max(0, Math.min(100, row.share * 100))}%` }} />
        </div>
      ))}
    </section>
  );
}
