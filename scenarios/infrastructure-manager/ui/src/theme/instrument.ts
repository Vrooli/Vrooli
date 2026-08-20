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
  NOT_GRADEABLE: { label: "Not gradeable", mark: "∅", tone: "not-evaluated" },
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


/**
 * The five-rung observability ladder.
 *
 * The order is a DEPENDENCY CHAIN, not a preference: a rung cannot report
 * covered while the rung below it is blind. Crash evidence cannot be kept for
 * a device that was never enumerated, and nothing can be anticipated about a
 * device whose current state cannot be read.
 *
 * `index` is the wire/display ordinal (1-based). `tag` is what appears on a
 * legend plate and a column header — it is a real reference, not decoration.
 */
export const RUNGS = {
  IDENTITY: {
    index: 1,
    tag: "R1",
    label: "Identity",
    question: "Does the platform know this device exists?",
  },
  TELEMETRY: {
    index: 2,
    tag: "R2",
    label: "Telemetry",
    question: "Can its current state be read?",
  },
  EVIDENCE: {
    index: 3,
    tag: "R3",
    label: "Evidence",
    question: "When it fails, is the proof kept?",
  },
  CONTROL: {
    index: 4,
    tag: "R4",
    label: "Control",
    question: "Can the platform act on it?",
  },
  ANTICIPATION: {
    index: 5,
    tag: "R5",
    label: "Anticipation",
    question: "Can failure be seen coming?",
  },
} as const;

export type Rung = keyof typeof RUNGS;

/** The rungs in dependency order, innermost first. */
export const RUNG_ORDER: readonly Rung[] = [
  "IDENTITY",
  "TELEMETRY",
  "EVIDENCE",
  "CONTROL",
  "ANTICIPATION",
];

/**
 * The signal states a single (device, rung) cell can be in.
 *
 * EIGHT states, and the length of this list is the point rather than an
 * accident. This instrument exists because monitoring surfaces collapse
 * distinct facts into one another — "we could not read it", "there is nothing
 * to read", "nobody built a reader", and "the reader is down" all render as a
 * blank card or a green tick elsewhere, and they are four different problems
 * with four different owners.
 *
 * Facts about the PLANT that were measured:
 * - COVERED      a real value was obtained from the host.
 * - PARTIAL      obtained, with a stated limit, or inside a deadband but off ideal.
 * - EXCURSION    obtained and out of band. Rare by construction (EEMUA 191).
 *
 * Facts about the PLANT that were not measured, each distinct:
 * - UNMEASURABLE the rung applies and a value SHOULD exist, but the host
 *                refused or could not produce it (e.g. smartctl present,
 *                permission denied). Carries a REASON. Not zero. Not healthy.
 * - UNAVAILABLE  the mechanism that would produce the value is not present on
 *                this host at all — a missing tool, a missing interface. The
 *                host is answering honestly; nothing is broken.
 * - NOT_APPLICABLE the rung is meaningless for this device class, and is
 *                graded somewhere else. Rendering this as blindness would
 *                manufacture a gap that does not exist.
 *
 * A fact about the PLATFORM:
 * - BLIND        declared blindness. No sensor exists anywhere for this, and
 *                the cell carries the date that became true. This is a
 *                measurement, not an absence of one, and it is the state the
 *                whole board is built to make visible.
 *
 * A fact about the DENOMINATOR — about what anybody has thought to ask:
 * - UNAUTHORED   the owner's space document declares no cell for this rung on
 *                this class, so there is nothing to grade and nothing to be
 *                blind about. It is NOT blindness (nobody declared a gap), NOT
 *                inapplicable (nobody said the rung is meaningless here), and
 *                emphatically NOT a source outage. It means the question has
 *                not been asked yet, which is a fact about the SPACE rather
 *                than about the machine or the instrument.
 *
 * A fact about the INSTRUMENT, never about the plant:
 * - SOURCE_DOWN  the sensor source could not be reached at read time. Kept
 *                rigidly separate from every state above, and rendered on the
 *                instrument chrome rather than in the plant data, so an owner
 *                outage can never read as a coverage collapse.
 *
 * Every state carries a distinct `mark` and `short` as well as a `tone`, so the
 * panel stays readable when colour is removed — which the scenario's
 * `status-not-colour-alone` experience claim requires.
 */
export const SIGNAL_STATES = {
  COVERED: { label: "Covered", mark: "\u25cf", tone: "covered", short: "ON" },
  PARTIAL: { label: "Partial", mark: "\u25d0", tone: "partial", short: "PART" },
  EXCURSION: { label: "Excursion", mark: "!", tone: "excursion", short: "ALRM" },
  UNMEASURABLE: { label: "Unmeasurable", mark: "\u2298", tone: "unmeasurable", short: "N/M" },
  UNAVAILABLE: { label: "Mechanism absent", mark: "\u2205", tone: "unavailable", short: "NONE" },
  NOT_APPLICABLE: { label: "Not applicable", mark: "\u2013", tone: "not-applicable", short: "N/A" },
  BLIND: { label: "Blind", mark: "\u25cb", tone: "blind", short: "OFF" },
  UNAUTHORED: { label: "No cell authored", mark: "\u00b7", tone: "unauthored", short: "\u2014" },
  SOURCE_DOWN: { label: "Source unreachable", mark: "?", tone: "source-down", short: "SRC?" },
} as const;

export type SignalState = keyof typeof SIGNAL_STATES;

export function signalToken(state: SignalState) {
  return SIGNAL_STATES[state];
}

export function rungToken(rung: Rung) {
  return RUNGS[rung];
}

/**
 * Enforces the ladder's dependency rule when grading a device: no rung may
 * report COVERED while any rung below it is BLIND. A rung that would claim
 * coverage over a blind foundation is demoted to PARTIAL, because the claim
 * is not false so much as unsupported — and silently letting it stand is how
 * a ladder stops meaning anything.
 *
 * States that describe the instrument rather than the plant (UNAVAILABLE) and
 * states that are already honest about a limit (UNMEASURABLE, EXCURSION) pass
 * through untouched: demoting them would destroy information.
 */
export function enforceRungDependency(
  states: Readonly<Record<Rung, SignalState>>,
): Record<Rung, SignalState> {
  const result = { ...states } as Record<Rung, SignalState>;
  let foundationBlind = false;
  for (const rung of RUNG_ORDER) {
    if (foundationBlind && result[rung] === "COVERED") {
      result[rung] = "PARTIAL";
    }
    if (result[rung] === "BLIND") {
      foundationBlind = true;
    }
  }
  return result;
}
