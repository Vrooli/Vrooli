/**
 * Skeleton loader for template cards.
 * Shows while template data is loading.
 */
export function TemplateCardSkeleton() {
  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4 sm:p-5 animate-pulse" aria-hidden="true">
      <div className="flex items-start justify-between gap-3 mb-2">
        <div className="flex-1 space-y-2">
          <div className="h-3 w-16 bg-slate-700/50 rounded"></div>
          <div className="h-5 w-40 bg-slate-700/50 rounded"></div>
        </div>
        <div className="h-5 w-10 bg-slate-700/50 rounded-full"></div>
      </div>
      <div className="space-y-2 mt-2">
        <div className="h-3 w-full bg-slate-700/50 rounded"></div>
        <div className="h-3 w-3/4 bg-slate-700/50 rounded"></div>
      </div>
    </div>
  );
}
