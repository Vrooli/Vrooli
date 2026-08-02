import type { ReactNode } from "react";

type MetricTone = "neutral" | "warning" | "success";

export interface MetricProps {
  label: string;
  value: string;
  detail: ReactNode;
  tone?: MetricTone;
}

export function Metric({ label, value, detail, tone = "neutral" }: MetricProps) {
  return (
    <div className="rounded-panel border border-app-border bg-app-surface-muted p-4">
      <p className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">{label}</p>
      <p className="mt-2 text-2xl font-semibold tracking-tight">{value}</p>
      <p className={`mt-1 text-xs ${tone === "warning" ? "text-app-warning" : tone === "success" ? "text-app-success" : "text-app-muted-foreground"}`}>
        {detail}
      </p>
    </div>
  );
}
