import { Check, ChevronDown, ChevronRight, X } from "lucide-react";
import type { Finding, FixResult, RuleResult, RuleWithState } from "../lib/api";
import { FindingItem } from "./FindingItem";
import { FixActions, FixScenarioButton } from "./FixActions";
import { FixResultsPanel } from "./FixResultsPanel";

function groupByScenario(findings: Finding[]): Map<string, Finding[]> {
  const groups = new Map<string, Finding[]>();
  for (const f of findings) {
    const key = f.scenario_name || "(all)";
    const list = groups.get(key) || [];
    list.push(f);
    groups.set(key, list);
  }
  return groups;
}

export function RuleResultCard({
  result,
  ruleDefinition,
  expanded,
  onToggleExpand,
  fixResults,
  onFix,
  fixPending,
  dryRun,
  onToggleDryRun
}: {
  result: RuleResult;
  ruleDefinition?: RuleWithState;
  expanded: boolean;
  onToggleExpand: () => void;
  fixResults?: FixResult[];
  onFix: (ruleId: string, scenarioNames: string[], dryRun: boolean) => void;
  fixPending: boolean;
  dryRun: boolean;
  onToggleDryRun: () => void;
}) {
  const findings = result.findings || [];
  const errorCount = result.error_count;
  const warnCount = result.warn_count;
  const infoCount = findings.length - errorCount - warnCount;
  const scenarioGroups = groupByScenario(findings);
  const scenarioNames = [...scenarioGroups.keys()].filter((k) => k !== "(all)");
  const fixable = ruleDefinition?.fixable ?? false;

  return (
    <div className="rounded-2xl border border-white/10 bg-white/5 backdrop-blur">
      <button
        className="flex w-full items-start gap-3 p-5 text-left"
        onClick={onToggleExpand}
      >
        {expanded ? (
          <ChevronDown className="mt-0.5 h-4 w-4 shrink-0 text-slate-400" />
        ) : (
          <ChevronRight className="mt-0.5 h-4 w-4 shrink-0 text-slate-400" />
        )}
        {result.passed ? (
          <Check className="mt-0.5 h-4 w-4 shrink-0 text-emerald-300" />
        ) : (
          <X className="mt-0.5 h-4 w-4 shrink-0 text-red-300" />
        )}
        <div className="min-w-0 flex-1">
          <p className="font-medium text-slate-50">{ruleDefinition?.title || result.rule_id}</p>
          <p className="mt-0.5 text-sm text-slate-400">
            {result.passed ? (
              "No findings"
            ) : (
              <>
                {errorCount > 0 && <span className="text-red-300">{errorCount} error{errorCount !== 1 ? "s" : ""}</span>}
                {errorCount > 0 && (warnCount > 0 || infoCount > 0) && ", "}
                {warnCount > 0 && <span className="text-amber-300">{warnCount} warning{warnCount !== 1 ? "s" : ""}</span>}
                {warnCount > 0 && infoCount > 0 && ", "}
                {infoCount > 0 && <span className="text-slate-300">{infoCount} info</span>}
                {errorCount === 0 && warnCount === 0 && infoCount === 0 && (
                  <span className="text-red-300">Failed (no details available)</span>
                )}
              </>
            )}
          </p>
        </div>
      </button>

      {expanded && (
        <div className="border-t border-white/5 px-5 pb-5 pt-3">
          {[...scenarioGroups.entries()].map(([scenarioName, scenarioFindings]) => (
            <div key={scenarioName} className="mt-3 first:mt-0">
              <div className="flex items-center gap-2">
                <h4 className="text-sm font-medium text-slate-300">{scenarioName}</h4>
                <span className="text-xs text-slate-500">({scenarioFindings.length})</span>
                {fixable && scenarioName !== "(all)" && (
                  <FixScenarioButton
                    ruleId={result.rule_id}
                    scenarioName={scenarioName}
                    dryRun={dryRun}
                    onFix={onFix}
                    isPending={fixPending}
                  />
                )}
              </div>
              <div className="mt-2 space-y-2">
                {scenarioFindings.map((f, i) => (
                  <FindingItem key={i} finding={f} />
                ))}
              </div>
            </div>
          ))}

          {fixable && scenarioNames.length > 0 && (
            <FixActions
              ruleId={result.rule_id}
              scenarioNames={scenarioNames}
              fixable={fixable}
              dryRun={dryRun}
              onToggleDryRun={onToggleDryRun}
              onFix={onFix}
              isPending={fixPending}
            />
          )}

          {fixResults && fixResults.length > 0 && <FixResultsPanel results={fixResults} />}
        </div>
      )}
    </div>
  );
}
