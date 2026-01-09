import { Rocket, AlertCircle, X } from 'lucide-react';

interface PromoteDialogProps {
  scenarioId: string | null;
  onClose: () => void;
  onConfirm: () => void;
}

/**
 * Dialog for confirming scenario promotion to production.
 * Shows warnings about what will happen and provides cancel/confirm options.
 */
export function PromoteDialog({ scenarioId, onClose, onConfirm }: PromoteDialogProps) {
  if (!scenarioId) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fade-in"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-labelledby="promote-dialog-title"
    >
      <div
        className="relative w-full max-w-md rounded-2xl border-2 border-purple-500/40 bg-gradient-to-br from-slate-900 via-slate-900 to-purple-900/30 p-6 shadow-2xl shadow-purple-500/20"
        onClick={(e) => e.stopPropagation()}
      >
        <button
          onClick={onClose}
          className="absolute top-4 right-4 text-slate-400 hover:text-slate-200 transition-colors focus:outline-none focus:ring-2 focus:ring-purple-500 rounded"
          aria-label="Close dialog"
        >
          <X className="h-5 w-5" aria-hidden="true" />
        </button>

        <div className="space-y-4">
          <div className="flex items-start gap-3">
            <div className="flex-shrink-0 w-12 h-12 rounded-xl bg-purple-500/20 border-2 border-purple-500/40 flex items-center justify-center">
              <Rocket className="h-6 w-6 text-purple-300" aria-hidden="true" />
            </div>
            <div className="flex-1">
              <h3 id="promote-dialog-title" className="text-xl font-bold text-purple-100 mb-1">
                Promote to Production?
              </h3>
              <p className="text-sm text-purple-200/80">
                Move <code className="px-1.5 py-0.5 rounded bg-slate-800 text-emerald-300 font-mono text-xs">{scenarioId}</code> to production
              </p>
            </div>
          </div>

          <div className="rounded-xl border border-amber-500/30 bg-amber-500/10 p-4 space-y-2">
            <div className="flex items-start gap-2">
              <AlertCircle className="h-4 w-4 flex-shrink-0 mt-0.5 text-amber-400" aria-hidden="true" />
              <div className="text-xs text-amber-100 space-y-1">
                <p className="font-semibold">This action will:</p>
                <ul className="list-disc list-inside space-y-0.5 text-amber-200/90">
                  <li>Stop the scenario if running</li>
                  <li>Move from <code className="px-1 py-0.5 rounded bg-slate-900/60 font-mono text-[10px]">generated/</code> to <code className="px-1 py-0.5 rounded bg-slate-900/60 font-mono text-[10px]">scenarios/</code></li>
                  <li>Remove from staging list</li>
                </ul>
              </div>
            </div>
          </div>

          <div className="flex gap-3 pt-2">
            <button
              onClick={onClose}
              className="flex-1 px-4 py-2.5 text-sm font-medium rounded-lg border border-slate-500/30 bg-slate-500/10 text-slate-300 hover:bg-slate-500/20 hover:border-slate-400 transition-all focus:outline-none focus:ring-2 focus:ring-slate-500"
            >
              Cancel
            </button>
            <button
              onClick={onConfirm}
              className="flex-1 px-4 py-2.5 text-sm font-bold rounded-lg border-2 border-purple-500/60 bg-gradient-to-r from-purple-500/30 to-pink-500/30 text-purple-100 hover:from-purple-500/40 hover:to-pink-500/40 hover:border-purple-400 transition-all shadow-lg shadow-purple-500/20 focus:outline-none focus:ring-2 focus:ring-purple-500"
            >
              Promote to Production
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
