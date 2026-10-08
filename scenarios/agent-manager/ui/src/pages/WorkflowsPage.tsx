import { useEffect, useMemo, useState } from "react";
import { GitBranch, RefreshCw, ShieldCheck } from "lucide-react";
import type { WorkflowExecution } from "@vrooli/proto-types/agent-manager/v1/domain/workflow_pb";
import { WorkflowExecutionStatus } from "@vrooli/proto-types/agent-manager/v1/domain/workflow_pb";
import { useWorkflowExecutions, type WorkflowTraceView } from "../hooks/useApi";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";

const statusNames: Record<number, string> = {
  [WorkflowExecutionStatus.PENDING]: "pending",
  [WorkflowExecutionStatus.RUNNING]: "running",
  [WorkflowExecutionStatus.WAITING]: "waiting",
  [WorkflowExecutionStatus.SUCCEEDED]: "succeeded",
  [WorkflowExecutionStatus.BLOCKED]: "blocked",
  [WorkflowExecutionStatus.ABSTAINED]: "abstained",
  [WorkflowExecutionStatus.BUDGET_EXHAUSTED]: "budget exhausted",
  [WorkflowExecutionStatus.FAILED]: "failed",
  [WorkflowExecutionStatus.CANCELLED]: "cancelled",
  [WorkflowExecutionStatus.CANCELLING]: "cancelling",
};

function statusName(status: WorkflowExecution["status"]): string {
  return statusNames[status] ?? "unknown";
}

function shortId(value: string): string {
  return value ? value.slice(0, 8) : "—";
}

export function WorkflowsPage() {
  const workflows = useWorkflowExecutions();
  const getTrace = workflows.getTrace;
  const [selectedId, setSelectedId] = useState<string>("");
  const [trace, setTrace] = useState<WorkflowTraceView | null>(null);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [signalName, setSignalName] = useState("");
  const [signalPayload, setSignalPayload] = useState("{}");

  const selected = useMemo(
    () => workflows.data?.find((execution) => execution.id === selectedId) ?? workflows.data?.[0],
    [selectedId, workflows.data],
  );

  useEffect(() => {
    if (!selected) {
      setTrace(null);
      return;
    }
    setSelectedId(selected.id);
    setDetailError(null);
    void getTrace(selected.id).then(setTrace).catch((error: unknown) => {
      setDetailError(error instanceof Error ? error.message : "Failed to load workflow trace");
    });
  }, [selected, getTrace]);

  const runControl = async (operation: "cancel" | "retry" | "resume") => {
    if (!selected) return;
    setBusy(true);
    setDetailError(null);
    try {
      await workflows.control(selected, operation);
      setTrace(await workflows.getTrace(selected.id));
    } catch (error) {
      setDetailError(error instanceof Error ? error.message : `Failed to ${operation} workflow`);
    } finally {
      setBusy(false);
    }
  };

  const sendSignal = async () => {
    if (!selected || signalName.trim() === "") return;
    setBusy(true);
    setDetailError(null);
    try {
      const payload: unknown = JSON.parse(signalPayload);
      await workflows.signal(selected, signalName.trim(), payload);
      setTrace(await workflows.getTrace(selected.id));
      setSignalName("");
      setSignalPayload("{}");
    } catch (error) {
      setDetailError(error instanceof Error ? error.message : "Failed to signal workflow");
    } finally {
      setBusy(false);
    }
  };

  return (
    <section className="h-full overflow-auto p-4 sm:p-6" aria-labelledby="workflow-console-title">
      <div className="mx-auto max-w-7xl space-y-4">
        <header className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 id="workflow-console-title" className="flex items-center gap-2 text-xl font-semibold">
              <GitBranch className="h-5 w-5 text-primary" /> Workflow executions
            </h1>
            <p className="mt-1 text-sm text-muted-foreground">Pinned revisions, independent attempts, durable waits, child lineage, and budget usage.</p>
          </div>
          <Button variant="outline" size="sm" onClick={() => void workflows.refetch()} disabled={workflows.loading}>
            <RefreshCw className="mr-2 h-4 w-4" /> Refresh
          </Button>
        </header>

        <div className="flex items-center gap-2 rounded-md border border-border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
          <ShieldCheck className="h-4 w-4 shrink-0 text-success" /> Routine inspection is metadata-only. Inputs, prompts, journal payloads, and results are redacted.
        </div>

        {workflows.error ? <div role="alert" className="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive">{workflows.error}</div> : null}
        {workflows.loading && !workflows.data?.length ? <div className="rounded-md border border-border p-8 text-center text-sm text-muted-foreground">Loading workflow history…</div> : null}
        {!workflows.loading && !workflows.data?.length ? <div className="rounded-md border border-dashed border-border p-8 text-center text-sm text-muted-foreground">No workflow executions yet. Start one through the generated API or CLI.</div> : null}

        {workflows.data?.length ? (
          <div className="grid min-h-[32rem] gap-4 lg:grid-cols-[minmax(18rem,0.8fr)_minmax(0,1.7fr)]">
            <div className="overflow-hidden rounded-md border border-border bg-card" aria-label="Workflow execution history">
              {workflows.data.map((execution) => (
                <Button
                  key={execution.id}
                  type="button"
                  onClick={() => setSelectedId(execution.id)}
                  aria-pressed={selected?.id === execution.id}
                  variant="ghost"
                  className={`h-auto block w-full rounded-none border-b border-border px-3 py-3 text-left whitespace-normal last:border-b-0 ${selected?.id === execution.id ? "bg-accent" : "hover:bg-muted/50"}`}
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="truncate font-medium">{execution.workflowKey}</span>
                    <Badge variant="outline">{statusName(execution.status)}</Badge>
                  </div>
                  <div className="mt-1 flex flex-wrap gap-x-3 text-xs text-muted-foreground">
                    <span>{shortId(execution.id)}</span><span>node {execution.currentNodeId || "—"}</span><span>v{execution.version.toString()}</span><span>depth {execution.depth}</span>
                  </div>
                </Button>
              ))}
            </div>

            {selected ? (
              <article className="space-y-4 rounded-md border border-border bg-card p-4" aria-label={`Workflow ${selected.workflowKey} details`}>
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <h2 className="font-semibold">{selected.workflowKey}</h2>
                    <p className="break-all font-mono text-xs text-muted-foreground">{selected.id}</p>
                    <p className="mt-1 text-xs text-muted-foreground">digest {selected.definitionDigest} · parent {shortId(selected.parentExecutionId)} · parent attempt {shortId(selected.parentAttemptId)}</p>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <Button size="sm" variant="outline" onClick={() => void runControl("resume")} disabled={busy}>Resume</Button>
                    <Button size="sm" variant="outline" onClick={() => void runControl("retry")} disabled={busy}>Retry</Button>
                    <Button size="sm" variant="destructive" onClick={() => void runControl("cancel")} disabled={busy}>Cancel</Button>
                  </div>
                </div>

                {detailError ? <div role="alert" className="rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive">{detailError}</div> : null}

                <dl className="grid gap-2 text-sm sm:grid-cols-3">
                  <div className="rounded bg-muted/40 p-2"><dt className="text-xs text-muted-foreground">Turns / tokens</dt><dd>{selected.budgetUsage?.turns ?? 0} / {selected.budgetUsage?.tokens ?? 0}</dd></div>
                  <div className="rounded bg-muted/40 p-2"><dt className="text-xs text-muted-foreground">Cost / attempts</dt><dd>${(selected.budgetUsage?.costUsd ?? 0).toFixed(4)} / {selected.budgetUsage?.nodeAttempts ?? 0}</dd></div>
                  <div className="rounded bg-muted/40 p-2"><dt className="text-xs text-muted-foreground">Children / retries</dt><dd>{selected.budgetUsage?.children ?? 0} / {selected.budgetUsage?.retries ?? 0}</dd></div>
                </dl>

                {selected.status === WorkflowExecutionStatus.WAITING ? (
                  <div className="grid gap-2 rounded-md border border-border p-3 sm:grid-cols-[1fr_1.3fr_auto]">
                    <label className="text-xs text-muted-foreground">Signal name<Input value={signalName} onChange={(event) => setSignalName(event.target.value)} className="mt-1" /></label>
                    <label className="text-xs text-muted-foreground">JSON payload<Input value={signalPayload} onChange={(event) => setSignalPayload(event.target.value)} className="mt-1 font-mono" /></label>
                    <Button size="sm" className="self-end" onClick={() => void sendSignal()} disabled={busy || signalName.trim() === ""}>Signal</Button>
                  </div>
                ) : null}

                <section aria-labelledby="attempts-title">
                  <h3 id="attempts-title" className="mb-2 text-sm font-semibold">Attempts and identity</h3>
                  {!trace ? <p className="text-sm text-muted-foreground">Loading trace…</p> : trace.attempts.length === 0 ? <p className="text-sm text-muted-foreground">No node attempts recorded.</p> : (
                    <div className="overflow-x-auto"><table className="w-full text-left text-xs"><thead className="text-muted-foreground"><tr><th className="p-2">Node / identity</th><th className="p-2">Strategy</th><th className="p-2">Status</th><th className="p-2">Run / conversation</th><th className="p-2">Continuation / child</th><th className="p-2">Bound input</th></tr></thead><tbody>{trace.attempts.map((attempt) => <tr key={attempt.id} className="border-t border-border"><td className="p-2">{attempt.nodeId} #{attempt.ordinal}<span className="block text-muted-foreground">{attempt.profileIdentity || "—"}</span></td><td className="p-2">{attempt.strategy}</td><td className="p-2">{attempt.status}</td><td className="p-2 font-mono">{shortId(attempt.runId)} / {shortId(attempt.conversationId)}</td><td className="p-2 font-mono">{shortId(attempt.sourceAttemptId)} / {shortId(attempt.childExecutionId)}</td><td className="p-2 font-mono">{attempt.inputSnapshotSizeBytes.toString()} B · {shortId(attempt.inputSnapshotDigest.replace("sha256:", ""))}</td></tr>)}</tbody></table></div>
                  )}
                </section>

                <section aria-labelledby="journal-title">
                  <h3 id="journal-title" className="mb-2 text-sm font-semibold">Lifecycle journal</h3>
                  {trace?.journal.length ? <ol className="max-h-64 space-y-1 overflow-auto text-xs">{trace.journal.map((entry) => <li key={entry.id} className="grid grid-cols-[3rem_1fr_auto] gap-2 rounded bg-muted/30 p-2"><span>#{entry.sequence.toString()}</span><span>{entry.kind} · node {entry.nodeId || "—"} · attempt {shortId(entry.attemptId)}</span><span className="text-muted-foreground">{entry.payloadSizeBytes.toString()} B</span></li>)}</ol> : <p className="text-sm text-muted-foreground">No journal events recorded.</p>}
                </section>
              </article>
            ) : null}
          </div>
        ) : null}
      </div>
    </section>
  );
}
