/** @vrooliComponentSource visualization.cartesian-charts */
import type { CSSProperties } from "react";

export interface CartesianPoint {
  id: string;
  label: string;
  value: number;
  detail?: string;
}
export function CartesianCharts({
  title,
  description,
  data,
}: {
  title: string;
  description?: string;
  data: CartesianPoint[];
}) {
  const max = Math.max(...data.map((point) => point.value), 1);
  return (
    <section
      aria-labelledby="cartesian-chart-title"
      data-testid="cartesian-charts"
      className="rounded-panel border border-app-border bg-app-surface p-space-sm"
    >
      <h3 id="cartesian-chart-title" className="text-sm font-semibold">
        {title}
      </h3>
      {description && (
        <p className="mt-space-3xs text-xs text-app-muted-foreground">{description}</p>
      )}
      <div
        role="img"
        aria-label={title}
        data-rcl-chart
        className="mt-space-sm grid gap-space-xs"
      >
        {data.length === 0 ? (
          <p role="status" className="text-xs text-app-muted-foreground">
            No progression data is available.
          </p>
        ) : (
          data.map((point) => (
            <div
              key={point.id}
              className="grid grid-cols-[7rem_minmax(0,1fr)_auto] items-center gap-space-xs text-xs"
            >
              <span className="font-mono">{point.label}</span>
              <span className="h-3 rounded-pill bg-app-surface-muted" aria-hidden="true">
                <span
                  className="block h-full rounded-pill bg-app-primary"
                  style={
                    {
                      "--chart-width": `${Math.max(4, (point.value / max) * 100)}%`,
                      width: "var(--chart-width)",
                    } as CSSProperties
                  }
                />
              </span>
              <span className="font-mono text-app-muted-foreground">
                {point.value}
                {point.detail ? ` · ${point.detail}` : ""}
              </span>
            </div>
          ))
        )}
      </div>
      <table className="sr-only" data-rcl-chart-table>
        <caption>{title} data</caption>
        <thead>
          <tr>
            <th>Version</th>
            <th>Value</th>
          </tr>
        </thead>
        <tbody>
          {data.map((point) => (
            <tr key={point.id}>
              <th>{point.label}</th>
              <td>{point.value}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}
