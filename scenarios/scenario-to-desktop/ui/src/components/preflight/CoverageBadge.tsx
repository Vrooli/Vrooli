/**
 * Badge component showing coverage status for a validation aspect.
 */

interface CoverageBadgeProps {
  label: string;
  active: boolean;
}

export function CoverageBadge({ label, active }: CoverageBadgeProps) {
  return (
    <div className={`rounded-md border px-2 py-1 text-[10px] ${active ? "border-sky-800/70 bg-sky-950/40 text-sky-200" : "border-slate-800/70 bg-slate-950/60 text-slate-400"}`}>
      <div className="flex items-center justify-between gap-2">
        <span className="uppercase tracking-wide">{label}</span>
        <span className="text-[9px]">{active ? "On" : "Off"}</span>
      </div>
    </div>
  );
}
