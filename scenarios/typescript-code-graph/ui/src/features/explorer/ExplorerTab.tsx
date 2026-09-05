import * as React from "react";

import { AlertTriangle } from "lucide-react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { CodeGraph } from "../../api/graph";
import { GraphCanvas } from "./GraphCanvas";
import { GraphAccessibleList } from "./GraphAccessibleList";
import { GraphFilterBar } from "./GraphFilterBar";
import { GraphLegend } from "./GraphLegend";
import { FileSymbolPanel } from "./FileSymbolPanel";
import { buildGraphLayout, buildFileIndex } from "./lib/graphAdapter";

export interface ExplorerTabProps {
  graph: CodeGraph | undefined;
  target: string;
}

/**
 * Graph explorer tab. Owns the package filter selection and composes the SVG
 * canvas (primary visual surface), the always-present accessible list (WCAG
 * text alternative), the legend, the filter bar, the cycle banner, and the
 * file → symbol drill-down. All render models come from the pure adapter.
 */
export function ExplorerTab({ graph, target }: ExplorerTabProps) {
  const { t } = useTranslation();
  const [selected, setSelected] = React.useState<ReadonlySet<string>>(new Set());

  const layout = React.useMemo(() => buildGraphLayout(graph, selected), [graph, selected]);
  const files = React.useMemo(() => buildFileIndex(graph), [graph]);

  // Resolve each node's outgoing import targets to package labels for the
  // accessible list (so SR users get the same edges the canvas draws).
  const importLabels = React.useMemo(() => {
    const labelById = new Map<string, string>();
    for (const node of layout.nodes) labelById.set(node.id, node.path);
    const map = new Map<string, string[]>();
    for (const edge of layout.edges) {
      const list = map.get(edge.from) ?? [];
      const label = labelById.get(edge.to);
      if (label !== undefined) list.push(label);
      map.set(edge.from, list);
    }
    return map;
  }, [layout]);

  return (
    <div data-testid={selectors.features.explorer.root} className="flex flex-col gap-3">
      {layout.cycleCount > 0 ? (
        <div
          data-testid={selectors.features.explorer.cycleBanner}
          role="status"
          className="flex items-center gap-2 rounded-panel border border-app-danger/40 bg-app-danger/10 p-3 text-sm text-app-foreground"
        >
          <AlertTriangle aria-hidden="true" className="h-4 w-4 text-app-danger" />
          <span className="font-semibold text-app-danger">
            {t(strings.explorer.cycleBanner.title)}
          </span>
          <span className="text-app-muted-foreground">
            {t(strings.explorer.cycleBanner.message, { count: layout.cycleCount })}
          </span>
        </div>
      ) : null}

      <GraphFilterBar packages={layout.packages} selected={selected} onChange={setSelected} />

      <div className="grid gap-3 lg:grid-cols-[2fr_1fr]">
        <GraphCanvas layout={layout} target={target} />
        <div className="flex flex-col gap-3">
          <GraphLegend />
          <GraphAccessibleList layout={layout} importLabels={importLabels} />
        </div>
      </div>

      <FileSymbolPanel files={files} />
    </div>
  );
}
