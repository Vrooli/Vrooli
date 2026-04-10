/**
 * DetailActionButtons
 *
 * Renders action buttons from the action registry for a given entity type.
 * Adapts the ActionButton pattern from Inspector.tsx for use in detail pages.
 *
 * Navigation actions call the detail selection store directly instead of
 * using route-based navigation.
 */

import { useCallback, useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import { cn } from "../../lib/utils";
import { useGraphDataStore } from "../../surfaces/graph/stores/graph-data-store";
import type { EntityType } from "../../surfaces/graph/stores/graph-settings-store";
import { getActionsForNode, type InspectorAction } from "../../surfaces/graph/lib/action-registry";
import type { GraphNode } from "../../surfaces/graph/types";
import { useDetailSelectionStore } from "../../stores/detail-selection-store";
import type { DetailSelection } from "../../stores/detail-selection-store";

interface DetailActionButtonsProps {
  entityType: EntityType;
  /** The graph node, if available (for handler/enabled checks). */
  node?: GraphNode | null;
  /** Layout direction for buttons. */
  direction?: "row" | "column";
  className?: string;
}

function ActionButton({
  action,
  node,
  onNavigate,
}: {
  action: InspectorAction;
  node: GraphNode | null;
  onNavigate: (selection: DetailSelection) => void;
}) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isEnabled = node && action.enabled ? action.enabled(node) : true;
  const Icon = action.icon;

  useEffect(() => {
    if (!error) return;
    const timer = window.setTimeout(() => setError(null), 5000);
    return () => window.clearTimeout(timer);
  }, [error]);

  const handleClick = useCallback(async () => {
    if (!node) return;

    // Navigation actions return a DetailSelection.
    if (action.navigateTo) {
      const target = action.navigateTo(node);
      if (target) onNavigate(target);
      return;
    }

    setLoading(true);
    setError(null);
    try {
      await action.handler(node);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Action failed");
    } finally {
      setLoading(false);
    }
  }, [action, node, onNavigate]);

  return (
    <div>
      <button
        type="button"
        onClick={() => void handleClick()}
        disabled={!isEnabled || loading}
        className={cn(
          "flex items-center justify-center gap-1.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
          action.variant === "destructive"
            ? "bg-red-500/15 text-red-300 hover:bg-red-500/25 disabled:bg-red-500/5 disabled:text-red-400/50"
            : "bg-cyan-600/20 text-cyan-300 hover:bg-cyan-600/30 disabled:bg-cyan-600/5 disabled:text-cyan-400/50",
        )}
        data-testid={`detail-action-${action.id}`}
      >
        {loading ? (
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
        ) : (
          <Icon className="h-3.5 w-3.5" />
        )}
        {action.label}
      </button>
      {error && (
        <p className="mt-1 text-xs text-red-400" data-testid={`detail-action-error-${action.id}`}>
          {error}
        </p>
      )}
    </div>
  );
}

export function DetailActionButtons({
  entityType,
  node,
  direction = "row",
  className,
}: DetailActionButtonsProps) {
  const lens = useGraphDataStore((s) => s.lens);
  const actions = getActionsForNode(lens, entityType);
  const store = useDetailSelectionStore();

  const handleNavigate = useCallback(
    (selection: DetailSelection) => {
      switch (selection.entityType) {
        case "backlog":
          if (selection.kind && selection.name) {
            store.selectBacklog(selection.kind, selection.name, selection.tab);
          }
          break;
        case "scenario":
          if (selection.name) store.selectScenario(selection.name, selection.tab);
          break;
        case "execution":
          if (selection.identifier) store.selectExecution(selection.identifier);
          break;
        case "initiative":
          if (selection.name) store.selectInitiative(selection.name, selection.tab);
          break;
      }
    },
    [store],
  );

  // Filter out "View Details" actions since we're already on the detail page.
  const filteredActions = actions.filter((a) =>
    a.id !== "view-backlog-details"
    && a.id !== "view-scenario-details"
    && a.id !== "view-execution-details"
    && a.id !== "edit-backlog"
    && a.id !== "edit-initiative"
    && a.id !== "edit-scenario",
  );

  if (filteredActions.length === 0) return null;

  return (
    <div
      className={cn(
        "flex gap-2",
        direction === "column" ? "flex-col" : "flex-row flex-wrap",
        className,
      )}
      data-testid="detail-action-buttons"
    >
      {filteredActions.map((action) => (
        <ActionButton
          key={action.id}
          action={action}
          node={node ?? null}
          onNavigate={handleNavigate}
        />
      ))}
    </div>
  );
}
