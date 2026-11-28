export function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-panel border border-app-border bg-app-surface p-4">
      <p className="text-xs font-semibold uppercase text-app-muted-foreground">{label}</p>
      <p className="mt-2 text-2xl font-semibold">{value}</p>
    </div>
  );
}
