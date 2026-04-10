import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  fetchHealth,
  fetchRules,
  fetchScenarios,
  fixRules,
  putConfig,
  runRules,
  type FixResult,
  type RulesConfig
} from "./lib/api";
import { HealthStatus } from "./components/HealthStatus";
import { RunControls } from "./components/RunControls";
import { ScenarioPicker } from "./components/ScenarioPicker";
import { RuleSelector } from "./components/RuleSelector";
import { ResultsPanel } from "./components/ResultsPanel";
import { ConfigPanel } from "./components/ConfigPanel";
import { DiffReviewModal } from "./components/DiffReviewModal";

function toggleRule(config: RulesConfig, id: string, enabled: boolean): RulesConfig {
  return {
    ...config,
    enabled_rules: {
      ...config.enabled_rules,
      [id]: enabled
    }
  };
}

export default function App() {
  const qc = useQueryClient();

  const health = useQuery({ queryKey: ["health"], queryFn: fetchHealth });
  const rules = useQuery({ queryKey: ["rules"], queryFn: fetchRules });
  const scenarios = useQuery({ queryKey: ["scenarios"], queryFn: fetchScenarios });

  const config = useMemo(() => rules.data?.config, [rules.data]);

  // Selected rules for next run (initialized from enabled rules).
  const [selectedRuleIds, setSelectedRuleIds] = useState<Set<string>>(new Set());
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (rules.data && !initialized) {
      const enabled = new Set(
        rules.data.rules.filter((r) => r.enabled).map((r) => r.id)
      );
      setSelectedRuleIds(enabled);
      setInitialized(true);
    }
  }, [rules.data, initialized]);

  // Scenario selection.
  const [allScenarios, setAllScenarios] = useState(true);
  const [selectedScenarios, setSelectedScenarios] = useState<string[]>([]);

  // Results state.
  const [expandedRuleId, setExpandedRuleId] = useState<string | null>(null);
  const [fixResultsMap, setFixResultsMap] = useState<Record<string, FixResult[]>>({});
  const [dryRun, setDryRun] = useState(false);

  // Diff review modal state for two-phase fix flow.
  const [pendingReview, setPendingReview] = useState<{ ruleId: string; scenarioNames: string[]; results: FixResult[] } | null>(null);

  // Config panel collapse state.
  const [configExpanded, setConfigExpanded] = useState(true);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const resultsRef = useRef<HTMLDivElement>(null);

  // Mutations.
  const saveConfig = useMutation({
    mutationFn: (cfg: RulesConfig) => putConfig(cfg),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["rules"] })
  });

  const run = useMutation({
    mutationFn: () => runRules(allScenarios ? undefined : selectedScenarios),
    onSuccess: () => {
      setFixResultsMap({});
      setConfigExpanded(false);
      setTimeout(() => {
        const container = scrollContainerRef.current;
        const target = resultsRef.current;
        if (container && target) {
          container.scrollTo({ top: target.offsetTop - container.offsetTop, behavior: "smooth" });
        }
      }, 100);
    }
  });

  const fix = useMutation({
    mutationFn: (args: { ruleId: string; scenarioNames: string[]; dryRun: boolean }) =>
      fixRules({
        scenario_names: args.scenarioNames,
        rule_ids: [args.ruleId],
        dry_run: args.dryRun
      }),
    onSuccess: (data, variables) => {
      if (variables.dryRun) {
        setFixResultsMap((prev) => ({
          ...prev,
          [variables.ruleId]: data.results
        }));
      }
    }
  });

  // Preview fix: dry-run to get diffs, then show review modal
  const previewFix = useMutation({
    mutationFn: (args: { ruleId: string; scenarioNames: string[] }) =>
      fixRules({
        scenario_names: args.scenarioNames,
        rule_ids: [args.ruleId],
        dry_run: true
      }),
    onSuccess: (data, variables) => {
      setPendingReview({ ruleId: variables.ruleId, scenarioNames: variables.scenarioNames, results: data.results });
    }
  });

  // Apply fix: actually write changes, then update results and close modal
  const applyFix = useMutation({
    mutationFn: (args: { ruleId: string; scenarioNames: string[] }) =>
      fixRules({
        scenario_names: args.scenarioNames,
        rule_ids: [args.ruleId],
        dry_run: false
      }),
    onSuccess: (data, variables) => {
      setFixResultsMap((prev) => ({
        ...prev,
        [variables.ruleId]: data.results
      }));
      setPendingReview(null);
    }
  });

  const handleToggleSelected = useCallback((id: string) => {
    setSelectedRuleIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const handleSelectAll = useCallback(() => {
    if (rules.data) {
      setSelectedRuleIds(new Set(rules.data.rules.map((r) => r.id)));
    }
  }, [rules.data]);

  const handleSelectNone = useCallback(() => {
    setSelectedRuleIds(new Set());
  }, []);

  const handleToggleScenario = useCallback((name: string) => {
    setSelectedScenarios((prev) =>
      prev.includes(name) ? prev.filter((s) => s !== name) : [...prev, name]
    );
  }, []);

  const handleFix = useCallback(
    (ruleId: string, scenarioNames: string[], isDryRun: boolean) => {
      if (isDryRun) {
        // Dry run: show inline results directly
        fix.mutate({ ruleId, scenarioNames, dryRun: true });
      } else {
        // Real fix: preview first (dry-run to get diffs), then show review modal
        previewFix.mutate({ ruleId, scenarioNames });
      }
    },
    [fix, previewFix]
  );

  const handleApplyReview = useCallback(() => {
    if (pendingReview) {
      applyFix.mutate({ ruleId: pendingReview.ruleId, scenarioNames: pendingReview.scenarioNames });
    }
  }, [pendingReview, applyFix]);

  const handleCancelReview = useCallback(() => {
    setPendingReview(null);
  }, []);

  const scenarioList = scenarios.data?.scenarios ?? [];
  const noRulesSelected = selectedRuleIds.size === 0;

  return (
    <div className="h-full flex flex-col overflow-hidden bg-slate-950 text-slate-50">
      <div ref={scrollContainerRef} className="flex-1 overflow-auto p-6">
      <div className="mx-auto w-full max-w-5xl">
        <header className="flex flex-col gap-2">
          <p className="text-sm uppercase tracking-[0.2em] text-slate-400">Developer Tools</p>
          <h1 className="text-3xl font-semibold">Scenario Stack Governor</h1>
          <p className="max-w-3xl text-slate-300">
            Rule packs that prevent repo-wide footguns by enforcing scenario stack invariants. Toggle
            what's enabled, select scenarios, and run checks with detailed results.
          </p>
        </header>

        {/* Health status */}
        <section className="mt-6">
          <HealthStatus
            data={health.data}
            isLoading={health.isLoading}
            error={health.error}
            onRefresh={() => health.refetch()}
          />
        </section>

        {/* Configuration + Run Controls */}
        <ConfigPanel
          expanded={configExpanded}
          onToggle={() => setConfigExpanded((prev) => !prev)}
          selectedRuleCount={selectedRuleIds.size}
          totalRuleCount={rules.data?.rules.length ?? 0}
          allScenarios={allScenarios}
          selectedScenarioCount={selectedScenarios.length}
          runControls={
            <>
              <RunControls
                disabled={noRulesSelected || !config}
                isPending={run.isPending}
                onRun={() => run.mutate()}
              />
              {run.error && <p className="mt-1 text-xs text-red-300">Run failed: {run.error instanceof Error ? run.error.message : String(run.error)}</p>}
            </>
          }
        >
          <ScenarioPicker
            scenarios={scenarioList}
            selectedScenarios={selectedScenarios}
            allSelected={allScenarios}
            onToggleAll={() => setAllScenarios((prev) => !prev)}
            onToggleScenario={handleToggleScenario}
          />
          <div>
            {rules.isLoading && <p className="text-sm text-slate-300">Loading rules...</p>}
            {rules.error && <p className="text-sm text-red-300">Failed to load rules.</p>}
            {rules.data?.rules && config && (
              <RuleSelector
                rules={rules.data.rules}
                config={config}
                selectedRuleIds={selectedRuleIds}
                saving={saveConfig.isPending}
                onToggleEnabled={(id, enabled) => saveConfig.mutate(toggleRule(config, id, enabled))}
                onToggleSelected={handleToggleSelected}
                onSelectAll={handleSelectAll}
                onSelectNone={handleSelectNone}
              />
            )}
          </div>
        </ConfigPanel>

        {/* Results */}
        <div ref={resultsRef} />
        {run.data && rules.data?.rules && (
          <ResultsPanel
            results={run.data.results}
            rules={rules.data.rules}
            expandedRuleId={expandedRuleId}
            onToggleExpand={(id) => setExpandedRuleId((prev) => (prev === id ? null : id))}
            fixResults={fixResultsMap}
            onFix={handleFix}
            fixPending={fix.isPending || previewFix.isPending}
            dryRun={dryRun}
            onToggleDryRun={() => setDryRun((prev) => !prev)}
          />
        )}
      </div>
      </div>

      <DiffReviewModal
        open={pendingReview !== null}
        results={pendingReview?.results ?? []}
        onApply={handleApplyReview}
        onCancel={handleCancelReview}
        applying={applyFix.isPending}
      />
    </div>
  );
}
