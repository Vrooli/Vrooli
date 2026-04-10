import { Trash2 } from "lucide-react";
import type { Thought } from "../lib/types";

interface ThoughtNodeProps {
  thought: Thought;
  isSource: boolean;
  isLinkMode: boolean;
  onClick: () => void;
  onDelete: () => void;
}

export function ThoughtNode({ thought, isSource, isLinkMode, onClick, onDelete }: ThoughtNodeProps) {
  return (
    <div
      data-testid="thought-node"
      onClick={onClick}
      style={{ left: thought.canvas_x, top: thought.canvas_y }}
      className={`absolute group rounded-lg border p-3 min-w-[140px] max-w-[220px] cursor-pointer select-none transition-colors ${
        isSource
          ? "border-blue-500 bg-blue-900/30"
          : isLinkMode
            ? "border-white/20 bg-slate-800/90 hover:border-blue-400"
            : "border-white/10 bg-slate-800/90 hover:border-white/20"
      }`}
    >
      <div className="flex items-start justify-between gap-1">
        <h3 className="text-sm font-medium text-white">{thought.title || "Untitled"}</h3>
        <button
          onClick={(e) => {
            e.stopPropagation();
            onDelete();
          }}
          className="p-0.5 rounded opacity-0 group-hover:opacity-100 text-slate-500 hover:text-red-400"
          aria-label={`Delete thought: ${thought.title || "Untitled"}`}
        >
          <Trash2 className="h-3 w-3" aria-hidden="true" />
        </button>
      </div>
      {thought.body && <p className="mt-1 text-xs text-slate-400 line-clamp-3">{thought.body}</p>}
    </div>
  );
}
