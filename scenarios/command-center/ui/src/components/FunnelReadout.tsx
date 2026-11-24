import type { Reading } from "../lib/api";
import { resolveReading } from "@vrooli/react-component-library/ProvenanceInk/0.1.3";
import { ReadingTrustLine } from "./ReadingTrustLine";

export function FunnelReadout({ reading }: { reading: Reading }) {
  const resolution = resolveReading(reading);
  const rows = resolution.figure === "measured" ? reading.rows ?? [] : resolution.figure === "sample" ? reading.sample?.rows ?? [] : [];
  return <section className="cc-funnel-readout" data-reading data-kind="funnel" data-coverage={reading.coverage} data-trust={reading.trust} data-ink={resolution.ink} aria-label={reading.label}>
    <div className="cc-panel-readout__heading">{reading.label}</div>
    {rows.length ? <ol>{rows.map((row) => <li key={row.key} data-funnel-step={row.key}>
      <span className="cc-funnel-readout__label">{row.label}</span>
      <strong>{row.value.toLocaleString()}</strong>
      <span>{row.denominator ? `${row.value.toLocaleString()} of ${row.denominator.toLocaleString()}` : row.key === "visitors" ? "base population" : "denominator unavailable"}</span>
      {row.rate !== undefined && row.key !== "visitors" ? <small>{(row.rate * 100).toFixed(1)}% of prior step</small> : null}
    </li>)}</ol> : <p className="cc-panel-readout__empty">No measured funnel steps in this observation.</p>}
    <ReadingTrustLine reading={reading} resolution={resolution} />
  </section>;
}
