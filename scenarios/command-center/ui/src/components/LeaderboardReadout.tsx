import type { Reading } from "../lib/api";
import { qualify, resolveReading } from "@vrooli/react-component-library/ProvenanceInk/0.1.2";

const verdict = (value?: string): string => (value ?? "").replace(/^EXPERIMENT_VERDICT_/, "").replace(/_/g, " ") || "INSUFFICIENT DATA";

export function LeaderboardReadout({ reading }: { reading: Reading }) {
  const resolution = resolveReading(reading);
  const rows = resolution.figure === "measured" ? reading.rows ?? [] : resolution.figure === "sample" ? reading.sample?.rows ?? [] : [];
  const qualifier = qualify(reading, resolution);
  return <section className="cc-leaderboard-readout" data-reading data-kind="leaderboard" data-coverage={reading.coverage} data-trust={reading.trust} data-ink={resolution.ink} data-qualifier={qualifier.text} aria-label={reading.label}>
    <div className="cc-panel-readout__heading">{reading.label}</div>
    {rows.length ? <ol>{rows.map((row) => <li key={row.key} data-leaderboard-arm={row.key}>
      <span>{row.label}{row.is_control ? <em>CONTROL</em> : null}</span>
      <span>{row.value.toLocaleString()} / {(row.denominator ?? 0).toLocaleString()}</span>
      <strong>{row.denominator ? `${((row.rate ?? row.value / row.denominator) * 100).toFixed(1)}%` : "––"}</strong>
      <span className={`cc-verdict cc-verdict-${verdict(row.verdict).toLowerCase().replace(/ /g, "-")}`}>{verdict(row.verdict)}</span>
      <small>{row.cta_trials ? `CTA ${(row.cta_clicks ?? 0).toLocaleString()} / ${row.cta_trials.toLocaleString()}` : null}{["LEADING", "TRAILING"].includes(verdict(row.verdict)) && row.probability ? ` · ${(row.probability * 100).toFixed(0)}% probability` : null}</small>
    </li>)}</ol> : <p className="cc-panel-readout__empty">{qualifier.text}</p>}
    <span className="cc-qualifier" data-qualifier data-tone={qualifier.tone}>{qualifier.text}</span>
  </section>;
}
