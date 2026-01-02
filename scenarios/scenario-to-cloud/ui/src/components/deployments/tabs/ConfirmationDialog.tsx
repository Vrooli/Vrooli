import { Loader2, AlertTriangle } from "lucide-react";
import { cn } from "../../../lib/utils";

interface ConfirmationDialogProps {
  title: string;
  description: string;
  confirmText: string;
  inputValue: string;
  onInputChange: (value: string) => void;
  onConfirm: () => void;
  onCancel: () => void;
  isPending: boolean;
  isDestructive?: boolean;
}

export function ConfirmationDialog({
  title,
  description,
  confirmText,
  inputValue,
  onInputChange,
  onConfirm,
  onCancel,
  isPending,
  isDestructive = false,
}: ConfirmationDialogProps) {
  const canConfirm = inputValue === confirmText;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-slate-900 border border-white/10 rounded-lg p-6 max-w-md w-full mx-4 shadow-xl">
        <div className="flex items-start gap-3 mb-4">
          {isDestructive && (
            <div className="p-2 rounded-lg bg-red-500/20">
              <AlertTriangle className="h-5 w-5 text-red-400" />
            </div>
          )}
          <div>
            <h3
              className={cn(
                "text-lg font-semibold",
                isDestructive ? "text-red-400" : "text-white"
              )}
            >
              {title}
            </h3>
            <p className="text-sm text-slate-400 mt-1">{description}</p>
          </div>
        </div>

        <div className="mb-4">
          <label className="block text-sm text-slate-400 mb-2">
            Type{" "}
            <code className="bg-slate-800 px-1.5 py-0.5 rounded text-amber-400 font-mono">
              {confirmText}
            </code>{" "}
            to confirm
          </label>
          <input
            type="text"
            value={inputValue}
            onChange={(e) => onInputChange(e.target.value)}
            className="w-full px-3 py-2 rounded-lg bg-slate-800 border border-white/10 text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono"
            placeholder={confirmText}
            autoFocus
            disabled={isPending}
          />
        </div>

        <div className="flex justify-end gap-3">
          <button
            onClick={onCancel}
            disabled={isPending}
            className="px-4 py-2 rounded-lg font-medium text-slate-400 hover:text-white hover:bg-white/5 transition-colors disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            disabled={!canConfirm || isPending}
            className={cn(
              "px-4 py-2 rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2",
              isDestructive
                ? "bg-red-500 hover:bg-red-600 text-white"
                : "bg-blue-500 hover:bg-blue-600 text-white"
            )}
          >
            {isPending && <Loader2 className="h-4 w-4 animate-spin" />}
            {isPending ? "Processing..." : "Confirm"}
          </button>
        </div>
      </div>
    </div>
  );
}
