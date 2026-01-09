/**
 * Coverage map showing what preflight validates vs full app.
 */

import { COVERAGE_ROWS } from "../../lib/preflight-constants";
import { CoverageBadge } from "./CoverageBadge";

export function CoverageMap() {
  return (
    <details className="rounded-md border border-slate-800 bg-slate-950/40 p-3 text-xs text-slate-200">
      <summary className="cursor-pointer text-xs font-semibold text-slate-100">Coverage map</summary>
      <p className="mt-2 text-[11px] text-slate-400">
        Preflight starts the runtime supervisor, validates bundle files, applies secrets, starts services, and
        captures readiness plus log tails. The Electron UI is not started during preflight.
      </p>
      <div className="mt-3 space-y-2">
        {COVERAGE_ROWS.map((row) => {
          const isActive = row.id === "preflight";
          return (
            <div
              key={row.id}
              className={`rounded-md border px-3 py-2 ${isActive ? "border-slate-600/70 bg-slate-900/60" : "border-slate-800/70 bg-slate-950/60"}`}
            >
              <div className="flex flex-wrap items-center justify-between gap-2 text-[11px] text-slate-400">
                <span className="text-slate-200">{row.label}</span>
                <span>{row.note}</span>
              </div>
              <div className="mt-2 grid gap-2 sm:grid-cols-2 lg:grid-cols-5">
                {row.cells.map((cell) => (
                  <CoverageBadge key={`${row.id}-${cell.label}`} label={cell.label} active={cell.active} />
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </details>
  );
}
