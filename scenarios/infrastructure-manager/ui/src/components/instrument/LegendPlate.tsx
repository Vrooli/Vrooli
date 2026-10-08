import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type { ReactNode } from "react";

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

export interface LegendPlateProps {
  /**
   * A REAL reference — a substrate cell id (`SB1`), a rung tag (`R5`), or a
   * device address (`pci:0000:01:00.0`). Omit it when the section has no real
   * reference to carry; this plate never renders a decorative counter.
   */
  tag?: string;
  /** The engraved legend. Rendered uppercase by the language, not by the caller. */
  legend: string;
  /** Optional right-aligned annotation: a count, a timestamp, a denominator. */
  aside?: ReactNode;
  /**
   * Heading level for the legend. Sections nested inside a page that already
   * has an `<h1>`/`<h2>` should step this down so the document outline stays
   * correct for screen readers.
   */
  as?: "h2" | "h3" | "h4";
  /** Ties the plate to the region it labels via `aria-labelledby`. */
  id?: string;
  className?: string;
}

/**
 * The engraved legend plate: this language's section header.
 *
 * Structure carries information here. The tag is a real reference, the rule
 * runs to the panel edge the way a machined plate does, and the aside holds
 * the figure that qualifies the section rather than repeating its title.
 */
export function LegendPlate({ tag, legend, aside, as: Heading = "h2", id, className }: LegendPlateProps) {
  return (
    <div className={cn("plate", className)}>
      {tag ? (
        <span className="plate__tag" aria-hidden="true">
          {tag}
        </span>
      ) : null}
      <Heading id={id} className="plate__legend">
        {legend}
      </Heading>
      <span className="plate__rule" aria-hidden="true" />
      {aside ? <span className="plate__aside">{aside}</span> : null}
    </div>
  );
}
