import type { ReactNode } from "react";
import type { ExecutionLimits } from "../../types/backlog";
import { Popover, PopoverContent, PopoverTrigger } from "@vrooli/react-component-library/Popover/1.2.9";
import { CircleDollarSign, Info, ShieldCheck } from "lucide-react";

export function ExecutionLimitsSummary({ limits }: { limits: ExecutionLimits }) {
  const money = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" }).format(limits.maxChargeMicroUsd / 1_000_000);
  const budget = [
    ["Agent spend", money],
    ["Tokens", limits.maxTokens.toLocaleString()],
    ["Wall time", `${(limits.maxWallSeconds / 3600).toLocaleString()} hours`],
  ];
  const guardrails = [
    ["Slices", limits.maxSlices.toLocaleString()],
    ["Turns", limits.maxTurns.toLocaleString()],
    ["Child launches", limits.maxChildren.toLocaleString()],
    ["Retries", limits.maxRetries.toLocaleString()],
    ["Node attempts", limits.maxNodeAttempts.toLocaleString()],
  ];

  return <section className="rounded-xl border border-cyan-400/20 bg-gradient-to-br from-cyan-400/[0.08] via-slate-950/40 to-slate-950/40 p-4" aria-label="Execution limits">
    <div className="flex items-start gap-3">
      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-cyan-400/10 text-cyan-200"><ShieldCheck className="h-5 w-5" aria-hidden="true" /></div>
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2"><h3 className="text-sm font-semibold text-white">Approved run guardrails</h3><span className="rounded-full border border-cyan-300/20 bg-cyan-300/10 px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-cyan-200">Included allowance</span></div>
        <p className="mt-1 text-xs leading-5 text-slate-400">This item already has a reviewed allowance. Spending and token limits pause new work once usage is known; work already in progress can finish.</p>
      </div>
    </div>
    <div className="mt-4 grid gap-3 sm:grid-cols-2">
      <LimitGroup icon={<CircleDollarSign className="h-4 w-4" aria-hidden="true" />} title="Budget" description="How much capacity the run can consume." entries={budget}>
        <p>
          The approved spend ceiling is stored in micro-USD, so the displayed
          amount is <span className="font-medium text-slate-100">max charge ÷ 1,000,000</span>.
          For example, 30,000,000 micro-USD equals $30.00.
        </p>
        <p>
          The strategy estimate shown above is calculated separately as
          <span className="font-medium text-slate-100"> cost per turn × configured maximum turns</span>.
          It is a planning estimate, not a guarantee or an extra allowance.
          Actual spend comes from measured provider receipts, while token and
          wall-time limits remain independent caps.
        </p>
      </LimitGroup>
      <LimitGroup icon={<ShieldCheck className="h-4 w-4" aria-hidden="true" />} title="Safety rails" description="How the work is kept bounded." entries={guardrails}>
        <p>
          These limits control execution shape rather than price. A slice is one
          bounded pass through part of the plan; turns are model interaction
          cycles; child launches start delegated work; node attempts count
          workflow-node tries; and retries allow failed work to be tried again.
        </p>
        <p>
          Reaching any rail prevents additional work beyond that allowance. The
          rails do not convert into one another or increase the approved budget.
        </p>
      </LimitGroup>
    </div>
  </section>;
}

function LimitGroup({ icon, title, description, entries, children }: { icon: ReactNode; title: string; description: string; entries: string[][]; children: ReactNode }) {
  return <div className="rounded-lg border border-white/10 bg-slate-950/35 p-3">
    <div className="flex items-center gap-2 text-slate-200"><span className="text-cyan-300">{icon}</span><h4 className="text-xs font-semibold uppercase tracking-wide">{title}</h4><Popover placement="bottom-start" responsive="auto"><PopoverTrigger asChild><button type="button" className="ml-auto inline-flex h-7 w-7 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-white/10 hover:text-cyan-200" aria-label={`Explain ${title}`}><Info className="h-3.5 w-3.5" aria-hidden="true" /></button></PopoverTrigger><PopoverContent initialFocus="none" aria-label={`${title} explained`} className="space-y-3 p-3 text-xs leading-5 text-slate-300"><h5 className="text-sm font-semibold text-slate-100">{title} explained</h5>{children}</PopoverContent></Popover></div>
    <p className="mt-1 text-[11px] leading-4 text-slate-500">{description}</p>
    <dl className="mt-3 space-y-2 text-xs">{entries.map(([label, value]) => <div key={label} className="flex items-baseline justify-between gap-3"><dt className="text-slate-400">{label}</dt><dd className="font-medium text-slate-100">{value}</dd></div>)}</dl>
  </div>;
}
