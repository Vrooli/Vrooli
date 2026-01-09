import { useEffect, useMemo, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';
import type {
  ExecutionFeedbackEntry,
  ExecutionFeedbackEntryPayload,
  ProfilePerformance,
} from '@/types/api';

const FEEDBACK_CATEGORIES = [
  { value: 'infrastructure', label: 'Infrastructure / automation' },
  { value: 'layout', label: 'Layout / mobile' },
  { value: 'ux', label: 'UX / polish' },
  { value: 'tests', label: 'Tests / coverage' },
  { value: 'minimal_change', label: 'Minimal change / investigation' },
  { value: 'other', label: 'Other' },
];

const FEEDBACK_SEVERITIES = [
  { value: 'low', label: 'Low' },
  { value: 'medium', label: 'Medium' },
  { value: 'high', label: 'High' },
  { value: 'critical', label: 'Critical' },
];

interface ExecutionFeedbackPanelProps {
  executionId?: string;
  entries?: ExecutionFeedbackEntry[];
  onSubmitted?: () => void;
  disabled?: boolean;
  disabledMessage?: string;
  isLoading?: boolean;
}

export function ExecutionFeedbackPanel({
  executionId,
  entries = [],
  onSubmitted,
  disabled = false,
  disabledMessage,
  isLoading = false,
}: ExecutionFeedbackPanelProps) {
  const [category, setCategory] = useState(FEEDBACK_CATEGORIES[0].value);
  const [severity, setSeverity] = useState('medium');
  const [suggestedAction, setSuggestedAction] = useState('');
  const [comments, setComments] = useState('');
  const [subsystem, setSubsystem] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [statusText, setStatusText] = useState<string | null>(null);
  const [errorText, setErrorText] = useState<string | null>(null);

  useEffect(() => {
    setCategory(FEEDBACK_CATEGORIES[0].value);
    setSeverity('medium');
    setSuggestedAction('');
    setComments('');
    setSubsystem('');
    setStatusText(null);
    setErrorText(null);
  }, [executionId]);

  const handleSubmit = async () => {
    if (!executionId) {
      setErrorText('Execution ID missing');
      return;
    }
    if (!category || !severity) {
      setErrorText('Category and severity are required');
      return;
    }
    setIsSubmitting(true);
    setErrorText(null);
    try {
      await api.submitExecutionFeedbackEntry(executionId, {
        category,
        severity,
        suggested_action: suggestedAction.trim() || undefined,
        comments: comments.trim() || undefined,
        metadata: subsystem.trim() ? { subsystem: subsystem.trim() } : undefined,
      });
      setStatusText('Feedback recorded');
      setSuggestedAction('');
      setComments('');
      setSubsystem('');
      onSubmitted?.();
    } catch (err) {
      console.error('Failed to submit feedback entry', err);
      setErrorText('Unable to submit feedback right now');
    } finally {
      setIsSubmitting(false);
    }
  };

  const sortedEntries = useMemo(
    () =>
      entries
        .slice()
        .sort(
          (a, b) =>
            new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
        ),
    [entries],
  );

  const formDisabled = disabled || !executionId || isLoading;

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div className="text-sm font-semibold text-white">Execution feedback</div>
        <div className="text-xs text-slate-500">Guide insights & follow-up</div>
      </div>
      <div className="rounded-lg border border-white/5 bg-slate-900/70 p-4 space-y-4">
        <div className="space-y-2">
          {sortedEntries.length === 0 ? (
            <div className="text-xs text-slate-400">No feedback yet.</div>
          ) : (
            sortedEntries.map((entry) => (
              <div
                key={entry.id}
                className="rounded-md border border-white/5 bg-slate-950/40 p-3 space-y-1"
              >
                <div className="flex items-center justify-between">
                  <div className="text-xs uppercase tracking-wide text-slate-400">
                    {entry.category}
                  </div>
                  <span
                    className={`text-[11px] rounded-full px-2 py-1 font-semibold uppercase ${severityTone(
                      entry.severity,
                    )}`}
                  >
                    {entry.severity}
                  </span>
                </div>
                <p className="text-sm text-slate-200">
                  {entry.comments || 'No description provided.'}
                </p>
                {entry.suggested_action && (
                  <p className="text-xs text-slate-400">
                    Action: {entry.suggested_action}
                  </p>
                )}
                {entry.metadata?.subsystem && (
                  <p className="text-xs text-slate-400">
                    Subsystem: {entry.metadata.subsystem}
                  </p>
                )}
                <div className="text-[11px] text-slate-500">
                  {formatFeedbackDate(entry.created_at)}
                </div>
              </div>
            ))
          )}
        </div>

        {isLoading && (
          <div className="text-xs text-slate-400">Loading feedback…</div>
        )}

        {disabledMessage && (
          <div className="text-xs text-slate-400">{disabledMessage}</div>
        )}

        <div className="border-t border-white/5 pt-3 text-sm space-y-3">
          <div className="grid gap-3 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="feedback-category">Category</Label>
              <Select
                id="feedback-category"
                value={category}
                onValueChange={(value) => setCategory(value)}
                disabled={formDisabled}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {FEEDBACK_CATEGORIES.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="feedback-severity">Severity</Label>
              <Select
                id="feedback-severity"
                value={severity}
                onValueChange={(value) => setSeverity(value)}
                disabled={formDisabled}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {FEEDBACK_SEVERITIES.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid gap-3 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="feedback-action">Suggested action</Label>
              <Input
                id="feedback-action"
                value={suggestedAction}
                onChange={(event) => setSuggestedAction(event.target.value)}
                placeholder="e.g. pause Auto Steer / rerun tests"
                disabled={formDisabled}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="feedback-subsystem">Affected subsystem</Label>
              <Input
                id="feedback-subsystem"
                value={subsystem}
                onChange={(event) => setSubsystem(event.target.value)}
                placeholder="e.g. browser-automation-studio"
                disabled={formDisabled}
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="feedback-comments">Comments</Label>
            <Textarea
              id="feedback-comments"
              value={comments}
              onChange={(event) => setComments(event.target.value)}
              placeholder="Explain what you observed and what you’d like the agent to focus on."
              className="h-24"
              disabled={formDisabled}
            />
          </div>

          {errorText && (
            <div className="text-xs text-rose-300">{errorText}</div>
          )}
          {statusText && (
            <div className="text-xs text-emerald-300">{statusText}</div>
          )}

          <Button
            size="sm"
            onClick={handleSubmit}
            disabled={formDisabled || isSubmitting}
          >
            {isSubmitting ? 'Submitting...' : 'Report issue'}
          </Button>
        </div>
      </div>
    </div>
  );
}

function severityTone(severity: string) {
  switch (severity) {
    case 'critical':
      return 'bg-rose-500/20 text-rose-200 border border-rose-500/40';
    case 'high':
      return 'bg-orange-500/20 text-orange-200 border border-orange-500/40';
    case 'medium':
      return 'bg-amber-500/20 text-amber-200 border border-amber-500/40';
    default:
      return 'bg-emerald-500/20 text-emerald-200 border border-emerald-500/40';
  }
}

function formatFeedbackDate(value?: string) {
  if (!value) return '--';
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return parsed.toLocaleString();
}

interface ExecutionFeedbackDialogProps {
  executionId?: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ExecutionFeedbackDialog({
  executionId,
  open,
  onOpenChange,
}: ExecutionFeedbackDialogProps) {
  const { data, isFetching, refetch } = useQuery<ProfilePerformance | null>({
    queryKey: executionId ? ['execution-feedback-history', executionId] : ['execution-feedback-history', 'missing'],
    queryFn: async () => {
      if (!executionId) {
        return null;
      }
      try {
        return await api.getAutoSteerExecution(executionId);
      } catch (err) {
        if (err instanceof Error && err.message.includes('(404)')) {
          return null;
        }
        throw err;
      }
    },
    enabled: Boolean(executionId),
    retry: false,
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Execution feedback</DialogTitle>
        </DialogHeader>
        <ExecutionFeedbackPanel
          executionId={executionId ?? undefined}
          entries={data?.feedback_entries ?? []}
          onSubmitted={() => {
            refetch();
            onOpenChange(false);
          }}
          disabled={!executionId}
          disabledMessage={!executionId ? 'Select an execution first.' : undefined}
          isLoading={isFetching}
        />
      </DialogContent>
    </Dialog>
  );
}
