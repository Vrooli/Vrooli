import type { FixResult, RuleResult, RuleWithState } from "../lib/api";
import { RuleResultCard } from "./RuleResultCard";

export function ResultsPanel({
  results,
  rules,
  expandedRuleId,
  onToggleExpand,
  fixResults,
  onFix,
  fixPending,
  dryRun,
  onToggleDryRun
}: {
  results: RuleResult[];
  rules: RuleWithState[];
  expandedRuleId: string | null;
  onToggleExpand: (ruleId: string) => void;
  fixResults: Record<string, FixResult[]>;
  onFix: (ruleId: string, scenarioNames: string[], dryRun: boolean) => void;
  fixPending: boolean;
  dryRun: boolean;
  onToggleDryRun: () => void;
}) {
  const ruleMap = new Map(rules.map((r) => [r.id, r]));

  return (
    <section className="mt-8">
      <h2 className="text-xl font-semibold text-slate-50">Results</h2>
      <p className="mt-1 text-sm text-slate-300">
        Click a rule to expand findings grouped by scenario.
      </p>
      <div className="mt-4 grid gap-3">
        {results.map((result) => (
          <RuleResultCard
            key={result.rule_id}
            result={result}
            ruleDefinition={ruleMap.get(result.rule_id)}
            expanded={expandedRuleId === result.rule_id}
            onToggleExpand={() => onToggleExpand(result.rule_id)}
            fixResults={fixResults[result.rule_id]}
            onFix={onFix}
            fixPending={fixPending}
            dryRun={dryRun}
            onToggleDryRun={onToggleDryRun}
          />
        ))}
      </div>
    </section>
  );
}
