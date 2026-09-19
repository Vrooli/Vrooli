import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Download, RefreshCw } from "lucide-react";
import { NodeKind } from "@vrooli/proto-types/tech-tree-designer/v1/graph/graph_pb";

import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { describeTechTree, ExportFormat, exportTechTree, type GroupBy } from "../../api/techTree";
import { GraphCanvas } from "./GraphCanvas";

const groupOptions = [
  { value: "none", labelKey: strings.graph.group.none },
  { value: "sector", labelKey: strings.graph.group.sector },
  { value: "tier", labelKey: strings.graph.group.tier },
] as const satisfies ReadonlyArray<{ value: GroupBy; labelKey: string }>;

export function GraphPage() {
  const { t } = useTranslation();
  const [groupBy, setGroupBy] = useState<GroupBy>("sector");
  const graphQuery = useQuery({
    queryKey: ["tech-tree-graph", groupBy],
    queryFn: () => describeTechTree(groupBy),
  });

  const graph = graphQuery.data?.graph;
  const plannedCount = graph?.nodes.filter((node) => node.kind === NodeKind.PLANNED).length ?? 0;
  const liveCount = (graph?.nodes.length ?? 0) - plannedCount;

  const handleExport = async () => {
    const response = await exportTechTree(ExportFormat.DOT);
    const blob = new Blob([response.content], { type: response.mediaType || "text/vnd.graphviz" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "tech-tree.dot";
    link.click();
    URL.revokeObjectURL(url);
  };

  return (
    <section data-testid={selectors.pages.graph} className="flex flex-col gap-5">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p className="text-sm font-medium uppercase text-app-muted-foreground">{t(strings.graph.eyebrow)}</p>
          <h2 className="text-3xl font-semibold">{t(strings.graph.title)}</h2>
          <p className="mt-2 max-w-3xl text-sm text-app-muted-foreground">
            {t(strings.graph.description)}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <select
            data-testid={selectors.graph.groupBy}
            value={groupBy}
            onChange={(event) => setGroupBy(event.target.value as GroupBy)}
            className="h-10 rounded-md border border-app-border bg-app-surface px-3 text-sm"
          >
            {groupOptions.map((option) => (
              <option key={option.value} value={option.value}>{t(option.labelKey)}</option>
            ))}
          </select>
          <Button variant="outline" onClick={() => void graphQuery.refetch()}>
            <RefreshCw aria-hidden className="mr-2 h-4 w-4" />
            {t(strings.graph.actions.refresh)}
          </Button>
          <Button variant="outline" onClick={() => void handleExport()}>
            <Download aria-hidden className="mr-2 h-4 w-4" />
            {t(strings.graph.actions.dot)}
          </Button>
        </div>
      </div>

      <div className="grid gap-3 md:grid-cols-4">
        <Metric label={t(strings.graph.metrics.live)} value={liveCount} />
        <Metric label={t(strings.graph.metrics.planned)} value={plannedCount} />
        <Metric label={t(strings.graph.metrics.dependencies)} value={graph?.edges.length ?? 0} />
        <Metric label={t(strings.graph.metrics.warnings)} value={graph?.errors.length ?? 0} />
      </div>

      {graphQuery.isLoading && <StatePanel label={t(strings.graph.states.loading)} />}
      {graphQuery.error && <StatePanel label={t(strings.graph.states.error)} tone="error" />}
      {graph && <GraphCanvas graph={graph} />}
      {graph?.errors.length ? (
        <div className="rounded-lg border border-amber-500/35 bg-amber-950/25 p-4 text-sm">
          <p className="font-medium text-amber-100">{t(strings.graph.warnings.title)}</p>
          <ul className="mt-2 space-y-1 text-amber-100/80">
            {graph.errors.map((error) => (
              <li key={`${error.source}-${error.scenario}-${error.message}`}>
                {t(strings.graph.warnings.item, {
                  source: error.source,
                  scenario: error.scenario ? `(${error.scenario})` : "",
                  message: error.message,
                })}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </section>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border border-app-border bg-app-surface p-4">
      <p className="text-xs uppercase text-app-muted-foreground">{label}</p>
      <p className="mt-2 text-2xl font-semibold">{value}</p>
    </div>
  );
}

function StatePanel({ label, tone = "default" }: { label: string; tone?: "default" | "error" }) {
  return (
    <div className={[
      "rounded-lg border p-5 text-sm",
      tone === "error" ? "border-red-500/40 bg-red-950/25 text-red-100" : "border-app-border bg-app-surface text-app-muted-foreground",
    ].join(" ")}>
      {label}
    </div>
  );
}
