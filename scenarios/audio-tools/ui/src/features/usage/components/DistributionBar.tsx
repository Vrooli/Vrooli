export function DistributionBar({ label, value, total }: { label: string; value: number; total: number }) {
  const pct = total > 0 ? Math.round((value / total) * 100) : 0;
  return (
    <li>
      <div className="mb-1 flex items-center justify-between text-xs text-app-muted-foreground">
        <span>{label}</span>
        <span>{value} · {pct}%</span>
      </div>
      <div className="h-2 w-full overflow-hidden rounded-pill bg-app-surface-muted">
        <div className="h-full bg-app-primary" style={{ width: `${pct}%` }} aria-hidden="true" />
      </div>
    </li>
  );
}
