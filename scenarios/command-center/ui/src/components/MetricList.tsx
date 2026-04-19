import { type MetricEntry } from "../lib/api";
import { GapBadge } from "./GapBadge";

interface MetricListProps {
  metrics: MetricEntry[];
}

/**
 * Renders a vertical list of metrics with gap/partial badges alongside
 * non-live entries. Descriptions are rendered in the muted foreground color.
 */
export function MetricList({ metrics }: MetricListProps) {
  if (metrics.length === 0) {
    return (
      <p className="cc-metric-description" data-testid="metric-list-empty">
        No metrics configured.
      </p>
    );
  }

  return (
    <ul className="cc-surface cc-metric-list" data-testid="metric-list">
      {metrics.map((metric) => (
        <li key={metric.id} className="cc-metric" data-metric-id={metric.id}>
          <div className="cc-metric-label">
            <GapBadge status={metric.dataSource} whatIsNeeded={metric.whatIsNeeded} />
            {metric.label}
          </div>
          {metric.description ? (
            <div className="cc-metric-description">{metric.description}</div>
          ) : null}
        </li>
      ))}
    </ul>
  );
}
