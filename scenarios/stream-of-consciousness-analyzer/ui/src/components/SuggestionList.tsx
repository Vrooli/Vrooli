// DOC: docs/concepts/ARCHITECTURE.md#ui-layer
// [REQ:P1-001] [REQ:P1-003] LLM suggestions as dismissible ghost nodes
import { useState } from "react";
import { X, Sparkles } from "lucide-react";

interface Suggestion {
  id: string;
  source_id: string;
  target_id: string;
  label: string;
  confidence: number;
  dismissed: boolean;
}

interface Props {
  suggestions: Suggestion[];
  onDismiss: (id: string) => void;
  onAccept: (suggestion: Suggestion) => void;
}

export function SuggestionList({ suggestions, onDismiss, onAccept }: Props) {
  const [showDismissed, setShowDismissed] = useState(false);

  const visible = showDismissed ? suggestions : suggestions.filter((s) => !s.dismissed);

  if (suggestions.length === 0) return null;

  return (
    <div data-testid="suggestion-list" className="border-t border-white/10 p-3">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-1.5 text-xs text-slate-400">
          <Sparkles className="h-3 w-3" aria-hidden="true" />
          <span>Suggestions ({visible.length})</span>
        </div>
        {suggestions.some((s) => s.dismissed) && (
          <button
            onClick={() => setShowDismissed(!showDismissed)}
            className="text-[10px] text-slate-600 hover:text-slate-400"
          >
            {showDismissed ? "Hide dismissed" : "Show all"}
          </button>
        )}
      </div>
      <div className="space-y-1">
        {visible.map((s) => (
          <div
            key={s.id}
            data-testid="suggestion-item"
            className={`flex items-center gap-2 rounded px-2 py-1 text-xs ${
              s.dismissed ? "opacity-40" : "bg-white/5 hover:bg-white/10"
            }`}
          >
            <button
              onClick={() => onAccept(s)}
              className="flex-1 text-left text-slate-300 truncate"
              disabled={s.dismissed}
              aria-label={`Accept suggestion: ${s.label || "Connection"}`}
            >
              {s.label || "Connection"}{" "}
              <span className="text-slate-600">({Math.round(s.confidence * 100)}%)</span>
            </button>
            {!s.dismissed && (
              <button
                onClick={() => onDismiss(s.id)}
                className="p-0.5 text-slate-600 hover:text-red-400"
                aria-label={`Dismiss suggestion: ${s.label || "Connection"}`}
              >
                <X className="h-3 w-3" aria-hidden="true" />
              </button>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
