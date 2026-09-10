import type { ExecutionLimits } from "../../types/backlog";

export function ExecutionLimitsSummary({ limits }: { limits: ExecutionLimits }) {
  const money = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" }).format(limits.maxChargeMicroUsd / 1_000_000);
  const entries = [
    ["Slices", limits.maxSlices.toLocaleString()],
    ["Tokens", limits.maxTokens.toLocaleString()],
    ["Wall time", `${(limits.maxWallSeconds / 3600).toLocaleString()} hours`],
    ["Turns", limits.maxTurns.toLocaleString()],
    ["Agent spend", money],
    ["Child launches", limits.maxChildren.toLocaleString()],
    ["Node attempts", limits.maxNodeAttempts.toLocaleString()],
    ["Retries", limits.maxRetries.toLocaleString()],
  ];
  return <section className="rounded-lg border border-white/10 bg-slate-950/40 p-3" aria-label="Execution limits">
    <h3 className="text-sm font-semibold text-white">Execution limits</h3>
    <p className="mt-1 text-xs text-slate-400">Allowances included in this item's approval. Token and spend limits pause new work after a run's usage is known. An active run can exceed the remaining allowance.</p>
    <dl className="mt-2 grid grid-cols-2 gap-x-4 gap-y-2 text-xs">{entries.map(([label, value]) => <div key={label}><dt className="text-slate-400">{label}</dt><dd className="mt-0.5 text-slate-100">{value}</dd></div>)}</dl>
  </section>;
}
