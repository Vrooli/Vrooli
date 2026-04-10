// DOC: docs/concepts/ARCHITECTURE.md#ui-layer
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Trash2, Link } from "lucide-react";
import { listThoughts, createThought, deleteThought, listEdges, createEdge, deleteEdge } from "../lib/api";
import { THOUGHT_PLACEMENT_WIDTH, THOUGHT_PLACEMENT_HEIGHT, EDGE_STROKE_COLOR, EDGE_STROKE_WIDTH, GRAPH_MIN_HEIGHT, LINK_MODE_WAITING } from "../lib/config";
import { randomCanvasPosition, deduplicateEdges } from "../lib/utils";
import { useMutationErrors } from "../hooks/useMutationErrors";
import { ErrorBanner } from "./ErrorBanner";
import { ThoughtNode } from "./ThoughtNode";
import type { Thought } from "../lib/types";

interface Props {
  schemeId: string;
}

export function GraphView({ schemeId }: Props) {
  const qc = useQueryClient();
  const [newTitle, setNewTitle] = useState("");
  const [linkSource, setLinkSource] = useState<string | null>(null);

  const { data: thoughts = [] } = useQuery({
    queryKey: ["thoughts", schemeId],
    queryFn: () => listThoughts(schemeId),
  });

  // Collect edges for all thoughts
  const { data: allEdges = [] } = useQuery({
    queryKey: ["edges", schemeId, thoughts.map((t) => t.id).sort().join(",")],
    queryFn: async () => {
      if (thoughts.length === 0) return [];
      const edgeSets = await Promise.all(thoughts.map((t) => listEdges(t.id)));
      return deduplicateEdges(edgeSets);
    },
    enabled: thoughts.length > 0,
  });

  const createMut = useMutation({
    mutationFn: () => {
      const pos = randomCanvasPosition(THOUGHT_PLACEMENT_WIDTH, THOUGHT_PLACEMENT_HEIGHT);
      return createThought({
        scheme_id: schemeId,
        title: newTitle.trim() || "New thought",
        body: "",
        canvas_x: pos.x,
        canvas_y: pos.y,
      });
    },
    onSuccess: () => {
      setNewTitle("");
      qc.invalidateQueries({ queryKey: ["thoughts", schemeId] });
    },
  });

  const deleteMut = useMutation({
    mutationFn: deleteThought,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["thoughts", schemeId] });
      qc.invalidateQueries({ queryKey: ["edges", schemeId] });
    },
  });

  const linkMut = useMutation({
    mutationFn: ({ source, target }: { source: string; target: string }) =>
      createEdge(source, { target_id: target, label: "" }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["edges", schemeId] });
    },
    onSettled: () => {
      // Always exit link mode after attempt, whether success or failure.
      // Prevents user from being stuck in link mode after an error.
      setLinkSource(null);
    },
  });

  const deleteEdgeMut = useMutation({
    mutationFn: ({ thoughtId, edgeId }: { thoughtId: string; edgeId: string }) =>
      deleteEdge(thoughtId, edgeId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["edges", schemeId] }),
  });

  const handleThoughtClick = (thought: Thought) => {
    if (linkSource && linkSource !== thought.id) {
      linkMut.mutate({ source: linkSource, target: thought.id });
    }
  };

  const thoughtMap = new Map(thoughts.map((t) => [t.id, t]));

  const { activeError, resetAll } = useMutationErrors([createMut, deleteMut, linkMut, deleteEdgeMut]);

  return (
    <div data-testid="graph-view" className="flex-1 flex flex-col overflow-hidden">
      {activeError && (
        <div className="px-3 pt-2">
          <ErrorBanner error={activeError} onRetry={resetAll} onDismiss={resetAll} />
        </div>
      )}
      {/* Toolbar */}
      <div className="flex items-center gap-2 px-3 py-2 border-b border-white/10 bg-slate-900/50">
        <input
          data-testid="thought-title-input"
          aria-label="New thought title"
          value={newTitle}
          onChange={(e) => setNewTitle(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && createMut.mutate()}
          placeholder="New thought title..."
          className="flex-1 rounded border border-white/10 bg-black/30 px-2 py-1 text-sm text-white placeholder-slate-500 focus:outline-none focus:ring-1 focus:ring-white/30"
        />
        <button
          data-testid="create-thought-btn"
          onClick={() => createMut.mutate()}
          className="p-1.5 rounded bg-white/10 text-white hover:bg-white/20"
          aria-label="Add thought"
        >
          <Plus className="h-4 w-4" aria-hidden="true" />
        </button>
        <button
          data-testid="link-mode-btn"
          onClick={() => setLinkSource(linkSource ? null : LINK_MODE_WAITING)}
          className={`p-1.5 rounded text-white ${linkSource ? "bg-blue-600" : "bg-white/10 hover:bg-white/20"}`}
          aria-label={linkSource ? "Cancel linking" : "Link thoughts"}
          aria-pressed={!!linkSource}
        >
          <Link className="h-4 w-4" aria-hidden="true" />
        </button>
      </div>

      {/* Link mode guidance */}
      {linkSource && (
        <div data-testid="link-mode-hint" className="flex items-center gap-2 px-3 py-1.5 bg-blue-900/30 border-b border-blue-500/30 text-xs text-blue-300">
          <Link className="h-3 w-3 shrink-0" aria-hidden="true" />
          <span>
            {linkSource === LINK_MODE_WAITING
              ? "Click a thought to select it as the source."
              : "Now click another thought to connect them."}
          </span>
        </div>
      )}
      <div className="sr-only" aria-live="polite" role="status">
        {linkSource
          ? linkSource === LINK_MODE_WAITING
            ? "Link mode active. Click a thought to select it as the source."
            : "Source selected. Click another thought to create a connection."
          : ""}
      </div>

      {/* Graph content */}
      <div className="flex-1 overflow-auto p-4">
        {thoughts.length === 0 && (
          <p className="text-center text-sm text-slate-500 mt-8">No thoughts yet. Add one above.</p>
        )}

        {/* SVG edges */}
        {allEdges.length > 0 && (
          <svg className="absolute inset-0 pointer-events-none" style={{ width: "100%", height: "100%" }}>
            {allEdges.map((edge) => {
              const src = thoughtMap.get(edge.source_id);
              const tgt = thoughtMap.get(edge.target_id);
              if (!src || !tgt) return null;
              return (
                <line
                  key={edge.id}
                  x1={src.canvas_x + 70}
                  y1={src.canvas_y + 20}
                  x2={tgt.canvas_x + 70}
                  y2={tgt.canvas_y + 20}
                  stroke={EDGE_STROKE_COLOR}
                  strokeWidth={EDGE_STROKE_WIDTH}
                />
              );
            })}
          </svg>
        )}

        {/* Thought nodes */}
        <div className="relative" style={{ minHeight: GRAPH_MIN_HEIGHT }}>
          {thoughts.map((t) => (
            <ThoughtNode
              key={t.id}
              thought={t}
              isSource={linkSource === t.id}
              isLinkMode={!!linkSource}
              onClick={() => {
                if (linkSource === LINK_MODE_WAITING) {
                  setLinkSource(t.id);
                } else {
                  handleThoughtClick(t);
                }
              }}
              onDelete={() => deleteMut.mutate(t.id)}
            />
          ))}
        </div>

        {/* Edge list */}
        {allEdges.length > 0 && (
          <div className="mt-6 border-t border-white/10 pt-4">
            <h4 className="text-xs uppercase tracking-widest text-slate-500 mb-2">Connections</h4>
            {allEdges.map((edge) => {
              const src = thoughtMap.get(edge.source_id);
              const tgt = thoughtMap.get(edge.target_id);
              return (
                <div key={edge.id} data-testid="edge-item" className="flex items-center gap-2 text-xs text-slate-400 py-1">
                  <span>{src?.title || "?"}</span>
                  <span className="text-slate-600">&rarr;</span>
                  <span>{tgt?.title || "?"}</span>
                  <button
                    onClick={() => deleteEdgeMut.mutate({ thoughtId: edge.source_id, edgeId: edge.id })}
                    className="ml-auto p-0.5 text-slate-600 hover:text-red-400"
                    aria-label={`Delete connection from ${src?.title || "unknown"} to ${tgt?.title || "unknown"}`}
                  >
                    <Trash2 className="h-3 w-3" aria-hidden="true" />
                  </button>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
