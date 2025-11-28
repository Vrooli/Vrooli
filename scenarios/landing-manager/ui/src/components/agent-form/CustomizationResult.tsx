import { CheckCircle } from 'lucide-react';
import { type CustomizeResult } from '../../lib/api';

interface CustomizationResultProps {
  result: CustomizeResult;
}

export function CustomizationResult({ result }: CustomizationResultProps) {
  return (
    <div className="rounded-xl border border-white/10 bg-slate-900/50 p-4 space-y-2 text-sm text-slate-200" data-testid="customization-result">
      <div className="flex items-center gap-2">
        <CheckCircle className="h-4 w-4 text-emerald-300" />
        <div data-testid="customization-result-message">Issue filed and agent queued</div>
      </div>
      <div className="grid md:grid-cols-2 gap-2 text-xs text-slate-300">
        <div data-testid="customization-issue-id">Issue ID: {result.issue_id || 'unknown'}</div>
        <div data-testid="customization-agent">Agent: {result.agent || 'auto'}</div>
        <div data-testid="customization-run-id">Run ID: {result.run_id || 'pending'}</div>
        <div data-testid="customization-status">Status: {result.status}</div>
      </div>
      {result.tracker_url && (
        <div className="text-xs text-slate-300">
          Tracker API: <code className="px-1 py-0.5 rounded bg-slate-900">{result.tracker_url}</code>
        </div>
      )}
      {result.message && <div className="text-xs text-slate-400">{result.message}</div>}
    </div>
  );
}
