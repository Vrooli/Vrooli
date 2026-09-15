import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

import { strings } from "../../consts/strings.generated";
import { useTranslation } from "../../i18n";
import { signalToken, type SignalState } from "../../theme/instrument";

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

export interface LampProps {
  state: SignalState;
  /**
   * What this lamp is reporting on, in words — for example
   * "storage · anticipation". Combined with the state label it becomes the
   * lamp's accessible name, so a screen reader hears the finding rather than
   * a glyph.
   */
  subject: string;
  /**
   * Why the state is what it is. Required for UNMEASURABLE and UNAVAILABLE:
   * those two states are only honest when they carry their reason, and a lamp
   * that reports "unmeasurable" with no reason is the failure this instrument
   * exists to remove.
   */
  reason?: string;
  /** For BLIND lamps: how long this has been blind, from `gap_open_days`. */
  blindDays?: number;
  className?: string;
}

/**
 * One annunciator lamp: a single (device, rung) state.
 *
 * State is carried three ways at once — colour, a distinct glyph, and a text
 * label in the accessible name — so the lamp survives colour removal, which
 * the scenario's `status-not-colour-alone` experience claim requires.
 *
 * BLIND is the default visual rather than a special case. On this panel an
 * unlit lamp is the content: it means declared blindness, dated, and it is
 * the thing the reader is meant to notice first.
 */
export function Lamp({ state, subject, reason, blindDays, className }: LampProps) {
  const { t } = useTranslation();
  const token = signalToken(state);
  const parts = [subject, token.label];
  if (typeof blindDays === "number" && state === "BLIND") {
    parts.push(t(strings.instrument.blindFor, { days: blindDays }));
  }
  if (reason) {
    parts.push(reason);
  }

  return (
    <span
      className={cn("lamp", `lamp--${token.tone}`, className)}
      role="img"
      aria-label={parts.join(", ")}
      title={reason}
      data-state={state}
    >
      <span aria-hidden="true">{token.mark}</span>
      <span aria-hidden="true">{token.short}</span>
    </span>
  );
}

export interface LampLegendProps {
  /**
   * The states actually present on the surface this legend describes. A key
   * for a state that does not appear teaches the reader a distinction the
   * page never makes, so callers pass only what they rendered.
   */
  states: readonly SignalState[];
  className?: string;
}

/** The key that teaches the panel's vocabulary, rendered beneath a panel. */
export function LampLegend({ states, className }: LampLegendProps) {
  return (
    <ul className={cn("flex flex-wrap gap-x-space-md gap-y-space-2xs list-none p-0 m-0", className)}>
      {states.map((state) => {
        const token = signalToken(state);
        return (
          <li key={state} className="legend-key">
            <Lamp state={state} subject={token.label} />
            <span>{token.label}</span>
          </li>
        );
      })}
    </ul>
  );
}
