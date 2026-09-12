export type HostCapabilityStanding = "available" | "unsupported" | "denied" | "disconnected" | "revoked" | "rate_limited" | "unconfigured";

export interface HostCapabilityStatus {
  capability: string;
  standing: HostCapabilityStanding;
  reason?: string;
}

export function HostCapabilityStatusCard({ hostLabel, capabilities }: { hostLabel: string; capabilities: HostCapabilityStatus[] }) {
  return (
    <section aria-label="Host capabilities" className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
      <div className="mb-2 flex items-center justify-between"><h3 className="text-xs font-medium text-slate-300">Host capabilities</h3><span className="text-[11px] text-slate-500">{hostLabel}</span></div>
      <div className="grid gap-2 sm:grid-cols-2">
        {capabilities.map((item) => <div key={item.capability} className="flex items-center justify-between gap-2 text-[11px]"><span className="text-slate-400">{item.capability}</span><span className={item.standing === "available" ? "text-emerald-300" : "text-amber-300"} title={item.reason}>{item.standing}</span></div>)}
      </div>
    </section>
  );
}
