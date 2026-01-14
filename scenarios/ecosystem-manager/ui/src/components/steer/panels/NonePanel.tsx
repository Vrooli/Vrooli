import { Circle } from 'lucide-react';

export function NonePanel() {
  return (
    <div className="flex flex-col items-center justify-center py-8 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-slate-500/10 mb-4">
        <Circle className="h-6 w-6 text-slate-400" />
      </div>
      <h3 className="text-sm font-medium text-slate-200 mb-2">Default Steering</h3>
      <p className="text-sm text-slate-400 max-w-xs">
        No explicit steering configuration. The task will use the{' '}
        <span className="font-medium text-slate-300">Progress</span> mode by default.
      </p>
    </div>
  );
}
