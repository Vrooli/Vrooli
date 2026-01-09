import { FileText, RefreshCcw } from 'lucide-react';

interface TemplateEmptyStateProps {
  onRetry: () => void;
}

export function TemplateEmptyState({ onRetry }: TemplateEmptyStateProps) {
  return (
    <div
      className="col-span-full rounded-xl border border-white/10 bg-slate-900/40 p-8 sm:p-12 text-center space-y-4"
      role="status"
    >
      <div className="flex justify-center">
        <FileText className="h-12 w-12 text-slate-600" aria-hidden="true" />
      </div>
      <div className="space-y-2">
        <h3 className="text-lg font-semibold text-slate-300">No templates available</h3>
        <p className="text-sm text-slate-400 max-w-md mx-auto">
          Templates will appear here once they're loaded. Try refreshing or check your API connection.
        </p>
      </div>
      <button
        onClick={onRetry}
        className="inline-flex items-center gap-2 px-4 py-2 text-sm bg-emerald-500/10 border border-emerald-500/30 text-emerald-300 rounded-lg hover:bg-emerald-500/20 transition-colors focus:outline-none focus:ring-2 focus:ring-emerald-500"
      >
        <RefreshCcw className="h-4 w-4" aria-hidden="true" />
        Try again
      </button>
    </div>
  );
}
