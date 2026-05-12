import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useParams, useSearchParams } from "react-router-dom";

import {
  fetchFlowDetail,
  fetchRuns,
  type RunRow,
} from "../../api/inventory";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";
import { InspectorPanel } from "../../components/InspectorPanel";
import { TabList, TabPanel } from "../../components/ui/Tabs";

import { CounterexampleDiff } from "./CounterexampleDiff";
import { StateGraph } from "./StateGraph";
import { TracePlayer } from "./TracePlayer";

type TabKey = "graph" | "traces" | "history";

const FLOW_DETAIL_KEY = (scenarioId: string, flowId: string) =>
  ["flow-detail", scenarioId, flowId] as const;
const FLOW_RUNS_KEY = (flowId: string) => ["runs", "flow", flowId] as const;

/**
 * FlowDetailPage — the per-flow visual debugger.
 *
 * Reads `:flowId` from the route + an optional `?scenario=<id>` hint
 * (skips the cross-scenario lookup on the server). Tabs: Graph
 * (StateGraph), Traces (TracePlayer), History (run list +
 * CounterexampleDiff on failing-run selection).
 */
export function FlowDetailPage() {
  const { t } = useTranslation();
  const { flowId } = useParams<{ flowId: string }>();
  const [searchParams] = useSearchParams();
  const scenarioId = searchParams.get("scenario") ?? "";
  const [tab, setTab] = useState<TabKey>("graph");

  if (!flowId) {
    return (
      <section
        data-testid="flow-detail-missing"
        className="rounded-panel border border-app-border bg-app-surface p-4 text-app-danger"
      >
        {t("flowDetail.missingId", { defaultValue: "No flowId in route." })}
      </section>
    );
  }

  return (
    <FlowDetailBody
      flowId={flowId}
      scenarioId={scenarioId}
      tab={tab}
      onTabChange={setTab}
      t={t}
    />
  );
}

interface BodyProps {
  flowId: string;
  scenarioId: string;
  tab: TabKey;
  onTabChange: (tab: TabKey) => void;
  t: ReturnType<typeof useTranslation>["t"];
}

function FlowDetailBody({ flowId, scenarioId, tab, onTabChange, t }: BodyProps) {
  const detailQuery = useQuery({
    queryKey: FLOW_DETAIL_KEY(scenarioId, flowId),
    queryFn: () => fetchFlowDetail(flowId, { scenarioId: scenarioId || undefined }),
  });
  const [inspectedRun, setInspectedRun] = useState<RunRow | null>(null);

  if (detailQuery.isLoading) {
    return (
      <section
        data-testid="flow-detail-loading"
        className="rounded-panel border border-app-border bg-app-surface p-4 text-app-foreground"
      >
        {t("flowDetail.loading", { defaultValue: "Loading flow…" })}
      </section>
    );
  }

  if (detailQuery.error || !detailQuery.data) {
    return (
      <section
        data-testid="flow-detail-error"
        className="rounded-panel border border-app-border bg-app-surface p-4 text-app-danger"
      >
        {errorMessage(detailQuery.error, t)}
        <div className="mt-3">
          <Link
            data-testid="flow-detail-back"
            to="/flows"
            className="text-sm text-app-primary underline"
          >
            {t("flowDetail.back", { defaultValue: "Back to inventory" })}
          </Link>
        </div>
      </section>
    );
  }

  const detail = detailQuery.data;

  return (
    <div data-testid="flow-detail-page" className="flex min-h-0 gap-4">
      <section
        aria-label={t("flowDetail.title", { defaultValue: "Flow detail" })}
        className="min-w-0 flex-1 rounded-panel border border-app-border bg-app-surface p-4"
      >
        <header className="flex flex-wrap items-baseline justify-between gap-3">
        <div>
          <h2 className="text-sm font-medium text-app-muted-foreground">
            {t("flowDetail.title", { defaultValue: "Flow detail" })}
          </h2>
          <p data-testid="flow-detail-id" className="mt-1 font-mono text-base text-app-foreground">
            {detail.flowId}
          </p>
          <p className="mt-1 text-xs text-app-muted-foreground">
            <span data-testid="flow-detail-lang">{detail.language}</span>
            {" · "}
            <span>{t("flowDetail.statesCount", { defaultValue: `${detail.states.length} states` })}</span>
            {" · "}
            <span>{t("flowDetail.eventsCount", { defaultValue: `${detail.events.length} events` })}</span>
          </p>
        </div>
        <Link
          data-testid="flow-detail-back"
          to="/flows"
          className="text-sm text-app-primary underline"
        >
          {t("flowDetail.back", { defaultValue: "Back to inventory" })}
        </Link>
      </header>

      <TabList
        idPrefix="flow-detail"
        value={tab}
        onChange={onTabChange}
        aria-label={t("flowDetail.tabsAria", { defaultValue: "Flow detail sections" })}
        className="mt-4"
        items={[
          { value: "graph", label: tabLabel("graph", t) },
          { value: "traces", label: tabLabel("traces", t) },
          { value: "history", label: tabLabel("history", t) },
        ]}
      />

      <div className="mt-4">
        <TabPanel idPrefix="flow-detail" value="graph" active={tab}>
          <StateGraph
            states={detail.states}
            events={detail.events}
            transitions={detail.transitions}
            initialState={detail.initialState}
          />
        </TabPanel>
        <TabPanel idPrefix="flow-detail" value="traces" active={tab}>
          <TracePlayer
            traces={detail.traces}
            graphProps={{
              states: detail.states,
              events: detail.events,
              transitions: detail.transitions,
              initialState: detail.initialState,
            }}
          />
        </TabPanel>
        <TabPanel idPrefix="flow-detail" value="history" active={tab}>
          <HistoryTab
            flowId={detail.flowId}
            inspectedRunId={inspectedRun?.id}
            onInspect={setInspectedRun}
            t={t}
          />
        </TabPanel>
      </div>
      </section>

      <InspectorPanel
        open={inspectedRun !== null}
        onClose={() => setInspectedRun(null)}
        title={
          inspectedRun
            ? t("flowDetail.inspectorTitle", {
                defaultValue: `Counterexample · ${inspectedRun.id.slice(0, 8)}`,
              })
            : ""
        }
      >
        {inspectedRun && (
          <CounterexampleDiff
            counterexampleJson={inspectedRun.counterexample ?? ""}
            expectedTransitions={detail.transitions}
          />
        )}
      </InspectorPanel>
    </div>
  );
}

function tabLabel(key: TabKey, t: BodyProps["t"]): string {
  switch (key) {
    case "graph":
      return t("flowDetail.tabGraph", { defaultValue: "Graph" });
    case "traces":
      return t("flowDetail.tabTraces", { defaultValue: "Traces" });
    case "history":
      return t("flowDetail.tabHistory", { defaultValue: "History" });
  }
}

interface HistoryTabProps {
  flowId: string;
  inspectedRunId: string | undefined;
  onInspect: (run: RunRow) => void;
  t: BodyProps["t"];
}

function HistoryTab({ flowId, inspectedRunId, onInspect, t }: HistoryTabProps) {
  const runsQuery = useQuery({
    queryKey: FLOW_RUNS_KEY(flowId),
    queryFn: () => fetchRuns({ flowId, limit: 50 }),
  });

  const runs = useMemo(() => runsQuery.data ?? [], [runsQuery.data]);

  if (runsQuery.isLoading) {
    return (
      <p data-testid="flow-history-loading" className="text-app-foreground">
        {t("flowDetail.historyLoading", { defaultValue: "Loading run history…" })}
      </p>
    );
  }

  if (runs.length === 0) {
    return (
      <p data-testid="flow-history-empty" className="text-app-foreground">
        {t("flowDetail.historyEmpty", {
          defaultValue: "No verification runs yet for this flow.",
        })}
      </p>
    );
  }

  return (
    <div data-testid="flow-history" className="space-y-4">
      <table className="w-full text-left text-sm text-app-foreground">
        <thead className="text-xs uppercase text-app-muted-foreground">
          <tr>
            <th className="py-1 pr-3">{t("flowDetail.colWhen", { defaultValue: "When" })}</th>
            <th className="py-1 pr-3">{t("flowDetail.colStatus", { defaultValue: "Status" })}</th>
            <th className="py-1 pr-3">{t("flowDetail.colDuration", { defaultValue: "Duration" })}</th>
            <th className="py-1 pr-3">{t("flowDetail.colRun", { defaultValue: "Run" })}</th>
            <th className="py-1 pr-3" />
          </tr>
        </thead>
        <tbody>
          {runs.map((run) => (
            <tr
              key={run.id}
              data-testid={`flow-history-row-${run.id}`}
              className={`border-t border-app-border ${run.id === inspectedRunId ? "bg-app-surface-muted" : ""}`}
            >
              <td className="py-1 pr-3 text-xs text-app-muted-foreground">
                {new Date(run.finishedAt).toLocaleString()}
              </td>
              <td className="py-1 pr-3">{run.status}</td>
              <td className="py-1 pr-3 text-xs text-app-muted-foreground">
                {t("flowDetail.historyDurationMs", {
                  defaultValue: "{{ms}} ms",
                  ms: run.durationMs,
                })}
              </td>
              <td className="py-1 pr-3">
                <Link
                  data-testid={`flow-history-link-${run.id}`}
                  to={`/runs/${run.id}`}
                  className="text-app-primary underline"
                >
                  {run.id.slice(0, 8)}
                </Link>
              </td>
              <td className="py-1 pr-3 text-right">
                {run.status !== "passed" && (
                  <button
                    type="button"
                    data-testid={`flow-history-inspect-${run.id}`}
                    onClick={() => onInspect(run)}
                    className="text-xs text-app-primary underline"
                  >
                    {t("flowDetail.inspect", { defaultValue: "Inspect" })}
                  </button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
