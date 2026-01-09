import { AlertCircle } from 'lucide-react';

export function NoScenariosWarning() {
  return (
    <div className="rounded-xl border border-amber-500/20 bg-amber-500/5 p-4 text-sm text-amber-100 flex items-start gap-3" role="alert">
      <AlertCircle className="h-4 w-4 mt-0.5 flex-shrink-0 text-amber-400" aria-hidden="true" />
      <div>
        <p className="font-medium mb-1">No scenarios available to customize</p>
        <p className="text-xs text-amber-200/80">Generate a landing page scenario first, then come back here to customize it with AI assistance.</p>
      </div>
    </div>
  );
}
