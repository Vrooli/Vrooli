import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

import { strings } from "../../consts/strings.generated";
import { formatNumber } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import {
  TRUST_VERDICTS,
  bandToken,
  trustToken,
  type BandVerdict,
  type ConfidenceLevel,
  type RatioConfidenceValue,
  type TrustTripleValue,
  type TrustVerdict,
} from "../../theme/instrument";

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

/**
 * The three closed vocabularies, mapped to catalog keys.
 *
 * `theme/instrument.ts` owns the vocabulary itself — the mark, the tone and
 * the dependency rules — and it is deliberately free of any i18n dependency
 * so non-component code can grade a reading without a translator in scope.
 * The rendered LABEL is a different concern and belongs to the surface, so it
 * is resolved here. These maps are exhaustive `Record`s: adding a verdict to
 * the vocabulary is a TypeScript error until its label exists in the catalog,
 * which is what keeps the two from drifting.
 */
const TRUST_LABEL_KEYS = {
  VALID: strings.instrument.trust.valid,
  GHOST: strings.instrument.trust.ghost,
  SATURATED: strings.instrument.trust.saturated,
  SHELVED: strings.instrument.trust.shelved,
  UNIT_MISMATCH: strings.instrument.trust.unitMismatch,
  UNAVAILABLE: strings.instrument.trust.unavailable,
  UNTRUSTED: strings.instrument.trust.untrusted,
} as const satisfies Record<TrustVerdict, string>;

const BAND_LABEL_KEYS = {
  IN_BAND: strings.instrument.band.inBand,
  OUT_OF_BAND: strings.instrument.band.outOfBand,
  PENDING_SUSTAIN: strings.instrument.band.pendingSustain,
  NEEDS_BASELINE: strings.instrument.band.needsBaseline,
  NOT_EVALUATED: strings.instrument.band.notEvaluated,
  NOT_GRADEABLE: strings.instrument.band.notGradeable,
} as const satisfies Record<BandVerdict, string>;

const CONFIDENCE_LABEL_KEYS = {
  AUTHORITATIVE: strings.instrument.confidence.authoritative,
  PARTIAL: strings.instrument.confidence.partial,
  SKETCH: strings.instrument.confidence.sketch,
} as const satisfies Record<ConfidenceLevel, string>;

/**
 * A denominator's confidence is a claim about the INSTRUMENT's own arithmetic,
 * so it borrows the signal palette rather than owning one: an authoritative
 * denominator reads lit, a partial one reads as a stated limit, and a sketch
 * denominator takes the excursion colour because a ratio measured against a
 * guessed population is a figure contradicted by its own evidence.
 */
const CONFIDENCE_TONE: Record<ConfidenceLevel, string> = {
  AUTHORITATIVE: "text-signal-covered",
  PARTIAL: "text-signal-partial",
  SKETCH: "text-signal-excursion",
};

/** Engraved-plate typography shared by every token on this surface. */
const TOKEN_TYPE = "font-mono text-body-sm uppercase tracking-[0.09em] tabular-nums";

/**
 * The trust distribution, its checked denominator and its total, rendered as
 * ONE unit.
 *
 * The scenario's `trust-triple-is-atomic` claim is the reason this is a single
 * component rather than three figures a page may arrange as it likes: a
 * verdict count quoted without the population it was drawn from is the exact
 * shape of an honest-looking lie.
 */
export function TrustTriple({ value }: { value: TrustTripleValue }) {
  const { t } = useTranslation();
  const entries = (Object.entries(value.distribution) as [TrustVerdict, number][]).filter(
    ([, count]) => count > 0,
  );
  return (
    <section className="instrument-triple" aria-label={t(strings.instrument.trustDistributionLabel)}>
      <div className="instrument-triple__counts">
        {entries.length === 0 ? (
          <span className="blind-note">{t(strings.instrument.noVerdicts)}</span>
        ) : (
          entries.map(([verdict, count]) => (
            <span
              key={verdict}
              className={cn("status-token", `status-token--${trustToken(verdict).tone}`, TOKEN_TYPE)}
            >
              <span aria-hidden="true">{trustToken(verdict).mark}</span>
              <span>{formatNumber(count)}</span>
              <span>{t(TRUST_LABEL_KEYS[verdict])}</span>
            </span>
          ))
        )}
      </div>
      <p className="instrument-triple__denominator font-mono tracking-[0.04em] tabular-nums">
        {t(strings.instrument.checkedOf, {
          checked: formatNumber(value.checked),
          total: formatNumber(value.total),
        })}
      </p>
    </section>
  );
}

/**
 * A ratio and the confidence of the denominator it was measured against, in
 * one visual unit, with the rationale for that confidence beneath.
 *
 * A ratio that could not be computed renders an em dash. It is never rendered
 * as `0%`: an unread space and a fully uninstrumented one are different facts,
 * and collapsing them is the specific dishonesty this instrument removes.
 */
export function RatioConfidence({ value }: { value: RatioConfidenceValue }) {
  const { t } = useTranslation();
  const confidence = t(CONFIDENCE_LABEL_KEYS[value.confidence]);
  const ratio =
    value.ratio === null
      ? null
      : formatNumber(value.ratio, { style: "percent", maximumFractionDigits: 1 });
  return (
    <section
      className="ratio-confidence"
      aria-label={
        ratio === null
          ? t(strings.instrument.ratioUncomputedLabel, { confidence })
          : t(strings.instrument.ratioLabel, { ratio, confidence })
      }
    >
      <strong className="ratio-confidence__ratio stat__value">
        {ratio ?? <span aria-label={t(strings.instrument.notAvailable)}>—</span>}
      </strong>
      <span
        className={cn(
          "self-center justify-self-start rounded-control border border-current px-space-2xs py-space-3xs",
          TOKEN_TYPE,
          CONFIDENCE_TONE[value.confidence],
        )}
      >
        {confidence}
      </span>
      <span className="ratio-confidence__rationale">{value.rationale}</span>
    </section>
  );
}

/**
 * One verdict from either closed vocabulary.
 *
 * Status is carried three ways — a mark, a text label and a tone — so the
 * token survives colour removal, which the `status-not-colour-alone` claim
 * requires. The mark is never rendered alone.
 */
export function StatusToken({ verdict }: { verdict: TrustVerdict | BandVerdict }) {
  const { t } = useTranslation();
  const isTrust = verdict in TRUST_VERDICTS;
  const token = isTrust ? trustToken(verdict as TrustVerdict) : bandToken(verdict as BandVerdict);
  const labelKey = isTrust
    ? TRUST_LABEL_KEYS[verdict as TrustVerdict]
    : BAND_LABEL_KEYS[verdict as BandVerdict];
  return (
    <span className={cn("status-token", `status-token--${token.tone}`, TOKEN_TYPE)}>
      <span aria-hidden="true">{token.mark}</span>
      <span>{t(labelKey)}</span>
    </span>
  );
}
