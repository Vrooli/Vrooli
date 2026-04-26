import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Shield, ArrowLeft, ChevronDown, ChevronRight, FileSearch, Search } from "lucide-react";
import {
  fetchStandards,
  evaluateRule,
  evaluateAllRules,
  type RuleEvalItem,
  type StandardRule,
} from "../lib/api";
import { Section } from "../components/ui/section";
import { ErrorAlert } from "../components/ui/error-alert";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Markdown } from "../components/markdown";

// [REQ:BM-REQ-API-STANDARDS] [REQ:BM-REQ-AUDIT-RULES] [REQ:BM-REQ-AUDIT-ENDPOINT]

interface StandardsPageProps {
  onNavigate: (path: string) => void;
}

type RuleEvalState = {
  pending: boolean;
  result?: RuleEvalItem;
  error?: string;
};

function severityClasses(severity: string): string {
  if (severity === "error") return "bg-red-500/20 text-red-300";
  if (severity === "warning") return "bg-amber-500/20 text-amber-300";
  return "bg-slate-500/20 text-slate-300";
}

function summaryColor(items: RuleEvalItem[]): string {
  if (items.every((r) => r.pass)) return "text-emerald-400";
  if (items.some((r) => !r.pass && r.severity === "error")) return "text-red-400";
  return "text-amber-400";
}

export default function StandardsPage({ onNavigate }: StandardsPageProps) {
  const { data: standards, isLoading, error, refetch } = useQuery({
    queryKey: ["standards"],
    queryFn: fetchStandards,
  });

  const [scenario, setScenario] = useState("");
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const [evalState, setEvalState] = useState<Record<string, RuleEvalState>>({});
  const [batch, setBatch] = useState<{
    pending: boolean;
    items?: RuleEvalItem[];
    error?: string;
    scenario?: string;
  }>({ pending: false });

  const trimmedScenario = scenario.trim();
  const canEvaluate = trimmedScenario.length > 0;

  const toggle = (id: string) =>
    setExpanded((prev) => ({ ...prev, [id]: !prev[id] }));

  const runOne = async (ruleId: string) => {
    if (!canEvaluate) return;
    setEvalState((s) => ({ ...s, [ruleId]: { pending: true } }));
    try {
      const res = await evaluateRule(trimmedScenario, ruleId);
      const item = res.results[0];
      setEvalState((s) => ({ ...s, [ruleId]: { pending: false, result: item } }));
    } catch (e) {
      const msg = e instanceof Error ? e.message : "evaluation failed";
      setEvalState((s) => ({ ...s, [ruleId]: { pending: false, error: msg } }));
    }
  };

  const runAll = async () => {
    if (!canEvaluate) return;
    setBatch({ pending: true });
    try {
      const res = await evaluateAllRules(trimmedScenario);
      setBatch({ pending: false, items: res.results, scenario: res.scenario });
      // Fan out into per-rule state so cards reflect the latest result.
      setEvalState((prev) => {
        const next = { ...prev };
        for (const item of res.results) {
          next[item.rule_id] = { pending: false, result: item };
        }
        return next;
      });
    } catch (e) {
      const msg = e instanceof Error ? e.message : "evaluation failed";
      setBatch({ pending: false, error: msg });
    }
  };

  return (
    <div data-testid="standards-page">
      <button
        onClick={() => onNavigate("/brands")}
        className="flex items-center gap-2 text-sm text-slate-400 hover:text-slate-50 mb-4 transition-colors"
        data-testid="back-to-brands"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Library
      </button>

      <div className="flex items-center gap-3 mb-2">
        <Shield className="h-6 w-6 text-slate-50" />
        <h1 className="text-2xl font-bold text-slate-50">Brand Standards</h1>
      </div>

      <p className="text-slate-400 text-sm mb-6">
        These rules define the branding standards enforced across all scenarios. Expand any rule for
        a full description, examples, and fix instructions, or run it against a scenario.
      </p>

      <Section title="Run audits" testId="standards-run">
        <div className="flex items-center gap-3">
          <div className="relative flex-1">
            <FileSearch className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
            <Input
              variant="search"
              placeholder="Enter scenario name..."
              value={scenario}
              onChange={(e) => setScenario(e.target.value)}
              data-testid="standards-scenario-input"
            />
          </div>
          <Button
            onClick={runAll}
            disabled={!canEvaluate || batch.pending}
            data-testid="standards-scan-all-btn"
          >
            <Search className="mr-2 h-4 w-4" />
            {batch.pending ? "Scanning..." : "Scan All Rules"}
          </Button>
        </div>

        {batch.error && (
          <p className="text-red-400 text-xs mt-3" data-testid="standards-scan-all-error">
            {batch.error}
          </p>
        )}
        {batch.items && (
          <div
            className={`mt-3 text-sm font-medium ${summaryColor(batch.items)}`}
            data-testid="standards-scan-summary"
          >
            {batch.items.filter((r) => r.pass).length} / {batch.items.length} rules passing
            {batch.scenario ? ` for ${batch.scenario}` : ""}
          </div>
        )}
      </Section>

      {isLoading && (
        <div className="text-center text-slate-400 py-12" data-testid="standards-loading">Loading standards...</div>
      )}

      {error && (
        <ErrorAlert
          error={error}
          fallbackMessage="Failed to load standards."
          onRetry={() => refetch()}
          testId="standards-error"
        />
      )}

      {standards && (
        <Section title="Branding Rules" testId="standards-list" className="mt-4">
          <div className="space-y-3">
            {standards.rules.map((rule) => (
              <RuleCard
                key={rule.id}
                rule={rule}
                expanded={!!expanded[rule.id]}
                onToggle={() => toggle(rule.id)}
                evalState={evalState[rule.id]}
                onCheck={() => runOne(rule.id)}
                canEvaluate={canEvaluate}
              />
            ))}
          </div>

          {standards.rules.length === 0 && (
            <p className="text-slate-500 text-sm py-4" data-testid="standards-empty">No standards defined.</p>
          )}
        </Section>
      )}
    </div>
  );
}

interface RuleCardProps {
  rule: StandardRule;
  expanded: boolean;
  onToggle: () => void;
  evalState?: RuleEvalState;
  onCheck: () => void;
  canEvaluate: boolean;
}

function RuleCard({ rule, expanded, onToggle, evalState, onCheck, canEvaluate }: RuleCardProps) {
  const result = evalState?.result;
  return (
    <div
      className="rounded-lg bg-white/5 border border-white/5 overflow-hidden"
      data-testid={`standard-${rule.id}`}
    >
      <button
        type="button"
        onClick={onToggle}
        className="w-full text-left px-3 py-3 hover:bg-white/5 transition-colors flex items-start gap-2"
        data-testid={`standard-${rule.id}-toggle`}
        aria-expanded={expanded}
      >
        <span className="mt-0.5 text-slate-400">
          {expanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
        </span>
        <div className="flex-1">
          <div className="flex items-center justify-between mb-1">
            <span className="text-slate-100 font-medium text-sm">{rule.name}</span>
            <div className="flex items-center gap-2">
              {result && (
                <span
                  className={`h-2 w-2 rounded-full ${result.pass ? "bg-emerald-500" : "bg-red-500"}`}
                  data-testid={`standard-${rule.id}-result-dot`}
                />
              )}
              <span className={`text-xs px-2 py-0.5 rounded-full ${severityClasses(rule.severity)}`}>
                {rule.severity}
              </span>
            </div>
          </div>
          <p className="text-slate-400 text-xs">{rule.description}</p>
          <p className="text-slate-600 text-xs mt-1">ID: {rule.id}</p>
        </div>
      </button>

      {expanded && (
        <div
          className="px-3 pb-3 pt-1 border-t border-white/5 space-y-4"
          data-testid={`standard-${rule.id}-details`}
        >
          {rule.detailed_description && (
            <div>
              <h3 className="text-xs uppercase tracking-wide text-slate-500 mb-1">What it checks</h3>
              <Markdown content={rule.detailed_description} />
            </div>
          )}

          {rule.target_files && rule.target_files.length > 0 && (
            <div>
              <h3 className="text-xs uppercase tracking-wide text-slate-500 mb-1">Target files</h3>
              <ul className="text-xs font-mono text-slate-300 space-y-0.5">
                {rule.target_files.map((f) => (
                  <li key={f}>• {f}</li>
                ))}
              </ul>
            </div>
          )}

          {rule.passing_example && (
            <div>
              <h3 className="text-xs uppercase tracking-wide text-emerald-400/80 mb-1">Passing example</h3>
              <Markdown content={rule.passing_example} />
            </div>
          )}

          {rule.failing_example && (
            <div>
              <h3 className="text-xs uppercase tracking-wide text-red-400/80 mb-1">Failing example</h3>
              <Markdown content={rule.failing_example} />
            </div>
          )}

          {rule.fix_instructions && (
            <div>
              <h3 className="text-xs uppercase tracking-wide text-slate-500 mb-1">How to fix</h3>
              <Markdown content={rule.fix_instructions} />
            </div>
          )}

          {rule.severity_rationale && (
            <div>
              <h3 className="text-xs uppercase tracking-wide text-slate-500 mb-1">Severity rationale</h3>
              <Markdown content={rule.severity_rationale} />
            </div>
          )}

          <div className="pt-2 border-t border-white/5 flex items-center gap-3">
            <Button
              size="sm"
              variant="outline"
              onClick={(e) => {
                e.stopPropagation();
                onCheck();
              }}
              disabled={!canEvaluate || evalState?.pending}
              data-testid={`standard-${rule.id}-check-btn`}
            >
              {evalState?.pending ? "Checking..." : "Check Scenario"}
            </Button>
            {!canEvaluate && (
              <span className="text-xs text-slate-500">Enter a scenario name above first.</span>
            )}
            {result && (
              <span
                className={`text-xs ${result.pass ? "text-emerald-400" : "text-red-400"}`}
                data-testid={`standard-${rule.id}-result`}
              >
                {result.pass ? "✓ pass" : "✗ fail"} — {result.message}
              </span>
            )}
            {evalState?.error && (
              <span className="text-xs text-red-400" data-testid={`standard-${rule.id}-error`}>
                {evalState.error}
              </span>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
