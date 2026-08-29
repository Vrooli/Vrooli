export function MachineSilhouette({ local = false, reachable = false }: { local?: boolean; reachable?: boolean }) {
  return (
    <div data-testid="machine-silhouette" className="flex h-full items-center justify-center p-5" aria-label={local ? "Local computer" : "Remote machine"}>
      <div className={`relative flex h-16 w-32 items-center justify-center rounded-lg border-2 bg-[var(--wc-device-body)] shadow-inner ${local ? "border-cyan-300/70" : "border-slate-500/70"}`}>
        <div className="h-10 w-24 rounded border border-slate-400/30 bg-[var(--wc-device-screen)]/20" />
        <span className={`absolute right-2 top-2 h-2 w-2 rounded-full ${reachable || local ? "bg-emerald-400" : "bg-amber-300"}`} />
      </div>
      <div className="absolute bottom-3 h-1.5 w-20 rounded-full bg-[var(--wc-device-rim)]/70" />
    </div>
  );
}

export default MachineSilhouette;
