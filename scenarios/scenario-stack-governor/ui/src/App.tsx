import { useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowRight, Check, Shield, Terminal, X } from "lucide-react";
import { Button } from "./components/ui/button";
import { fetchHealth, fetchRules, putConfig, runEnabledRules, type RuleWithState, type RulesConfig } from "./lib/api";

function toggleRule(config: RulesConfig, id: string, enabled: boolean): RulesConfig {
  return {
    ...config,
    enabled_rules: {
      ...config.enabled_rules,
      [id]: enabled
    }
  };
}

function severityLabel(severity: string) {
  switch (severity) {
    case "error":
      return { label: "Error", className: "bg-red-500/15 text-red-200 border-red-500/30" };
    case "warn":
      return { label: "Warn", className: "bg-amber-500/15 text-amber-200 border-amber-500/30" };
    default:
      return { label: "Info", className: "bg-slate-500/15 text-slate-200 border-slate-500/30" };
  }
}

export default function App() {
  const qc = useQueryClient();

  const health = useQuery({ queryKey: ["health"], queryFn: fetchHealth });
  const rules = useQuery({ queryKey: ["rules"], queryFn: fetchRules });

  const config = useMemo(() => rules.data?.config, [rules.data]);

  const saveConfig = useMutation({
    mutationFn: (cfg: RulesConfig) => putConfig(cfg),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["rules"] })
  });

  const run = useMutation({ mutationFn: runEnabledRules });

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50 p-6">
      <div className="mx-auto w-full max-w-5xl">
        <header className="flex flex-col gap-2">
          <p className="text-sm uppercase tracking-[0.2em] text-slate-400">Developer Tools</p>
          <h1 className="text-3xl font-semibold">Scenario Stack Governor</h1>
          <p className="max-w-3xl text-slate-300">
            Rule packs that prevent repo-wide footguns by enforcing scenario stack invariants. Toggle what’s enabled in
            `scenarios/scenario-stack-governor/config/rules.json`, or manage it here.
          </p>
        </header>

        <section className="mt-6 grid gap-4 md:grid-cols-3">
          <div className="rounded-2xl border border-white/10 bg-white/5 p-5 backdrop-blur">
            <div className="flex items-center gap-2 text-slate-200">
              <Shield className="h-4 w-4" />
              <p className="text-sm font-medium">API Health</p>
            </div>
            {health.isLoading && <p className="mt-2 text-sm text-slate-300">Checking…</p>}
            {health.error && (
              <p className="mt-2 text-sm text-red-300">
                Unable to reach the API. Start the scenario with `vrooli scenario start scenario-stack-governor`.
              </p>
            )}
            {health.data && (
              <p className="mt-2 text-sm text-slate-200">
                <span className="text-slate-400">Status:</span> {health.data.status}
              </p>
            )}
            <Button className="mt-3" variant="outline" size="sm" onClick={() => health.refetch()}>
              Refresh <ArrowRight className="ml-2 h-4 w-4" />
            </Button>
          </div>

          <div className="rounded-2xl border border-white/10 bg-white/5 p-5 backdrop-blur md:col-span-2">
            <div className="flex items-center justify-between gap-4">
              <div className="flex items-center gap-2 text-slate-200">
                <Terminal className="h-4 w-4" />
                <p className="text-sm font-medium">Run Enabled Rules</p>
              </div>
              <Button disabled={!config || run.isPending} onClick={() => run.mutate()}>
                {run.isPending ? "Running…" : "Run now"}
              </Button>
            </div>
            {run.error && <p className="mt-3 text-sm text-red-300">Run failed. Check API logs.</p>}
            {run.data && (
              <div className="mt-3 text-sm text-slate-200">
                <p className="text-slate-400">Repo root: {run.data.repo_root}</p>
                <div className="mt-2 grid gap-2 md:grid-cols-2">
                  {run.data.results.map((res) => (
                    <div
                      key={res.rule_id}
                      className="rounded-xl border border-white/10 bg-black/20 p-3 flex items-start gap-2"
                    >
                      {res.passed ? <Check className="mt-0.5 h-4 w-4 text-emerald-300" /> : <X className="mt-0.5 h-4 w-4 text-red-300" />}
                      <div className="min-w-0">
                        <p className="font-medium">{res.rule_id}</p>
                        {!res.passed && res.findings?.length ? (
                          <p className="mt-1 text-slate-300">
                            {res.findings.filter((f) => f.level === "error").length} error(s),{" "}
                            {res.findings.filter((f) => f.level === "warn").length} warning(s)
                          </p>
                        ) : (
                          <p className="mt-1 text-slate-300">No findings.</p>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </section>

        <section className="mt-8">
          <div className="flex items-end justify-between gap-4">
            <div>
              <h2 className="text-xl font-semibold">Rules</h2>
              <p className="mt-1 text-sm text-slate-300">
                Each rule is documented in plain language and ships with evidence-backed failures.
              </p>
            </div>
          </div>

          {rules.isLoading && <p className="mt-4 text-sm text-slate-300">Loading rules…</p>}
          {rules.error && <p className="mt-4 text-sm text-red-300">Failed to load rules.</p>}

          {rules.data?.rules && config && (
            <div className="mt-4 grid gap-4">
              {rules.data.rules.map((rule) => (
                <RuleCard
                  key={rule.id}
                  rule={rule}
                  config={config}
                  saving={saveConfig.isPending}
                  onToggle={(enabled) => saveConfig.mutate(toggleRule(config, rule.id, enabled))}
                />
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}

function RuleCard({
  rule,
  config,
  saving,
  onToggle
}: {
  rule: RuleWithState;
  config: RulesConfig;
  saving: boolean;
  onToggle: (enabled: boolean) => void;
}) {
  const enabled = config.enabled_rules[rule.id] ?? false;
  const badge = severityLabel(rule.severity);

  return (
    <div className="rounded-2xl border border-white/10 bg-white/5 p-6 backdrop-blur">
      <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-lg font-semibold">{rule.title}</h3>
            <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs ${badge.className}`}>
              {badge.label}
            </span>
            <span className="text-xs text-slate-400">{rule.category}</span>
          </div>
          <p className="mt-2 text-sm text-slate-300">{rule.summary}</p>
          <details className="mt-3">
            <summary className="cursor-pointer text-sm text-slate-200">Why this matters</summary>
            <p className="mt-2 text-sm text-slate-300">{rule.why_important}</p>
          </details>
        </div>

        <div className="flex items-center gap-3 md:flex-col md:items-end">
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={enabled}
              disabled={saving}
              onChange={(e) => onToggle(e.target.checked)}
              className="h-4 w-4 accent-slate-100"
            />
            <span className="text-slate-200">{enabled ? "Enabled" : "Disabled"}</span>
          </label>
          <p className="text-xs text-slate-400">{saving ? "Saving…" : rule.id}</p>
        </div>
      </div>
    </div>
  );
}
