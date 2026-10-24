import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type { ReactNode } from "react";

import { strings } from "../../consts/strings.generated";
import { useTranslation } from "../../i18n";

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

export type StatTone = "neutral" | "covered" | "excursion";

export interface StatPlateProps {
  /**
   * The figure. Pass the formatted string, not a number — the caller owns
   * units and precision, and a plate must never invent either.
   *
   * A figure that could not be computed is `null`, which renders an em dash.
   * It is NEVER rendered as zero: on this surface a fabricated zero is the
   * specific dishonesty the instrument exists to remove.
   */
  value: string | null;
  /** What the figure counts, in words. Rendered uppercase by the language. */
  label: ReactNode;
  tone?: StatTone;
  className?: string;
}

/**
 * One stat plate. Plates sit in a `.stat-strip` grid whose 1px gaps run over a
 * border-coloured ground, so the seams read as machined joints.
 */
export function StatPlate({ value, label, tone = "neutral", className }: StatPlateProps) {
  const { t } = useTranslation();
  return (
    <div className={cn("stat", className)}>
      <div
        className={cn(
          "stat__value",
          tone === "covered" && "stat__value--covered",
          tone === "excursion" && "stat__value--excursion",
        )}
      >
        {value ?? <span aria-label={t(strings.instrument.notAvailable)}>—</span>}
      </div>
      <div className="stat__key">{label}</div>
    </div>
  );
}

export interface StatStripProps {
  children: ReactNode;
  /** Accessible name for the group, e.g. "Substrate headline figures". */
  label: string;
  className?: string;
}

export function StatStrip({ children, label, className }: StatStripProps) {
  return (
    <section className={cn("stat-strip", className)} aria-label={label}>
      {children}
    </section>
  );
}
