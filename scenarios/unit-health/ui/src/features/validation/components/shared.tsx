import type { ReactNode } from "react";

export function Metric({
  label,
  value,
  testId,
  icon,
  tone = "border-app-border bg-app-surface text-app-foreground",
}: {
  label: string;
  value: string;
  testId: string;
  icon: ReactNode;
  tone?: string;
}) {
  return (
    <div data-testid={testId} className={`rounded-panel border p-4 ${tone}`}>
      <div className="flex items-center justify-between gap-3">
        <p className="text-xs font-semibold uppercase">{label}</p>
        {icon}
      </div>
      <p className="mt-3 text-xl font-semibold">{value}</p>
    </div>
  );
}

export function Panel({
  title,
  testId,
  children,
}: {
  title: string;
  testId: string;
  children: ReactNode;
}) {
  return (
    <section data-testid={testId} className="rounded-panel border border-app-border bg-app-surface p-4">
      <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">{title}</h3>
      <div className="mt-3">{children}</div>
    </section>
  );
}

export function Pill({ tone, children }: { tone: string; children: ReactNode }) {
  return <span className={`rounded-control border px-2 py-0.5 text-xs ${tone}`}>{children}</span>;
}
