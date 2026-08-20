/**
 * The instrument-panel status vocabulary is intentionally closed. Each token
 * has a label and a shape so status remains identifiable when colour is
 * removed or unavailable.
 */
export const TRUST_VERDICTS = {
  VALID: { label: "Valid", mark: "●", tone: "valid" },
  GHOST: { label: "Ghost", mark: "◇", tone: "ghost" },
  SATURATED: { label: "Saturated", mark: "≋", tone: "saturated" },
  SHELVED: { label: "Shelved", mark: "⊘", tone: "shelved" },
  UNIT_MISMATCH: { label: "Unit mismatch", mark: "△", tone: "unit-mismatch" },
  UNAVAILABLE: { label: "Unavailable", mark: "—", tone: "unavailable" },
  UNTRUSTED: { label: "Untrusted", mark: "?", tone: "untrusted" },
} as const;

export type TrustVerdict = keyof typeof TRUST_VERDICTS;

export const BAND_VERDICTS = {
  IN_BAND: { label: "In band", mark: "●", tone: "in-band" },
  OUT_OF_BAND: { label: "Out of band", mark: "!", tone: "out-of-band" },
  PENDING_SUSTAIN: { label: "Pending sustain", mark: "…", tone: "pending-sustain" },
  NEEDS_BASELINE: { label: "Needs baseline", mark: "↕", tone: "needs-baseline" },
  NOT_EVALUATED: { label: "Not evaluated", mark: "?", tone: "not-evaluated" },
} as const;

export type BandVerdict = keyof typeof BAND_VERDICTS;

export type ConfidenceLevel = "AUTHORITATIVE" | "PARTIAL" | "SKETCH";

export interface TrustTripleValue {
  distribution: Readonly<Partial<Record<TrustVerdict, number>>>;
  checked: number;
  total: number;
}

export interface RatioConfidenceValue {
  ratio: number | null;
  confidence: ConfidenceLevel;
  rationale: string;
}

export function trustToken(verdict: TrustVerdict) {
  return TRUST_VERDICTS[verdict];
}

export function bandToken(verdict: BandVerdict) {
  return BAND_VERDICTS[verdict];
}

