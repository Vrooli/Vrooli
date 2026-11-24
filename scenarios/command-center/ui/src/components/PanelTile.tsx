import type { PanelRow, Reading } from "../lib/api";
import { resolveReading } from "@vrooli/react-component-library/ProvenanceInk/0.1.3";
import { TileQualifier } from "./TileQualifier";
import { panelEmptyMessage } from "../lib/panelEmptyState";

const TILE_ROWS = 3;
const compact = new Intl.NumberFormat(undefined, { notation: "compact", maximumFractionDigits: 1 });

/**
 * A ranked panel in the supporting strip. A panel has no single figure, so the
 * tile shows its leading rows instead of a dash; authored rows stay marked.
 */
export function PanelTile({ reading, showOrigin = false }: { reading: Reading; showOrigin?: boolean }) {
  const resolution = resolveReading(reading);
  const sampled = resolution.figure === "sample";
  const untrustedCurrent = reading.coverage === "NOW" && reading.trust === "UNTRUSTED";
  const rows: PanelRow[] = untrustedCurrent ? [] : resolution.figure === "measured" ? reading.rows ?? [] : sampled ? reading.sample?.rows ?? [] : [];
  return (
    <li className="cc-reading cc-panel-tile" data-testid="panel-tile" data-kind="panel" data-reading data-metric-id={reading.id} data-origin-env={reading.origin_env} data-ink={resolution.ink} data-provenance={sampled ? "sample" : rows.length ? "measured" : "absent"} data-trust={reading.trust} title={reading.description}>
      <span className="cc-reading-label">{reading.label}</span>
      {rows.length ? (
        <ol className="cc-panel-tile__rows">
          {rows.slice(0, TILE_ROWS).map((row) => (
            <li key={row.key} className={`cc-panel-tile__row${reading.trust === "CACHED" ? " cc-panel-tile__row--cached" : ""}`}><span>{row.label}</span><span>{compact.format(row.value)}</span></li>
          ))}
        </ol>
      ) : <span className="cc-panel-tile__empty"><span className="cc-panel-tile__empty-mark" aria-hidden="true">∅</span>{panelEmptyMessage(reading.id, untrustedCurrent)}</span>}
      {rows.length > TILE_ROWS ? <span className="cc-panel-tile__more">+{rows.length - TILE_ROWS} more</span> : null}
      <TileQualifier reading={reading} showOrigin={showOrigin} />
    </li>
  );
}
