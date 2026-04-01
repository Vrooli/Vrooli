import { useState } from "react";
import { Check, Pencil, Trash2, RefreshCw } from "lucide-react";
import { cn } from "../../lib";
import type { ClarificationImpact } from "../../types/domain";

interface ClarificationActionButtonsProps {
  impact?: ClarificationImpact;
  disabled?: boolean;
  onAction: (action: string) => void;
}

export function ClarificationActionButtons({ impact, disabled, onAction }: ClarificationActionButtonsProps) {
  const [confirming, setConfirming] = useState<string | null>(null);

  const handleAction = (action: string) => {
    if (action === "remove_decision" || action === "invalidate_round") {
      if (confirming === action) {
        setConfirming(null);
        onAction(action);
      } else {
        setConfirming(action);
      }
    } else {
      onAction(action);
    }
  };

  const suggested = impact?.level;

  const actions = [
    { key: "got_it", label: "Got it", icon: Check, highlight: suggested === "none" },
    { key: "update_decision", label: "Update decision", icon: Pencil, highlight: suggested === "decision" },
    { key: "remove_decision", label: "Remove decision", icon: Trash2, highlight: false, confirm: true },
    { key: "invalidate_round", label: "Invalidate round", icon: RefreshCw, highlight: suggested === "round", confirm: true },
  ];

  return (
    <div className="flex flex-wrap gap-2">
      {actions.map(({ key, label, icon: Icon, highlight }) => {
        const isConfirming = confirming === key;
        return (
          <button
            key={key}
            type="button"
            disabled={disabled}
            onClick={() => handleAction(key)}
            className={cn(
              "flex items-center gap-1.5 rounded-md border px-2.5 py-1.5 text-xs font-medium transition-colors",
              isConfirming
                ? "border-red-500/40 bg-red-500/10 text-red-400"
                : highlight
                  ? "border-cyan-500/40 bg-cyan-500/10 text-cyan-400"
                  : "border-slate-600 bg-slate-800/50 text-slate-400 hover:border-slate-500 hover:text-slate-300",
              disabled && "opacity-50 cursor-not-allowed",
            )}
          >
            <Icon className="h-3 w-3" />
            {isConfirming ? `Confirm ${label.toLowerCase()}?` : label}
            {highlight && !isConfirming && (
              <span className="rounded bg-cyan-500/15 px-1 py-0.5 text-[8px] text-cyan-400">Suggested</span>
            )}
          </button>
        );
      })}
    </div>
  );
}
