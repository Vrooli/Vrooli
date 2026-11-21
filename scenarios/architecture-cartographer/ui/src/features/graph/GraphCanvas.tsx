import * as React from "react";
import { cva } from "class-variance-authority";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { cn } from "../../lib/utils";
import type { GraphLayout, GraphLayoutNode } from "./lib/graphAdapter";

/**
 * CVA variant ring for conflict overlay. The outer `<g>` for each node
 * carries one of these classes; severity-only nodes pick up a colored ring
 * plus a screen-reader label. Tokens are theme-driven; no raw colors.
 */
const nodeRingVariants = cva(
  "transition-colors",
  {
    variants: {
      severity: {
        none: "stroke-app-border",
        info: "stroke-app-info",
        low: "stroke-app-border",
        medium: "stroke-app-warning",
        high: "stroke-app-danger",
        critical: "stroke-app-danger",
      },
    },
    defaultVariants: { severity: "none" },
  },
);

const NODE_WIDTH = 160;
const NODE_HEIGHT = 40;
const PADDING = 24;

export interface GraphCanvasProps {
  layout: GraphLayout;
  scenario: string;
  className?: string;
}

export function GraphCanvas({ layout, scenario, className }: GraphCanvasProps) {
  const { t } = useTranslation();

  const width = React.useMemo(() => {
    if (layout.nodes.length === 0) return NODE_WIDTH + PADDING * 2;
    let max = 0;
    for (const node of layout.nodes) {
      if (node.x > max) max = node.x;
    }
    return max + NODE_WIDTH + PADDING * 2;
  }, [layout.nodes]);

  const height = React.useMemo(() => {
    if (layout.nodes.length === 0) return NODE_HEIGHT + PADDING * 2;
    let max = 0;
    for (const node of layout.nodes) {
      if (node.y > max) max = node.y;
    }
    return max + NODE_HEIGHT + PADDING * 2;
  }, [layout.nodes]);

  const nodeById = React.useMemo(() => {
    const map = new Map<string, GraphLayoutNode>();
    for (const node of layout.nodes) map.set(node.id, node);
    return map;
  }, [layout.nodes]);

  return (
    <div
      data-testid={selectors.features.graph.canvas.root}
      className={cn(
        "relative overflow-auto rounded-panel border border-app-border bg-app-surface p-2 backdrop-blur-sm",
        className,
      )}
    >
      <div
        data-testid={selectors.features.graph.canvas.summary}
        className="mb-2 flex flex-wrap gap-3 text-xs text-app-muted-foreground"
      >
        <span>{t(strings.pages.targetGraph.summaryNodes, { count: layout.nodes.length })}</span>
        <span>{t(strings.pages.targetGraph.summaryEdges, { count: layout.edges.length })}</span>
      </div>
      <svg
        aria-label={t(strings.features.graph.canvas.ariaLabel, { scenario })}
        viewBox={`0 0 ${width} ${height}`}
        width={width}
        height={height}
        className="block"
      >
        <g>
          {layout.edges.map((edge) => {
            const from = nodeById.get(edge.from);
            const to = nodeById.get(edge.to);
            if (from === undefined || to === undefined) return null;
            const x1 = from.x + PADDING + NODE_WIDTH;
            const y1 = from.y + PADDING + NODE_HEIGHT / 2;
            const x2 = to.x + PADDING;
            const y2 = to.y + PADDING + NODE_HEIGHT / 2;
            return (
              <line
                key={`${edge.from}->${edge.to}`}
                x1={x1}
                y1={y1}
                x2={x2}
                y2={y2}
                className="stroke-app-border"
                strokeWidth={1}
                aria-hidden="true"
              />
            );
          })}
        </g>
        <g>
          {layout.nodes.map((node) => {
            const severity = node.conflictSeverity ?? "none";
            const conflictLabel = node.conflictSeverity
              ? t(strings.features.graph.canvas.nodeConflictPresent, {
                severity: node.conflictSeverity,
              })
              : t(strings.features.graph.canvas.nodeConflictAbsent);
            const ariaLabel = t(strings.features.graph.canvas.nodeAriaLabel, {
              label: node.label,
              domain: node.domain.length > 0
                ? node.domain
                : t(strings.features.graph.accessibleList.noDomain),
              conflict: conflictLabel,
            });
            const x = node.x + PADDING;
            const y = node.y + PADDING;
            return (
              <g
                key={node.id}
                data-testid={selectors.features.graph.canvas.node({ id: node.id })}
                tabIndex={0}
                role="button"
                aria-label={ariaLabel}
                className="focus:outline-none focus-visible:[&>rect]:stroke-app-primary"
              >
                <rect
                  x={x}
                  y={y}
                  width={NODE_WIDTH}
                  height={NODE_HEIGHT}
                  rx={6}
                  className={cn(
                    "fill-app-surface-muted",
                    nodeRingVariants({ severity }),
                  )}
                  strokeWidth={node.conflictSeverity ? 2.5 : 1}
                />
                <text
                  x={x + 8}
                  y={y + NODE_HEIGHT / 2 + 4}
                  className="fill-app-foreground text-[11px] font-medium"
                  aria-hidden="true"
                >
                  {node.label.length > 22 ? `${node.label.slice(0, 21)}…` : node.label}
                </text>
              </g>
            );
          })}
        </g>
      </svg>
    </div>
  );
}
