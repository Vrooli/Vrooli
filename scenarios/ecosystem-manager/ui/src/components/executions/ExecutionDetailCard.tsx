import { cn } from '@/lib/utils';
import type { ExecutionHistory } from '@/types/api';

interface ExecutionDetailCardProps {
  execution: ExecutionHistory | null;
  promptText?: string;
  outputText?: string;
  isLoadingPrompt?: boolean;
  isLoadingOutput?: boolean;
  className?: string;
}

type ExecutionStatus = ExecutionHistory['status'];

const STATUS_STYLES: Record<ExecutionStatus, string> = {
  completed: 'bg-emerald-500/15 text-emerald-100 border-emerald-400/50',
  running: 'bg-amber-400/15 text-amber-100 border-amber-300/60',
  failed: 'bg-red-500/15 text-red-100 border-red-400/60',
  rate_limited: 'bg-orange-500/15 text-orange-100 border-orange-400/60',
};

const stripLogLine = (line: string) => {
  const trimmed = line.trim();
  const match = trimmed.match(/^[0-9T:.\-+]+ \[[^\]]+\] \([^)]+\)\s+(.*)$/);
  if (match && match[1]) {
    return match[1].trim();
  }
  return trimmed;
};

export const extractExecutionFinalMessage = (output?: string, maxLength = 600) => {
  if (!output) return '';
  const lines = output
    .split('\n')
    .map(stripLogLine)
    .filter(Boolean);

  if (lines.length === 0) return '';

  for (let i = lines.length - 1; i >= 0; i -= 1) {
    const line = lines[i].toLowerCase();
    const isSummaryHeading = /^#+\s+(task\s+)?(completion\s+)?summary/.test(line);
    const isFinalHeading = line.startsWith('final response') || line.startsWith('final message');
    if (isSummaryHeading || isFinalHeading) {
      const summaryLines = lines.slice(i + 1);
      if (summaryLines.length > 0) {
        const message = summaryLines.join(' ');
        if (message.length > maxLength) {
          return `${message.slice(0, maxLength)}…`;
        }
        return message;
      }
    }
  }

  const tailLines = lines.slice(-5).join(' ');
  if (tailLines.length > maxLength) {
    return tailLines.slice(tailLines.length - maxLength);
  }
  return tailLines;
};

const formatDateTime = (value?: string) => {
  if (!value) return '—';
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString();
};

const formatDurationMs = (ms?: number) => {
  if (!ms || ms <= 0) return '—';
  const totalSeconds = Math.floor(ms / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const hours = Math.floor(minutes / 60);
  if (hours > 0) {
    return `${hours}h ${minutes % 60}m`;
  }
  if (minutes > 0) {
    return `${minutes}m ${totalSeconds % 60}s`;
  }
  return `${totalSeconds}s`;
};

const formatExecutionDuration = (execution?: ExecutionHistory | null) => {
  if (!execution) return '—';
  if (execution.duration) return execution.duration;
  if (execution.start_time && execution.end_time) {
    const start = new Date(execution.start_time).getTime();
    const end = new Date(execution.end_time).getTime();
    if (!Number.isNaN(start) && !Number.isNaN(end) && end >= start) {
      return formatDurationMs(end - start);
    }
  }
  return '—';
};

const formatSteerInfo = (execution?: ExecutionHistory | null) => {
  if (!execution) return '—';
  const mode = execution.steer_mode || execution.steering_source || '';
  const phase = execution.steer_phase_index ? ` • phase ${execution.steer_phase_index}` : '';
  const iteration = execution.steer_phase_iteration ? ` • iteration ${execution.steer_phase_iteration}` : '';
  return mode ? `${mode}${phase}${iteration}` : '—';
};

const MetaItem = ({ label, value }: { label: string; value?: string | number | null }) => (
  <div className="rounded-md border border-white/5 bg-white/5 p-2">
    <div className="text-[11px] uppercase tracking-wide text-slate-400">{label}</div>
    <div className="text-sm text-white break-words">{value ?? '—'}</div>
  </div>
);

export function ExecutionDetailCard({
  execution,
  promptText,
  outputText,
  isLoadingPrompt,
  isLoadingOutput,
  className,
}: ExecutionDetailCardProps) {
  if (!execution) {
    return (
      <div
        className={cn(
          'rounded-lg border border-dashed border-white/15 bg-slate-950/60 p-6 text-sm text-slate-500 flex items-center justify-center',
          className,
        )}
      >
        Select an execution to inspect its details.
      </div>
    );
  }

  const metadata = execution.metadata && Object.keys(execution.metadata).length > 0
    ? execution.metadata
    : null;

  const pathItems = [
    { label: 'Prompt path', value: execution.prompt_path },
    { label: 'Output path', value: execution.clean_output_path || execution.output_path },
    { label: 'Transcript', value: execution.transcript_path },
    { label: 'Last message', value: execution.last_message_path },
  ].filter((item) => Boolean(item.value));

  return (
    <div
      className={cn(
        'rounded-lg border border-white/10 bg-slate-950/70 p-4 shadow-[0_10px_60px_-35px_rgba(0,0,0,0.9)] flex flex-col gap-4',
        className,
      )}
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="space-y-1">
          <div className="text-[11px] uppercase text-slate-400">Execution</div>
          <div className="font-mono text-sm text-white break-all">{execution.id}</div>
          <div className="text-xs text-slate-400">
            Task: {execution.task_title || execution.task_id || 'Unknown task'}
          </div>
        </div>
        <div className="text-right space-y-1">
          <span className={`inline-flex items-center gap-2 text-xs px-3 py-1 rounded-full border ${STATUS_STYLES[execution.status]}`}>
            {execution.status}
            {execution.rate_limited ? <span className="text-[10px] text-orange-200">rate limited</span> : null}
          </span>
          <div className="text-xs text-slate-400">
            Exit: {execution.exit_reason ?? (execution.exit_code !== undefined ? execution.exit_code : '—')}
          </div>
          <div className="text-[11px] text-slate-500">
            {formatDateTime(execution.start_time)}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
        <MetaItem label="Started" value={formatDateTime(execution.start_time)} />
        <MetaItem label="Ended" value={execution.end_time ? formatDateTime(execution.end_time) : '—'} />
        <MetaItem label="Duration" value={formatExecutionDuration(execution)} />
        <MetaItem label="Timeout" value={execution.timeout_allowed ?? '—'} />
        <MetaItem label="Prompt size" value={execution.prompt_size ?? '—'} />
        <MetaItem label="Agent" value={execution.agent_tag || '—'} />
        <MetaItem label="Process" value={execution.process_id ? `PID ${execution.process_id}` : '—'} />
        <MetaItem label="Steer" value={formatSteerInfo(execution)} />
        <MetaItem label="Auto Steer profile" value={execution.auto_steer_profile_id ?? '—'} />
      </div>

      {pathItems.length > 0 && (
        <div className="space-y-1">
          <div className="text-[11px] uppercase text-slate-400">Paths</div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
            {pathItems.map((item) => (
              <div key={item.label} className="rounded-md border border-white/5 bg-white/5 p-2 text-[11px] text-slate-300 truncate">
                <span className="uppercase text-slate-400">{item.label}: </span>
                <span className="font-mono text-slate-100">{item.value}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="space-y-2">
        <div className="text-[11px] uppercase text-slate-400">Output</div>
        {isLoadingOutput ? (
          <div className="text-xs text-slate-500">Loading output...</div>
        ) : outputText ? (
          <pre className="bg-slate-950/60 border border-white/10 rounded p-3 text-xs text-slate-100 max-h-48 overflow-y-auto whitespace-pre-wrap">
            {outputText}
          </pre>
        ) : (
          <div className="text-xs text-slate-500">No output captured for this execution.</div>
        )}
      </div>

      <div className="space-y-2">
        <div className="text-[11px] uppercase text-slate-400">Prompt sent to agent</div>
        {isLoadingPrompt ? (
          <div className="text-xs text-slate-500">Loading prompt...</div>
        ) : promptText ? (
          <pre className="bg-slate-950/60 border border-white/10 rounded p-3 text-xs text-slate-100 max-h-48 overflow-y-auto whitespace-pre-wrap">
            {promptText}
          </pre>
        ) : (
          <div className="text-xs text-slate-500">Prompt not captured for this execution.</div>
        )}
      </div>

      {metadata ? (
        <div className="space-y-1">
          <div className="text-[11px] uppercase text-slate-400">Metadata</div>
          <pre className="text-xs text-slate-200 bg-black/30 border border-white/5 rounded p-2 overflow-x-auto">
            {JSON.stringify(metadata, null, 2)}
          </pre>
        </div>
      ) : (
        <div className="text-[11px] uppercase text-slate-500">No additional metadata</div>
      )}
    </div>
  );
}
