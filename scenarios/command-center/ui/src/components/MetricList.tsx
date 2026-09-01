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
        <li key={metric.id} className={`cc-metric cc-ink-${inkFor(metric)}`} data-metric-id={metric.id} data-reading data-coverage={metric.coverage} data-trust={metric.trust} data-empirical={metric.empirical} data-provenance={provenanceFor(metric)}>
          <div className="cc-metric-label">
            <GapBadge status={metric.dataSource ?? (metric.coverage === "NOW" ? "live" : metric.coverage === "IN-REACH" ? "partial" : "gap")} whatIsNeeded={metric.whatIsNeeded} />
            {metric.label}
          </div>
          {metric.value !== null && metric.value !== undefined ? <strong className="cc-metric-value" data-figure>{formatValue(metric.value, metric.format)}{metric.unit ? ` ${metric.unit}` : ""}</strong> : metric.sample ? <strong className="cc-metric-value cc-sample-value" data-figure>{formatValue(metric.sample.value, metric.format)}{metric.unit ? ` ${metric.unit}` : ""}</strong> : null}
          <div className="cc-metric-description" data-qualifier>{qualifierFor(metric)}</div>
          {metric.description ? <div className="cc-metric-description">{metric.description}</div> : null}
        </li>
      ))}
    </ul>
  );
}

function provenanceFor(metric: MetricEntry): "measured" | "cached" | "sample" | "absent" {
  if (metric.coverage === "MISSING" || metric.coverage === "UNREGISTERED") return "absent";
  if (metric.coverage === "IN-REACH") return "sample";
  return metric.trust === "CACHED" ? "cached" : "measured";
}

function qualifierFor(metric: MetricEntry): string {
  if (metric.coverage === "MISSING" || metric.coverage === "UNREGISTERED") return `${metric.source?.team ?? metric.owner ?? "owner unknown"} · ${metric.gapOpenDays ?? 0} days open`;
  if (metric.coverage === "IN-REACH") return metric.whatIsNeeded ?? metric.sample?.basis ?? "Sensor is in reach; pipeline is not yet reporting.";
  if (metric.trust === "CACHED") return `Cached · observed ${metric.observedAt ?? "previously"}`;
  if (metric.trust === "UNAVAILABLE") return "Source unavailable; no value asserted.";
  return `${metric.source?.binding ?? metric.upstreamSource} · observed ${metric.observedAt ?? "now"}`;
}

function formatValue(value: number, format?: string): string {
  if (format === "percent") return `${(value * 100).toFixed(1)}%`;
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 }).format(value);
}

function inkFor(metric: MetricEntry): "solid" | "dimmed" | "hollow" | "dotted" {
  if (metric.coverage === "MISSING") return "dotted";
  if (metric.coverage === "IN-REACH") return "hollow";
  if (metric.trust === "CACHED") return "dimmed";
  return "solid";
}
