/**
 * NavigationGraph — renders a navigation flow's routes/affordances as a
 * directed graph alongside context toggles (viewport, auth, ...) and
 * reachability-invariant pass/fail badges.
 *
 * The renderer is fed a typed descriptor returned by
 * GetNavigationStudio. Filtering for the active context happens in the
 * browser via the shared predicate evaluator so the UI stays responsive
 * to toggle changes without round-tripping to the server.
 */
import { useMemo, useState } from "react";
import { Background, Position, ReactFlow, type Edge, type Node } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import dagre from "dagre";

import { useTranslation } from "../../i18n";
import { evaluatePredicate } from "../../lib/navigationPredicate";
import type {
  NavigationStudioAffordance,
  NavigationStudioContainer,
  NavigationStudioContext,
  NavigationStudioDescriptor,
  NavigationStudioInvariant,
  NavigationStudioRoute,
} from "../../api/inventory";

const NODE_WIDTH = 200;
const NODE_HEIGHT = 56;

export interface NavigationGraphProps {
  descriptor: NavigationStudioDescriptor;
}

interface LayoutResult {
  nodes: Node[];
  edges: Edge[];
}

function layoutGraph(
  routes: NavigationStudioRoute[],
  affordances: NavigationStudioAffordance[],
  active: Record<string, string>,
): LayoutResult {
  const lookup = (name: string) => active[name];
  const reachableRoutes = new Set(
    routes.filter((r) => evaluatePredicate(r.requires, lookup)).map((r) => r.id),
  );

  const g = new dagre.graphlib.Graph();
  g.setGraph({ rankdir: "LR", nodesep: 32, ranksep: 64 });
  g.setDefaultEdgeLabel(() => ({}));

  for (const r of routes) {
    g.setNode(r.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
  }
  // Parent links render as light structural edges so nested routes have visible context.
  const parentEdges: Array<{ from: string; to: string }> = [];
  for (const r of routes) {
    for (const p of r.parents) {
      parentEdges.push({ from: p, to: r.id });
      g.setEdge(p, r.id);
    }
  }

  // Affordance edges: collapse multiple presentations of the same affordance to
  // one logical edge per (from-route, to-route) pair. The "from" route of an
  // affordance is any host route of its container (we pick the first that's
  // reachable under the active context); if the presentation site is itself a
  // route, that's the from. We render one edge per resolved (from→to) pair.
  type Pair = { from: string; to: string; labels: string[]; invisible: boolean };
  const pairs = new Map<string, Pair>();
  for (const a of affordances) {
    const visible = evaluatePredicate(a.showWhen, lookup);
    for (const p of a.presentations) {
      const key = `${p.in}->${a.to}`;
      const existing = pairs.get(key);
      const label = p.label;
      if (existing) existing.labels.push(label);
      else
        pairs.set(key, {
          from: p.in,
          to: a.to,
          labels: [label],
          invisible: !visible,
        });
    }
  }
  for (const p of pairs.values()) {
    if (g.node(p.from) && g.node(p.to)) {
      g.setEdge(p.from, p.to);
    }
  }

  dagre.layout(g);

  const nodes: Node[] = routes.map((r) => {
    const node = g.node(r.id) as { x: number; y: number } | undefined;
    const pos = node ?? { x: 0, y: 0 };
    const reachable = reachableRoutes.has(r.id);
    return {
      id: r.id,
      data: {
        label: (
          <div style={{ fontSize: 12, lineHeight: 1.2 }}>
            <div style={{ fontWeight: 600 }}>{r.id}</div>
            <div style={{ fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace", fontSize: 11, opacity: 0.7 }}>
              {r.path}
            </div>
          </div>
        ),
      },
      position: { x: pos.x - NODE_WIDTH / 2, y: pos.y - NODE_HEIGHT / 2 },
      style: {
        width: NODE_WIDTH,
        height: NODE_HEIGHT,
        borderRadius: 8,
        border: reachable ? "1px solid var(--color-border)" : "1px dashed var(--color-warning)",
        background: reachable ? "var(--color-surface)" : "color-mix(in srgb, var(--color-warning) 12%, transparent)",
        color: "var(--color-foreground)",
        padding: "6px 8px",
        opacity: reachable ? 1 : 0.65,
      },
      sourcePosition: Position.Right,
      targetPosition: Position.Left,
    };
  });

  const edges: Edge[] = [];
  for (const pe of parentEdges) {
    edges.push({
      id: `parent-${pe.from}-${pe.to}`,
      source: pe.from,
      target: pe.to,
      style: { stroke: "var(--color-muted-foreground)", strokeDasharray: "2 4", opacity: 0.4 },
    });
  }
  for (const pair of pairs.values()) {
    edges.push({
      id: `aff-${pair.from}-${pair.to}`,
      source: pair.from,
      target: pair.to,
      label: pair.labels.join(", "),
      labelStyle: { fill: "var(--color-muted-foreground)", fontSize: 11 },
      labelBgStyle: { fill: "var(--color-surface)" },
      style: {
        stroke: "var(--color-muted-foreground)",
        opacity: pair.invisible ? 0.25 : 1,
        strokeDasharray: pair.invisible ? "4 4" : undefined,
      },
    });
  }
  return { nodes, edges };
}

function initialContextState(contexts: NavigationStudioContext[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const c of contexts) {
    out[c.name] = c.defaultValue;
  }
  return out;
}

export function NavigationGraph({ descriptor }: NavigationGraphProps) {
  const { t } = useTranslation();
  const [active, setActive] = useState<Record<string, string>>(() =>
    initialContextState(descriptor.contexts),
  );

  const { nodes, edges } = useMemo(
    () => layoutGraph(descriptor.routes, descriptor.affordances, active),
    [descriptor.routes, descriptor.affordances, active],
  );

  const visibleContainers = useMemo(
    () =>
      descriptor.containers.filter((c) =>
        evaluatePredicate(c.showWhen, (name) => active[name]),
      ),
    [descriptor.containers, active],
  );

  if (descriptor.routes.length === 0) {
    return (
      <p
        data-testid="navigation-graph-empty"
        className="rounded-panel border border-app-border bg-app-surface p-4 text-app-foreground"
      >
        {t("navigationGraph.empty", { defaultValue: "No routes declared." })}
      </p>
    );
  }

  return (
    <div data-testid="navigation-graph" className="space-y-3">
      <ContextToggles contexts={descriptor.contexts} active={active} onChange={setActive} />
      <ContainersStrip containers={visibleContainers} />
      <div
        role="img"
        aria-label={t("navigationGraph.aria", { defaultValue: "Navigation graph" })}
        className="rounded-panel border border-app-border bg-app-surface"
        style={{ height: 420 }}
      >
        <ReactFlow
          nodes={nodes}
          edges={edges}
          fitView
          nodesDraggable={false}
          nodesConnectable={false}
          edgesFocusable={false}
          proOptions={{ hideAttribution: true }}
        >
          <Background gap={16} />
        </ReactFlow>
      </div>
      <InvariantsList invariants={descriptor.invariants} />
      <p data-testid="navigation-graph-summary" className="px-3 py-1 text-xs text-app-muted-foreground">
        {t("navigationGraph.summary", {
          defaultValue: `${descriptor.routes.length} routes · ${descriptor.affordances.length} affordances · ${visibleContainers.length}/${descriptor.containers.length} containers visible`,
        })}
      </p>
    </div>
  );
}

interface TogglesProps {
  contexts: NavigationStudioContext[];
  active: Record<string, string>;
  onChange: (next: Record<string, string>) => void;
}

function ContextToggles({ contexts, active, onChange }: TogglesProps) {
  if (contexts.length === 0) return null;
  return (
    <div
      data-testid="navigation-graph-toggles"
      className="flex flex-wrap gap-3 rounded-panel border border-app-border bg-app-surface p-3"
    >
      {contexts.map((c) => (
        <label key={c.name} className="flex items-center gap-2 text-xs text-app-foreground">
          <span className="font-medium uppercase tracking-wide text-app-muted-foreground">{c.name}</span>
          {c.kind === "bool" ? (
            <input
              type="checkbox"
              data-testid={`nav-toggle-${c.name}`}
              checked={active[c.name] === "true"}
              onChange={(e) => onChange({ ...active, [c.name]: e.target.checked ? "true" : "false" })}
            />
          ) : (
            <select
              data-testid={`nav-toggle-${c.name}`}
              value={active[c.name] ?? c.defaultValue}
              onChange={(e) => onChange({ ...active, [c.name]: e.target.value })}
              className="rounded border border-app-border bg-app-surface px-2 py-1 text-xs text-app-foreground"
            >
              {c.values.map((v) => (
                <option key={v} value={v}>
                  {v}
                </option>
              ))}
            </select>
          )}
        </label>
      ))}
    </div>
  );
}

function ContainersStrip({ containers }: { containers: NavigationStudioContainer[] }) {
  if (containers.length === 0) {
    return (
      <p
        data-testid="navigation-graph-containers-empty"
        className="rounded-panel border border-dashed border-app-border bg-app-surface p-2 text-xs text-app-muted-foreground"
      >
        No containers visible under the current context.
      </p>
    );
  }
  return (
    <ul
      data-testid="navigation-graph-containers"
      className="flex flex-wrap gap-2 rounded-panel border border-app-border bg-app-surface p-2"
    >
      {containers.map((c) => (
        <li
          key={c.id}
          data-testid={`nav-container-${c.id}`}
          className="rounded border border-app-border bg-app-surface-muted px-2 py-1 text-xs text-app-foreground"
        >
          <span className="font-medium">{c.id}</span>
          <span className="ml-1 text-app-muted-foreground">({c.kind})</span>
        </li>
      ))}
    </ul>
  );
}

function InvariantsList({ invariants }: { invariants: NavigationStudioInvariant[] }) {
  if (invariants.length === 0) return null;
  return (
    <ul
      data-testid="navigation-graph-invariants"
      className="space-y-1 rounded-panel border border-app-border bg-app-surface p-2 text-xs"
    >
      {invariants.map((inv) => (
        <li
          key={inv.id}
          data-testid={`nav-invariant-${inv.id}`}
          className="flex items-baseline gap-2"
          data-passed={inv.passed ? "true" : "false"}
        >
          <span
            className={
              inv.passed
                ? "rounded bg-app-success/15 px-1.5 py-0.5 text-app-success"
                : "rounded bg-app-danger/15 px-1.5 py-0.5 text-app-danger"
            }
          >
            {inv.passed ? "PASS" : "FAIL"}
          </span>
          <span className="font-mono text-app-foreground">{inv.id}</span>
          <span className="text-app-muted-foreground">{inv.message}</span>
        </li>
      ))}
    </ul>
  );
}
