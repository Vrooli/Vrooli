// Renders an alert banner when one or more models have transitioned to
// failed within the last hour. Sourced from the operational health summary
// (server-computed `failing_last_hour`).

import { AlertCircle } from "lucide-react";
import { Link } from "react-router-dom";
import { useHealthSummary } from "../../hooks/useOperationalStats";

export function ModelFailureAlertBanner() {
  const { data } = useHealthSummary();
  const failing = data?.failing_last_hour ?? [];
  if (failing.length === 0) {
    return null;
  }
  return (
    <div
      role="alert"
      className="mb-3 flex items-start gap-2 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm"
      data-testid="model-failure-banner"
    >
      <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-destructive" />
      <div className="space-y-1">
        <p className="font-medium text-destructive">
          {failing.length} model{failing.length === 1 ? "" : "s"} failed in the last hour
        </p>
        <ul className="text-xs text-muted-foreground">
          {failing.slice(0, 5).map((entry) => (
            <li key={`${entry.runner}|${entry.model}`} className="font-mono">
              {entry.runner} / {entry.model}
              {entry.reason ? ` — ${entry.reason}` : ""}
            </li>
          ))}
        </ul>
        <p className="text-xs">
          <Link to="/observability" className="underline">
            Open the health page →
          </Link>
        </p>
      </div>
    </div>
  );
}
