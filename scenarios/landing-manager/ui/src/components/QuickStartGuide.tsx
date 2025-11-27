import { Rocket, Monitor, Sparkles } from 'lucide-react';

export function QuickStartGuide() {
  return (
    <div className="rounded-xl border border-emerald-500/30 bg-gradient-to-br from-emerald-500/10 to-blue-500/10 p-5 sm:p-6" role="region" aria-label="Quick start guide">
      <div className="flex items-start gap-4">
        <div className="flex-shrink-0 w-10 h-10 rounded-xl bg-emerald-500/20 border border-emerald-500/40 flex items-center justify-center">
          <Rocket className="h-5 w-5 text-emerald-300" aria-hidden="true" />
        </div>
        <div className="flex-1 space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
            <h2 className="text-lg font-bold text-emerald-100">Get Started in 3 Steps</h2>
            <span className="inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-bold bg-emerald-500/30 border border-emerald-500/50 text-emerald-200 rounded-full">
              <Monitor className="h-3 w-3" aria-hidden="true" />
              <span>100% UI · No Terminal Required</span>
            </span>
          </div>
          <div className="grid gap-3">
            <div className="flex items-start gap-3">
              <div className="flex-shrink-0 w-7 h-7 rounded-lg bg-emerald-500/20 border border-emerald-500/40 flex items-center justify-center text-sm font-bold text-emerald-300">1</div>
              <div className="flex-1">
                <p className="text-sm font-semibold text-emerald-100 mb-1">Select a Template</p>
                <p className="text-xs text-emerald-200/80">Browse templates below and click to select one (already done if you see a green checkmark)</p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <div className="flex-shrink-0 w-7 h-7 rounded-lg bg-emerald-500/20 border border-emerald-500/40 flex items-center justify-center text-sm font-bold text-emerald-300">2</div>
              <div className="flex-1">
                <p className="text-sm font-semibold text-emerald-100 mb-1">Generate Your Landing Page</p>
                <p className="text-xs text-emerald-200/80">Enter a name, click "Generate Now" — your scenario appears instantly in the staging area</p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <div className="flex-shrink-0 w-7 h-7 rounded-lg bg-emerald-500/20 border border-emerald-500/40 flex items-center justify-center text-sm font-bold text-emerald-300">3</div>
              <div className="flex-1">
                <p className="text-sm font-semibold text-emerald-100 mb-1">Launch & Test</p>
                <p className="text-xs text-emerald-200/80">Click "Start" on your scenario → get instant access links to live landing page + admin dashboard</p>
              </div>
            </div>
          </div>
          <div className="flex items-start gap-2 pt-3 border-t border-emerald-500/20">
            <Sparkles className="h-4 w-4 text-blue-300 mt-0.5 flex-shrink-0" aria-hidden="true" />
            <p className="text-xs text-blue-200 font-medium">
              <strong>Iterate risk-free:</strong> All new scenarios start in a staging folder. Test, refine, and customize before moving to production.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
