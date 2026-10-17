import type { Reading } from "../lib/api";

type Posture = {
  cashMinor?: number;
  burnMinor?: number;
  revenueMinor?: number;
  runwayMonths?: number;
  runwayAvailable?: boolean;
  gap?: string;
  source?: string;
  ageSeconds?: number;
};

const money = (value: unknown): string => typeof value === "number" ? `$${(value / 100).toLocaleString(undefined, { maximumFractionDigits: 0 })}` : "—";

export function PostureReadout({ reading }: { reading: Reading }): JSX.Element {
  const posture = (reading.value && typeof reading.value === "object" ? reading.value : {}) as Posture;
  const measured = reading.trust === "VALID" || reading.trust === "CACHED";
  return (
    <div className="cc-posture" data-reading data-testid="posture-readout" data-ink={measured ? "solid" : "unavailable"}>
      <div className="cc-posture-magnitudes">
        <div><span>Revenue</span><strong>{money(posture.revenueMinor)}</strong></div>
        <div><span>Burn</span><strong>{money(posture.burnMinor)}</strong></div>
        <div><span>Cash</span><strong>{money(posture.cashMinor)}</strong></div>
      </div>
      <span className="cc-hero-label">{reading.label}</span>
      <span className="cc-posture-gap" data-testid="posture-gap">{posture.gap ?? "Default-alive gap unavailable"}</span>
      <span className="cc-qualifier" data-qualifier data-tone={posture.runwayAvailable ? "live" : "quiet"}>
        {posture.runwayAvailable ? `${posture.runwayMonths ?? "—"} months runway` : "Runway unavailable"}
        {posture.source ? ` · ${posture.source}` : ""}
      </span>
    </div>
  );
}
