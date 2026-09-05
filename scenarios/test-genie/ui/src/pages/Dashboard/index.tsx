import { Button } from "../../components/ui/button";
import { useExecutions } from "../../hooks/useExecutions";
import { useScenarios } from "../../hooks/useScenarios";
import { useUIStore } from "../../stores/uiStore";
import { formatRelative } from "../../lib/formatters";

export function DashboardPage() {
  const { setActiveTab, navigateToScenarioDetail } = useUIStore();
  const { lastFailedExecution, executions } = useExecutions();
  const { scenarioDirectoryEntries, catalogStats } = useScenarios();
  const focus = lastFailedExecution ?? executions[0];
  return <div className="space-y-6">
    <section className="rounded-3xl border border-white/10 bg-white/[0.04] p-8"><p className="text-xs font-semibold uppercase tracking-[0.3em] text-slate-400">Test Genie</p><h1 className="mt-3 text-3xl font-semibold">Execution evidence, then remediation</h1><p className="mt-4 max-w-3xl text-base text-slate-300">Run descriptor-backed checks, inspect the structured findings, and launch one evidence-bound remediation job only when the run supports it.</p><div className="mt-6 flex flex-wrap gap-3"><Button onClick={() => setActiveTab("runs")}>Browse scenarios</Button>{focus && <Button variant="outline" onClick={() => navigateToScenarioDetail(focus.scenarioName)}>Inspect latest findings</Button>}</div></section>
    <section className="grid gap-4 md:grid-cols-3"><article className="rounded-2xl border border-white/10 bg-black/20 p-5"><p className="text-xs uppercase tracking-wide text-slate-400">Scenarios</p><p className="mt-2 text-3xl font-semibold">{catalogStats.tracked}</p><p className="text-sm text-slate-400">{catalogStats.failing} with a failed latest run</p></article><article className="rounded-2xl border border-white/10 bg-black/20 p-5"><p className="text-xs uppercase tracking-wide text-slate-400">Latest execution</p><p className="mt-2 text-lg font-semibold">{focus?.scenarioName ?? "No execution yet"}</p><p className="text-sm text-slate-400">{focus ? `${focus.success ? "Passed" : "Findings available"} ${formatRelative(focus.completedAt)}` : "Run a scenario to establish evidence."}</p></article><article className="rounded-2xl border border-white/10 bg-black/20 p-5"><p className="text-xs uppercase tracking-wide text-slate-400">Workflow</p><p className="mt-2 text-lg font-semibold">Findings-first</p><p className="text-sm text-slate-400">Agent completion is provisional until a server-owned rerun compares stable finding IDs.</p></article></section>
    {scenarioDirectoryEntries.length === 0 && <p className="text-sm text-slate-400">No scenarios are available yet.</p>}
  </div>;
}
