import { Check, X } from "lucide-react";
import type { FixResult } from "../lib/api";

export function FixResultsPanel({ results }: { results: FixResult[] }) {
  if (results.length === 0) return null;

  return (
    <div className="mt-3 space-y-2">
      <p className="text-xs font-medium uppercase tracking-wider text-slate-400">Fix Results</p>
      {results.map((r, i) => (
        <div key={i} className="rounded-lg border border-white/5 bg-black/20 p-3 text-sm">
          <div className="flex items-start gap-2">
            {r.fixed ? (
              <Check className="mt-0.5 h-4 w-4 shrink-0 text-emerald-300" />
            ) : (
              <X className="mt-0.5 h-4 w-4 shrink-0 text-red-300" />
            )}
            <div className="min-w-0">
              <p className="text-slate-200">
                <span className="font-medium">{r.scenario_name}</span>
                <span className="mx-1 text-slate-500">/</span>
                <span className="text-slate-400">{r.rule_id}</span>
              </p>
              {r.file_path && <p className="mt-0.5 text-xs text-slate-400 break-all">{r.file_path}</p>}
              {r.error && <p className="mt-1 text-xs text-red-300">{r.error}</p>}
              {r.changes && r.changes.length > 0 && (
                <ul className="mt-1 space-y-0.5">
                  {r.changes.map((c, j) => (
                    <li key={j} className="text-xs text-slate-400">
                      <span className="mr-1 rounded bg-slate-700/50 px-1 py-0.5 text-slate-300">{c.type}</span>
                      {c.detail}
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
