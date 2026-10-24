import { useState } from "react";
import { ShieldCheck, ShieldAlert, ShieldQuestion, Loader2, ChevronDown, ChevronRight } from "lucide-react";
import type { ScenarioIsolation } from "../hooks/useScenarioIsolation";

interface IsolationBadgeProps {
  isolation: ScenarioIsolation;
}

interface BadgeVisual {
  Icon: typeof ShieldCheck;
  className: string;
  headline: string;
  detail: string;
}

const VISUAL: Record<ScenarioIsolation["status"], BadgeVisual> = {
  routed: {
    Icon: ShieldCheck,
    className: "bg-green-900/40 border-green-700/50 text-green-200",
    headline: "Data isolation confirmed",
    detail: "Test-genie routes this scenario's database traffic through an isolated test pool. All workflow modes are safe to run.",
  },
  not_routed: {
    Icon: ShieldAlert,
    className: "bg-amber-900/40 border-amber-700/50 text-amber-200",
    headline: "Data isolation not available",
    detail: "Test-genie's auditor reports this scenario does not qualify for the routed test-db path. Inspect the violations and fix them before running mutating or destructive workflows.",
  },
  unknown: {
    Icon: ShieldQuestion,
    className: "bg-slate-800/60 border-slate-700/60 text-slate-200",
    headline: "Isolation status unavailable",
    detail: "GCT could not reach test-genie or the scenario is unknown to the auditor. Workflow runs may touch live state.",
  },
  loading: {
    Icon: Loader2,
    className: "bg-slate-800/40 border-slate-700/40 text-slate-300",
    headline: "Checking isolation status…",
    detail: "Asking test-genie whether this scenario qualifies for the routed test-db path.",
  },
};

export function IsolationBadge({ isolation }: IsolationBadgeProps) {
  const [open, setOpen] = useState(false);
  const visual = VISUAL[isolation.status];
  const { Icon } = visual;
  const spinning = isolation.status === "loading";

  const hasDetails = isolation.reasons.length > 0 || isolation.violations.length > 0;

  return (
    <div className={`w-full rounded-lg border px-3 py-2 ${visual.className}`}>
      <div className="flex items-start gap-2">
        <Icon className={`h-4 w-4 mt-0.5 shrink-0 ${spinning ? "animate-spin" : ""}`} />
        <div className="min-w-0 flex-1">
          <p className="text-xs font-semibold">{visual.headline}</p>
          <p className="text-[11px] opacity-80 mt-0.5">{visual.detail}</p>
        </div>
        {hasDetails && (
          <button
            type="button"
            onClick={() => setOpen(o => !o)}
            className="text-[10px] underline opacity-80 hover:opacity-100"
            aria-expanded={open}
          >
            {open ? (
              <span className="inline-flex items-center gap-1"><ChevronDown className="h-3 w-3" />Hide details</span>
            ) : (
              <span className="inline-flex items-center gap-1"><ChevronRight className="h-3 w-3" />Show details</span>
            )}
          </button>
        )}
      </div>
      {open && hasDetails && (
        <div className="mt-2 space-y-1.5">
          {isolation.reasons.length > 0 && (
            <ul className="list-disc pl-5 text-[11px] opacity-90">
              {isolation.reasons.map((reason, i) => (
                <li key={i}>{reason}</li>
              ))}
            </ul>
          )}
          {isolation.violations.length > 0 && (
            <ul className="text-[11px] opacity-90 space-y-0.5">
              {isolation.violations.map((v, i) => (
                <li key={i} className="font-mono">
                  [{v.severity?.toUpperCase()}] {v.rule_id}
                  {v.file ? ` — ${v.file}${v.line ? `:${v.line}` : ""}` : ""}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
