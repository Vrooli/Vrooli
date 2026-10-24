/**
 * @libraryId react-component-library:PlanSummary
 * @displayName PlanSummary
 * @description An accept-or-adjust summary card for a proposed action, with direct and implied facts distinguished in form and text.
 * @version 0.1.1
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { useId } from "react";

import { Button } from "@vrooli/react-component-library/Button/2";
import { StatusBadge, type StatusTone } from "@vrooli/react-component-library/StatusBadge/1";

export interface PlanFact {
  value: string;
  label: string;
  tone?: StatusTone;
}

export interface PlanSummaryProps {
  title: string;
  kicker?: string;
  note?: string;
  facts: PlanFact[];
  items?: Array<{ label: string; implied?: boolean }>;
  headingLevel?: 2 | 3 | 4;
  onAccept: () => void;
  acceptLabel: string;
  onAdjust?: () => void;
  adjustLabel?: string;
}

export function PlanSummary({
  title,
  kicker,
  note,
  facts,
  items = [],
  headingLevel = 2,
  onAccept,
  acceptLabel,
  onAdjust,
  adjustLabel,
}: PlanSummaryProps) {
  const titleId = useId();
  const Heading = `h${headingLevel}` as "h2" | "h3" | "h4";

  return (
    <section data-testid="plan-summary" className="plan-summary" aria-labelledby={titleId}>
      <header>
        {kicker && <span>{kicker}</span>}
        <Heading id={titleId}>{title}</Heading>
        {note && <p>{note}</p>}
      </header>
      <div className="plan-summary-facts">
        {facts.map((fact) => (
          <div key={fact.label}>
            <strong data-tone={fact.tone}>{fact.value}</strong>
            <span>{fact.label}</span>
          </div>
        ))}
      </div>
      {items.length > 0 && (
        <ul aria-label="Included items">
          {items.map((item) => (
            <li
              key={item.label}
              aria-label={`${item.label}${item.implied ? " (included by dependency)" : " (selected directly)"}`}
            >
              <StatusBadge tone={item.implied ? "neutral" : "info"}>{item.label}</StatusBadge>
            </li>
          ))}
        </ul>
      )}
      <footer>
        <Button type="button" onClick={onAccept}>
          {acceptLabel}
        </Button>
        {onAdjust && adjustLabel && (
          <Button type="button" variant="secondary" onClick={onAdjust}>
            {adjustLabel}
          </Button>
        )}
      </footer>
    </section>
  );
}
