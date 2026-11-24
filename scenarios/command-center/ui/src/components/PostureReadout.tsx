import type { Reading } from "../lib/api";
import type { JSX } from "react";
import { resolveInk } from "@vrooli/react-component-library/ProvenanceInk/0.1.3";
import { ReadingTrustLine } from "./ReadingTrustLine";

type Posture = {
  cashMinor?: number;
  burnMinor?: number;
  revenueMinor?: number;
  runwayMonths?: number;
  runwayAvailable?: boolean;
  gap?: string;
  source?: string;
  ageSeconds?: number;
  currency?: string;
};

const money = (value: unknown, currency?: string): string => {
  if (typeof value !== "number" || !Number.isFinite(value) || !currency?.trim()) return "—";
  try {
    return new Intl.NumberFormat(undefined, { style: "currency", currency: currency.trim(), maximumFractionDigits: 0 }).format(value / 100);
  } catch {
    return `${currency.trim().toUpperCase()} ${(value / 100).toLocaleString(undefined, { maximumFractionDigits: 0 })}`;
  }
};

export function PostureReadout({ reading }: { reading: Reading }): JSX.Element {
  const posture = (reading.value && typeof reading.value === "object" ? reading.value : {}) as Posture;
  const resolution = resolveInk(reading.coverage, reading.trust, false);
  const measured = resolution.figure === "measured";
  const magnitude = (value: unknown) => measured ? money(value, posture.currency) : "—";
  const runwayAvailable = measured && posture.runwayAvailable;
  return (
    <div className="cc-posture" data-reading data-testid="posture-readout" data-trust={reading.trust} data-ink={resolution.ink}>
      <div className="cc-posture-magnitudes">
        <div><span>Revenue</span><strong>{magnitude(posture.revenueMinor)}</strong></div>
        <div><span>Burn</span><strong>{magnitude(posture.burnMinor)}</strong></div>
        <div><span>Cash</span><strong>{magnitude(posture.cashMinor)}</strong></div>
      </div>
      <span className="cc-hero-label">{reading.label}</span>
      <span className="cc-posture-gap" data-testid="posture-gap">{measured && posture.gap ? posture.gap : "Default-alive gap unavailable"}</span>
      <ReadingTrustLine reading={reading} resolution={resolution} />
      <span className="cc-qualifier" data-qualifier data-tone={runwayAvailable ? "live" : "quiet"}>
        {runwayAvailable ? `${posture.runwayMonths ?? "—"} months runway` : "Runway unavailable"}
        {measured && posture.source ? ` · ${posture.source}` : ""}
      </span>
    </div>
  );
}
