import type { PanelRow, Reading } from "../lib/api";
import { RollingNumber } from "@vrooli/react-component-library/RollingNumber/0.1.5";

type Props = { reading: Reading; maxRows?: number };

/** Compact ranked figure for panel readings. Material, not hue, carries row provenance. */
export function PanelReadout({ reading, maxRows = 6 }: Props) {
  const rows = (reading.rows ?? []).slice(0, maxRows);
  return (
    <section className="cc-panel-readout" aria-label={reading.label} data-kind="panel">
      <div className="cc-panel-readout__heading">{reading.label}</div>
      {rows.length === 0 ? (
        <div className="cc-panel-readout__placeholder" aria-label="No observations">···</div>
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
