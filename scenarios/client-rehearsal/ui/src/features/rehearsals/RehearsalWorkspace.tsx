import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { ExperienceSurface, type ExperienceSurfaceState } from "@vrooli/react-component-library/ExperienceSurface/1";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { rehearsalClient, type Rehearsal } from "../../api/rehearsals";
import { RehearsalState, ReviewDecision } from "@vrooli/proto-types/client-rehearsal/v1/rehearsals/rehearsals_pb";

const QUERY_KEY = ["rehearsals"] as const;
type TriggerOption = { id: string; name: string; kind: string };
const SYNTHETIC_OPENAPI = JSON.stringify({
  kind: "openapi-catalog",
  services: [
    {
      id: "crm",
      document: {
        openapi: "3.0.3",
        info: { title: "Synthetic CRM", version: "1" },
        servers: [{ url: "https://crm.synthetic.invalid" }],
        paths: {
          "/v1/contacts": {
            post: {
              responses: {
                "200": {
                  content: {
                    "application/json": {
                      example: { id: "contact_synthetic_001", status: "upserted" },
                    },
                  },
                },
              },
            },
          },
        },
      },
    },
    {
      id: "sheets",
      document: {
        openapi: "3.0.3",
        info: { title: "Synthetic Sheets", version: "1" },
        servers: [{ url: "https://sheets.synthetic.invalid" }],
        paths: {
          "/v1/spreadsheets/{sheetId}/rows": {
            post: {
              responses: {
                "200": {
                  content: {
                    "application/json": {
                      example: { updatedRange: "A2:B2", status: "appended" },
                    },
                  },
                },
              },
            },
          },
        },
      },
    },
  ],
});

function stateLabel(item: Rehearsal, t: ReturnType<typeof useTranslation>["t"]) {
  switch (item.state) {
    case RehearsalState.REHEARSAL_STATE_NEEDS_REVIEW: return t(strings.rehearsals.stateNeedsReview);
    case RehearsalState.REHEARSAL_STATE_APPROVED: return t(strings.rehearsals.stateApproved);
    case RehearsalState.REHEARSAL_STATE_CHANGES_REQUESTED: return t(strings.rehearsals.stateChangesRequested);
    case RehearsalState.REHEARSAL_STATE_FAILED: return t(strings.rehearsals.stateFailed);
    default: return t(strings.rehearsals.stateUnknown);
  }
}

function workflowTriggers(raw: string): TriggerOption[] {
  try {
    const document = JSON.parse(raw) as Record<string, unknown>;
    const outer = (document.workflow && typeof document.workflow === "object" ? document.workflow : document) as Record<string, unknown>;
    const workflows = Array.isArray(outer.workflows) ? outer.workflows as Array<Record<string, unknown>> : [];
    const entryId = typeof outer.entryWorkflowId === "string" ? outer.entryWorkflowId : "";
    const selected = workflows.find((item) => item.id === entryId) ?? workflows[0] ?? outer;
    const nodes = Array.isArray(selected.nodes) ? selected.nodes as Array<Record<string, unknown>> : [];
    return nodes.flatMap((node, index) => {
      const type = typeof node.type === "string" ? node.type : "";
      if (type === "n8n-nodes-base.errorTrigger" || type === "n8n-nodes-base.executeWorkflowTrigger") return [];
      let kind = "";
      if (type === "n8n-nodes-base.manualTrigger") kind = "manual";
      else if (type === "n8n-nodes-base.webhook") kind = "webhook";
      else if (type === "n8n-nodes-base.scheduleTrigger") kind = "scheduled";
      else if (type === "n8n-nodes-base.chatTrigger" || type.endsWith(".chatTrigger")) kind = "chat";
      else if (type.endsWith("Trigger") && type.startsWith("n8n-nodes-base.")) kind = "polling";
      if (!kind) return [];
      const id = typeof node.id === "string" && node.id.trim() ? node.id : `node-${index + 1}`;
      const name = typeof node.name === "string" && node.name.trim() ? node.name : id;
      return [{ id, name, kind }];
    });
  } catch {
    return [];
  }
}

export function RehearsalWorkspace() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [file, setFile] = useState<File | null>(null);
  const [triggerOptions, setTriggerOptions] = useState<TriggerOption[]>([]);
  const [selectedTriggerNodeId, setSelectedTriggerNodeId] = useState("");
  const [title, setTitle] = useState("");
  const [selectedId, setSelectedId] = useState("");
  const [simulatorSource, setSimulatorSource] = useState("");
  const rehearsals = useQuery({ queryKey: QUERY_KEY, queryFn: () => rehearsalClient.listRehearsals({ limit: 50 }) });
  const selected = useQuery({
    queryKey: [...QUERY_KEY, selectedId],
    queryFn: () => rehearsalClient.getRehearsal({ id: selectedId }),
    enabled: Boolean(selectedId),
  });
  const create = useMutation({
    mutationFn: async () => {
      if (!file) throw new Error(t(strings.rehearsals.fileRequired));
      const raw = await file.text();
      let source = simulatorSource.trim();
      if (source) {
        try { JSON.parse(source); } catch { throw new Error(t(strings.rehearsals.invalidSimulator)); }
      }
      else source = SYNTHETIC_OPENAPI;
      return rehearsalClient.createAndRunRehearsal({
        title: title.trim() || file.name.replace(/\.json$/i, ""),
        workflowJson: raw,
        triggerNodeId: selectedTriggerNodeId,
        simulatorProfile: "synthetic-default",
        simulatorSourceJson: source,
        injectSeededFault: true,
      });
    },
    onSuccess: (result) => {
      setSelectedId(result.rehearsal?.id ?? "");
      setFile(null);
      setTriggerOptions([]);
      setSelectedTriggerNodeId("");
      setTitle("");
      void queryClient.invalidateQueries({ queryKey: QUERY_KEY });
    },
  });
  const review = useMutation({
    mutationFn: (decision: ReviewDecision) => rehearsalClient.recordReview({ id: selectedId, decision, note: "Reviewed synthetic run evidence in the rehearsal workspace." }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEY });
      void queryClient.invalidateQueries({ queryKey: [...QUERY_KEY, selectedId] });
    },
  });
  const rows = rehearsals.data?.rehearsals ?? [];
  const current = selected.data?.rehearsal;
  const surfaceState: ExperienceSurfaceState = rehearsals.isLoading ? "loading" : rehearsals.error ? "error" : rows.length === 0 ? "empty" : "ready";
  const error = useMemo(() => create.error ? errorMessage(create.error, t) : rehearsals.error ? errorMessage(rehearsals.error, t) : selected.error ? errorMessage(selected.error, t) : "", [create.error, rehearsals.error, selected.error, t]);

  return (
    <ExperienceSurface surfaceId="rehearsals" state={surfaceState} statusMessage={surfaceState === "loading" ? t(strings.rehearsals.loading) : surfaceState === "error" ? error : undefined} data-testid={selectors.rehearsals.surface}>
      <div className="grid gap-space-md xl:grid-cols-[minmax(18rem,0.8fr)_minmax(0,1.5fr)]">
        <Card>
          <CardHeader><CardTitle as="h2">{t(strings.rehearsals.importTitle)}</CardTitle></CardHeader>
          <CardContent className="flex flex-col gap-space-sm">
            <p className="text-sm text-app-muted-foreground">{t(strings.rehearsals.syntheticNotice)}</p>
            <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="workflow-export">{t(strings.rehearsals.workflowFile)}
              <input id="workflow-export" data-testid={selectors.rehearsals.fileInput} type="file" accept="application/json,.json" onChange={async (event) => {
                const nextFile = event.currentTarget.files?.[0] ?? null;
                setFile(nextFile);
                const options = nextFile ? workflowTriggers(await nextFile.text()) : [];
                setTriggerOptions(options);
                setSelectedTriggerNodeId(options[0]?.id ?? "");
              }} className="rounded-control border border-app-border bg-app-background p-2 text-sm" />
            </label>
            {triggerOptions.length > 1 && <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="workflow-trigger">{t(strings.rehearsals.triggerLabel)}
              <select id="workflow-trigger" data-testid={selectors.rehearsals.triggerSelect} value={selectedTriggerNodeId} onChange={(event) => setSelectedTriggerNodeId(event.currentTarget.value)} className="rounded-control border border-app-border bg-app-background px-3 py-2">
                {triggerOptions.map((trigger) => <option key={trigger.id} value={trigger.id}>{trigger.name} · {trigger.kind}</option>)}
              </select>
            </label>}
            <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="workflow-title">{t(strings.rehearsals.titleLabel)}
              <input id="workflow-title" value={title} onChange={(event) => setTitle(event.currentTarget.value)} maxLength={120} className="rounded-control border border-app-border bg-app-background px-3 py-2" />
            </label>
            <details>
              <summary className="cursor-pointer text-sm text-app-muted-foreground">{t(strings.rehearsals.simulatorAdvanced)}</summary>
              <label className="mt-2 flex flex-col gap-1 text-sm font-medium" htmlFor="simulator-source">{t(strings.rehearsals.simulatorSource)}
                <textarea id="simulator-source" data-testid={selectors.rehearsals.simulatorSource} rows={5} value={simulatorSource} onChange={(event) => setSimulatorSource(event.currentTarget.value)} placeholder={t(strings.rehearsals.simulatorDefault)} className="w-full rounded-control border border-app-border bg-app-background p-2 font-mono text-xs" />
              </label>
            </details>
            <Button data-testid={selectors.rehearsals.runButton} onClick={() => create.mutate()} disabled={!file || create.isPending}>
              {create.isPending ? t(strings.rehearsals.running) : t(strings.rehearsals.run)}
            </Button>
            {error && <p role="alert" data-testid={selectors.rehearsals.error} className="text-sm text-app-danger">{error}</p>}
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle as="h2">{t(strings.rehearsals.historyTitle)}</CardTitle></CardHeader>
          <CardContent className="flex flex-col gap-space-sm">
            {rows.length === 0 && <div data-testid={selectors.rehearsals.empty}><EmptyState title={t(strings.rehearsals.emptyTitle)} description={t(strings.rehearsals.emptyDescription)} /></div>}
            {rows.map((item) => (
              <button key={item.id} type="button" data-testid={selectors.rehearsals.rehearsalRow} onClick={() => setSelectedId(item.id)} aria-pressed={selectedId === item.id} className="flex min-w-0 flex-col gap-1 rounded-panel border border-app-border p-3 text-left hover:border-app-accent focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2">
                <span className="break-words font-medium">{item.title}</span>
                <span className="text-xs text-app-muted-foreground">{stateLabel(item, t)} · {item.trace.length} {t(strings.rehearsals.events)}</span>
              </button>
            ))}
          </CardContent>
        </Card>

        {current && <Card className="xl:col-span-2" data-testid={selectors.rehearsals.detail}>
          <CardHeader className="flex flex-wrap items-center justify-between gap-2"><CardTitle as="h2">{current.title} <span className="text-sm font-normal text-app-muted-foreground">· {stateLabel(current, t)}</span></CardTitle>
            {current.state === RehearsalState.REHEARSAL_STATE_NEEDS_REVIEW && <div className="flex flex-wrap gap-2">
              <Button data-testid={selectors.rehearsals.approveButton} size="sm" disabled={review.isPending} onClick={() => review.mutate(ReviewDecision.APPROVE)}>{t(strings.rehearsals.approve)}</Button>
              <Button data-testid={selectors.rehearsals.changesButton} size="sm" disabled={review.isPending} onClick={() => review.mutate(ReviewDecision.REQUEST_CHANGES)}>{t(strings.rehearsals.requestChanges)}</Button>
            </div>}
          </CardHeader>
          <CardContent className="grid gap-space-md lg:grid-cols-2">
            <section aria-labelledby="trace-heading"><h3 id="trace-heading" className="mb-2 font-semibold">{t(strings.rehearsals.traceTitle)}</h3>
              <ol className="flex flex-col gap-2" data-testid={selectors.rehearsals.trace}>
                {current.trace.map((event) => <li key={`${event.sequence}-${event.operation}`} className="min-w-0 rounded-panel border border-app-border p-3">
                  <div className="flex flex-wrap justify-between gap-2"><strong className="break-all">{event.operation}</strong><span className={event.status === "succeeded" ? "text-app-success" : "text-app-danger"}>{event.status}</span></div>
                  <p className="mt-1 break-words text-sm text-app-muted-foreground">{event.responseSummary}</p>
                </li>)}
              </ol>
              <p className="mt-3 text-sm"><strong>{t(strings.rehearsals.faultLabel)}:</strong> {current.faultSummary}</p>
              <p className="mt-1 text-sm"><strong>{t(strings.rehearsals.repairLabel)}:</strong> {current.repairSummary}</p>
            </section>
            <section aria-labelledby="handoff-heading"><h3 id="handoff-heading" className="mb-2 font-semibold">{t(strings.rehearsals.handoffTitle)}</h3>
              <p className="mb-2 text-sm">{t(strings.rehearsals.credentialsReplaced, { count: current.credentialReplacements })}</p>
              <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-words rounded-panel border border-app-border bg-app-surface-muted p-3 text-xs">{current.handoffMarkdown}</pre>
              {current.reviewNote && <p className="mt-2 text-sm">{t(strings.rehearsals.reviewNote)} {current.reviewNote}</p>}
            </section>
          </CardContent>
        </Card>}
      </div>
    </ExperienceSurface>
  );
}
