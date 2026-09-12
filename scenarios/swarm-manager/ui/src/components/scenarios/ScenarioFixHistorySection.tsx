/**
 * ScenarioFixHistorySection
 *
 * Renders the Fix History block on the scenario detail page. Lets the operator
 * partition fixes by Active / Archived / All and free-text search across
 * title and name. Mounted inside ScenarioCoverageSection so a single
 * /scenarios/{name}/context fetch covers all three blocks (goals,
 * orphans, fixes).
 *
 * The section answers: "has this scenario seen a related fix before?"
 */

import { useMemo, useState } from "react";
import { Wrench } from "lucide-react";
import { DetailSection } from "../detail/DetailSection";
import { EntityLink } from "../ui/entity-link";
import type { ScenarioFix, ScenarioFixHistory } from "../../services/scenarios-service";

type Scope = "active" | "archived" | "all";

const SCOPE_LABELS: Record<Scope, string> = {
  active: "Active",
  archived: "Archived",
  all: "All",
};

export interface ScenarioFixHistorySectionProps {
  fixes: ScenarioFixHistory;
}

export function ScenarioFixHistorySection({ fixes }: ScenarioFixHistorySectionProps) {
  const [scope, setScope] = useState<Scope>("active");
  const [search, setSearch] = useState("");

  const filtered = useMemo<ScenarioFix[]>(() => {
    const pool: ScenarioFix[] =
      scope === "active"
        ? fixes.active
        : scope === "archived"
          ? fixes.archived
          : [...fixes.active, ...fixes.archived];
    const q = search.trim().toLowerCase();
    if (!q) return pool;
    return pool.filter(
      (f) =>
        f.title.toLowerCase().includes(q) || f.name.toLowerCase().includes(q),
    );
  }, [scope, search, fixes]);

  const totals = `${fixes.active.length} active · ${fixes.archived.length} archived`;

  if (fixes.active.length === 0 && fixes.archived.length === 0) {
    return (
      <DetailSection title="Fix History" icon={Wrench} data-testid="scenario-fix-history">
        <p className="text-sm text-slate-400" data-testid="scenario-fix-history-empty">
          No recorded fixes for this scenario.
        </p>
      </DetailSection>
    );
  }

  return (
    <DetailSection
      title="Fix History"
      icon={Wrench}
      data-testid="scenario-fix-history"
    >
      <div className="space-y-3">
        <div className="flex flex-wrap items-center gap-2">
          <div
            className="inline-flex rounded-lg border border-slate-700/80 bg-slate-900/55 p-0.5"
            role="tablist"
            aria-label="Fix history scope"
          >
            {(["active", "archived", "all"] as Scope[]).map((s) => (
              <button
                key={s}
                type="button"
                role="tab"
                aria-selected={scope === s}
                data-testid={`scenario-fix-history-scope-${s}`}
                onClick={() => setScope(s)}
                className={
                  "rounded-md px-2.5 py-1 text-xs " +
                  (scope === s
                    ? "bg-slate-700/80 text-slate-100"
                    : "text-slate-400 hover:text-slate-200")
                }
              >
                {SCOPE_LABELS[s]}
              </button>
            ))}
          </div>
          <input
            type="search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Filter by title or name…"
            data-testid="scenario-fix-history-search"
            className="min-w-[160px] flex-1 rounded-md border border-slate-700/80 bg-slate-900/55 px-2 py-1 text-xs text-slate-100 placeholder:text-slate-500 focus:border-cyan-500/60 focus:outline-none"
          />
          <span className="text-[11px] text-slate-500">{totals}</span>
        </div>

        {filtered.length === 0 ? (
          <p
            className="text-sm text-slate-400"
            data-testid="scenario-fix-history-empty"
          >
            {emptyCopy(scope, fixes, search)}
          </p>
        ) : (
          <div className="grid gap-2" data-testid="scenario-fix-history-list">
            {filtered.map((fix) => (
              <div
                key={`${fix.name}-${fix.archivedAt ?? "active"}`}
                className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <EntityLink
                      entityType="backlog"
                      kind="fix"
                      name={fix.name}
                      label={fix.title || fix.name}
                    />
                    <p className="mt-1 truncate text-[11px] text-slate-500">
                      {fix.path}
                      {fix.goal ? ` · goal=${fix.goal}` : ""}
                    </p>
                  </div>
                  <div className="flex shrink-0 flex-col items-end gap-1 text-[11px] text-slate-500">
                    <span>
                      {fix.status}
                      {fix.priority > 0 ? ` · P${fix.priority}` : ""}
                    </span>
                    {fix.archivedAt && (
                      <span className="rounded-full border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 text-amber-300">
                        archived {fix.archivedAt}
                      </span>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </DetailSection>
  );
}

function emptyCopy(scope: Scope, fixes: ScenarioFixHistory, search: string): string {
  if (search.trim()) return "No fixes match that search.";
  if (scope === "active" && fixes.active.length === 0)
    return "No active fixes for this scenario.";
  if (scope === "archived" && fixes.archived.length === 0)
    return "No archived fixes for this scenario.";
  return "No fixes recorded for this scenario yet.";
}
