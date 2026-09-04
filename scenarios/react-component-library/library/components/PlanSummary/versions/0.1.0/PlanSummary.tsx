/**
 * @libraryId react-component-library:PlanSummary
 * @displayName Plan Summary
 * @description A named recommendation with quantified facts and accept-or-adjust actions.
 * @version 0.1.0
 * @tags ["data-display","approval"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
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
  onAccept,
  acceptLabel,
  onAdjust,
  adjustLabel,
}: PlanSummaryProps) {
  return (
    <section
      data-testid="plan-summary"
      className="plan-summary"
      role="region"
      aria-labelledby="plan-summary-title"
    >
      <header>
        <span>{kicker}</span>
        <h1 id="plan-summary-title">{title}</h1>
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
