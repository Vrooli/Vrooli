/**
 * Inspector - Floating panel that opens on node selection.
 *
 * Desktop: floating popover anchored near node, clamped to viewport.
 * Mobile: bottom sheet.
 *
 * Renders lens-specific action buttons from the action registry.
 */

import { useCallback, useEffect, useState } from "react";
import { ExternalLink, Loader2, X } from "lucide-react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useIsMobile } from "../../../hooks/useMediaQuery";
import { BottomSheet } from "../../../components/ui/bottom-sheet";
import { cn } from "../../../lib/utils";
import { useGraphDataStore } from "../stores/graph-data-store";
import type { EntityType } from "../stores/graph-data-store";
import { getActionsForNode, type InspectorAction } from "../lib/action-registry";
import { parseNodeId } from "../lib/node-id-parser";
import { getGraphNodeData, getGraphNodeLabel, type GraphNode } from "../types";

interface InspectorProps {
  isOpen: boolean;
  onClose: () => void;
  selectedNode: GraphNode | null;
}

/**
 * Build a details-page path for nodes that support drill-down.
 * Returns null for entity types with no detail page.
 */
function getDetailsPath(node: GraphNode): string | null {
  const parsed = parseNodeId(node.id);
  if (!parsed) {
    return null;
  }

  if (parsed.entityType === "backlog" && parsed.kind && parsed.name) {
    return `/details/backlog/${parsed.kind}/${parsed.name}`;
  }
  if (parsed.entityType === "scenario" && parsed.name) {
    return `/details/scenario/${parsed.name}`;
  }
  if (parsed.entityType === "execution") {
    return `/details/execution/${parsed.identifier}`;
  }
  return null;
}

function ActionButton({
  action,
  node,
  onNavigate,
}: {
  action: InspectorAction;
  node: GraphNode;
  onNavigate: (path: string) => void;
}) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isEnabled = action.enabled ? action.enabled(node) : true;
  const Icon = action.icon;

  // Auto-dismiss error after 5 seconds.
  useEffect(() => {
    if (!error) return;
    const timer = window.setTimeout(() => setError(null), 5000);
    return () => window.clearTimeout(timer);
  }, [error]);

  const handleClick = useCallback(async () => {
    // Navigation actions.
    if (action.navigateTo) {
      const path = action.navigateTo(node);
      if (path) onNavigate(path);
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
          "flex w-full items-center justify-center gap-1.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
          action.variant === "destructive"
            ? "bg-red-500/15 text-red-300 hover:bg-red-500/25 disabled:bg-red-500/5 disabled:text-red-400/50"
            : "bg-cyan-600/20 text-cyan-300 hover:bg-cyan-600/30 disabled:bg-cyan-600/5 disabled:text-cyan-400/50",
        )}
        data-testid={`inspector-action-${action.id}`}
      >
        {loading ? (
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
        ) : (
          <Icon className="h-3.5 w-3.5" />
        )}
        {action.label}
      </button>
      {error && (
        <p className="mt-1 text-xs text-red-400" data-testid={`inspector-action-error-${action.id}`}>
          {error}
        </p>
      )}
    </div>
  );
}

function InspectorContent({ node }: { node: GraphNode }) {
  const data = getGraphNodeData(node);
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const lens = useGraphDataStore((s) => s.lens);
  const entityType = (data.entityType as EntityType) ?? "backlog";
  const actions = getActionsForNode(lens, entityType);
  const detailsPath = getDetailsPath(node);

  const handleNavigate = useCallback(
    (path: string) => {
      const returnTo = `/graph?${searchParams.toString()}`;
      navigate(`${path}?returnTo=${encodeURIComponent(returnTo)}`);
    },
    [navigate, searchParams],
  );

  return (
    <div className="space-y-3" data-testid="inspector-content">
      <div>
        <span className="text-xs font-medium uppercase tracking-wider text-slate-500">
          {data.entityType ?? "Unknown"}
        </span>
        <h3 className="mt-0.5 text-base font-semibold text-slate-100">
          {data.label ?? node.id}
        </h3>
      </div>
      <div className="flex items-center gap-2">
        <span className="rounded-full bg-slate-700/60 px-2.5 py-0.5 text-xs text-slate-300">
          {data.status ?? "unknown"}
        </span>
        {"kind" in data && typeof data.kind === "string" && (
          <span className="rounded-full bg-cyan-500/15 px-2.5 py-0.5 text-xs text-cyan-300">
            {data.kind}
          </span>
        )}
      </div>

      {/* Action buttons from registry */}
      {actions.length > 0 && (
        <div className="space-y-2" data-testid="inspector-actions">
          {actions.map((action) => (
            <ActionButton
              key={action.id}
              action={action}
              node={node}
              onNavigate={handleNavigate}
            />
          ))}
        </div>
      )}

      {/* Fallback "View Details" for lenses without registered actions */}
      {actions.length === 0 && detailsPath && (
        <button
          type="button"
          onClick={() => handleNavigate(detailsPath)}
          className="flex w-full items-center justify-center gap-1.5 rounded-lg bg-cyan-600/20 px-3 py-2 text-sm font-medium text-cyan-300 transition-colors hover:bg-cyan-600/30"
          data-testid="inspector-view-details"
        >
          <ExternalLink className="h-3.5 w-3.5" />
          View Details
        </button>
      )}

      <p className="text-xs text-slate-400">
        Node ID: <code className="text-slate-300">{node.id}</code>
      </p>
    </div>
  );
}

export function Inspector({ isOpen, onClose, selectedNode }: InspectorProps) {
  const isMobile = useIsMobile();

  if (!isOpen || !selectedNode) return null;

  const title = getGraphNodeLabel(selectedNode);

  // Mobile: use existing BottomSheet.
  if (isMobile) {
    return (
      <BottomSheet
        isOpen={isOpen}
        onClose={onClose}
        title={title}
        data-testid="inspector"
      >
        <InspectorContent node={selectedNode} />
      </BottomSheet>
    );
  }

  // Desktop: floating popover.
  return (
    <div
      className={cn(
        "fixed right-4 top-20 z-30 w-80",
        "rounded-xl border border-slate-700/80 bg-slate-900/95 shadow-2xl backdrop-blur-sm",
      )}
      role="dialog"
      aria-label="Inspector"
      data-testid="inspector"
    >
      <div className="flex items-center justify-between border-b border-slate-700/50 px-4 py-2.5">
        <h2 className="text-sm font-semibold text-slate-100 truncate">{title}</h2>
        <button
          type="button"
          onClick={onClose}
          className="rounded p-1 text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition-colors"
          aria-label="Close inspector"
          data-testid="inspector-close"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
      <div className="max-h-[70vh] overflow-y-auto p-4">
        <InspectorContent node={selectedNode} />
      </div>
    </div>
  );
}
