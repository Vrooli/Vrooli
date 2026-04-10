import type { Finding } from "../lib/api";

function levelBadge(level: string) {
  switch (level) {
    case "error":
      return { label: "Error", className: "bg-red-500/15 text-red-200 border-red-500/30" };
    case "warn":
      return { label: "Warn", className: "bg-amber-500/15 text-amber-200 border-amber-500/30" };
    default:
      return { label: "Info", className: "bg-slate-500/15 text-slate-200 border-slate-500/30" };
  }
}

export function FindingItem({ finding }: { finding: Finding }) {
  const badge = levelBadge(finding.level);

  return (
    <div className="rounded-lg border border-white/5 bg-black/20 p-3">
      <div className="flex items-start gap-2">
        <span className={`mt-0.5 inline-flex shrink-0 items-center rounded-full border px-2 py-0.5 text-xs ${badge.className}`}>
          {badge.label}
        </span>
        <p className="text-sm text-slate-200">{finding.message}</p>
      </div>
      {finding.evidence && finding.evidence.length > 0 && (
        <details className="mt-2">
          <summary className="cursor-pointer text-xs text-slate-400">
            {finding.evidence.length} evidence item{finding.evidence.length !== 1 ? "s" : ""}
          </summary>
          <ul className="mt-1 space-y-1">
            {finding.evidence.map((ev, i) => (
              <li key={i} className="rounded border border-white/5 bg-black/10 px-2 py-1 text-xs">
                <span className="mr-2 rounded bg-slate-700/50 px-1.5 py-0.5 text-slate-300">{ev.type}</span>
                {ev.ref && <span className="text-slate-300 break-all">{ev.ref}</span>}
                {ev.detail && <p className="mt-0.5 text-slate-400 break-all">{ev.detail}</p>}
              </li>
            ))}
          </ul>
        </details>
      )}
    </div>
  );
}
