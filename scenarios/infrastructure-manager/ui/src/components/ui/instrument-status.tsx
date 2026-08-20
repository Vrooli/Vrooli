import { TRUST_VERDICTS, type BandVerdict, type RatioConfidenceValue, type TrustTripleValue, type TrustVerdict } from "../../theme/instrument";
import { bandToken, trustToken } from "../../theme/instrument";

export function TrustTriple({ value }: { value: TrustTripleValue }) {
  const entries = Object.entries(value.distribution).filter(([, count]) => (count ?? 0) > 0) as [TrustVerdict, number][];
  return (
    <section className="instrument-triple" aria-label="Trust distribution">
      <div className="instrument-triple__counts">
        {entries.length === 0 ? <span>no verdicts</span> : entries.map(([verdict, count]) => (
          <span className={`status-token status-token--${trustToken(verdict).tone}`} key={verdict}>
            <span aria-hidden="true">{trustToken(verdict).mark}</span> {count} {trustToken(verdict).label}
          </span>
        ))}
      </div>
      <div className="instrument-triple__denominator">
        {value.checked} checked <span aria-hidden="true">of</span> {value.total} readings
      </div>
    </section>
  );
}

export function RatioConfidence({ value }: { value: RatioConfidenceValue }) {
  const ratio = value.ratio === null ? "—" : `${Math.round(value.ratio * 100)}%`;
  return (
    <section className="ratio-confidence" aria-label={`Coverage ${ratio}, confidence ${value.confidence}`}>
      <strong className="ratio-confidence__ratio">{ratio}</strong>
      <span className={`confidence confidence--${value.confidence.toLowerCase()}`}>{value.confidence}</span>
      <span className="ratio-confidence__rationale">{value.rationale}</span>
    </section>
  );
}

export function StatusToken({ verdict }: { verdict: TrustVerdict | BandVerdict }) {
  const token = verdict in TRUST_VERDICTS
    ? trustToken(verdict as TrustVerdict)
    : bandToken(verdict as BandVerdict);
  return <span className={`status-token status-token--${token.tone}`}><span aria-hidden="true">{token.mark}</span> {token.label}</span>;
}
