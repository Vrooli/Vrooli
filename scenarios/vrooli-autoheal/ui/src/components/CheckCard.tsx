// Individual health check result card
// [REQ:UI-HEALTH-001] [REQ:UI-HEALTH-002]
import { useState } from "react";
import { StatusIcon } from "./StatusIcon";
import type { HealthResult } from "../lib/api";
import { selectors } from "../consts/selectors";

interface CheckCardProps {
  check: HealthResult;
}

export function CheckCard({ check }: CheckCardProps) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div
      className="rounded-lg border border-white/10 bg-white/5 p-4 hover:bg-white/[0.07] transition-colors cursor-pointer"
      onClick={() => setExpanded(!expanded)}
      data-testid={selectors.checkCard}
    >
      <div className="flex items-start gap-3">
        <StatusIcon status={check.status} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between gap-2">
            <h3 className="font-medium text-slate-200 truncate">{check.checkId}</h3>
            <span className="text-xs text-slate-500">{check.duration}ms</span>
          </div>
          <p className="text-sm text-slate-400 mt-1">{check.message}</p>

          {expanded && check.details && Object.keys(check.details).length > 0 && (
            <div className="mt-3 p-3 rounded-lg bg-black/30 text-xs font-mono">
              <pre className="overflow-x-auto">{JSON.stringify(check.details, null, 2)}</pre>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
