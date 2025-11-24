/**
 * TaskDetailsModal Component
 * Modal for viewing and editing task details with 5 tabs
 */

import { useState, useEffect, useMemo } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Checkbox } from '@/components/ui/checkbox';
import { Save, Archive, Trash2 } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { useUpdateTask, useDeleteTask } from '@/hooks/useTaskMutations';
import { useAutoSteerProfiles, useAutoSteerExecutionState, useResetAutoSteerExecution } from '@/hooks/useAutoSteer';
import { api } from '@/lib/api';
import { markdownToHtml } from '@/lib/markdown';
import { queryKeys } from '@/lib/queryKeys';
import type { Task, Priority, ExecutionHistory, UpdateTaskInput, LogEntry, SteerMode } from '@/types/api';
import { STEER_MODES } from '@/types/api';

interface TaskDetailsModalProps {
  task: Task | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const stripLogLine = (line: string) => {
  const trimmed = line.trim();
  const match = trimmed.match(/^[0-9T:.\-+]+ \[[^\]]+\] \([^)]+\)\s+(.*)$/);
  if (match && match[1]) {
    return match[1].trim();
  }
  return trimmed;
};

const PRIORITIES: Priority[] = ['critical', 'high', 'medium', 'low'];
const AUTO_STEER_NONE = 'none';

export function TaskDetailsModal({ task, open, onOpenChange }: TaskDetailsModalProps) {
  const [activeTab, setActiveTab] = useState('details');

  // Form state
  const [targetName, setTargetName] = useState('');
  const [priority, setPriority] = useState<Priority>('medium');
  const [notes, setNotes] = useState('');
  const [steerMode, setSteerMode] = useState<SteerMode | 'none'>('none');
  const [autoSteerProfileId, setAutoSteerProfileId] = useState<string>(AUTO_STEER_NONE);
  const [autoRequeue, setAutoRequeue] = useState(false);
  const [selectedExecutionId, setSelectedExecutionId] = useState<string | null>(null);

  const updateTask = useUpdateTask();
  const deleteTask = useDeleteTask();
  const { data: profiles = [] } = useAutoSteerProfiles();
  const {
    data: autoSteerState,
    isFetching: isAutoSteerStateLoading,
    isError: autoSteerStateError,
    refetch: refetchAutoSteerState,
  } = useAutoSteerExecutionState(task && autoSteerProfileId !== AUTO_STEER_NONE ? task.id : undefined);
  const resetAutoSteer = useResetAutoSteerExecution();

  // Fetch task logs
  const { data: rawLogs = [] } = useQuery({
    queryKey: queryKeys.tasks.logs(task?.id ?? ''),
    queryFn: () => api.getTaskLogs(task!.id),
    enabled: !!task && activeTab === 'logs',
  });
  const logs: LogEntry[] = Array.isArray(rawLogs) ? rawLogs as LogEntry[] : ((rawLogs as any)?.entries ?? []) as LogEntry[];

  // Fetch task prompt
  const { data: promptData } = useQuery({
    queryKey: queryKeys.tasks.prompt(task?.id ?? ''),
    queryFn: () => api.getAssembledPrompt(task!.id),
    enabled: !!task && activeTab === 'prompt',
  });
  const assembledPrompt =
    typeof promptData === 'string'
      ? promptData
      : promptData
        ? JSON.stringify(promptData, null, 2)
        : '';

  // Fetch task executions
  const { data: rawExecutions = [], isFetching: isFetchingExecutions } = useQuery({
    queryKey: queryKeys.tasks.executions(task?.id ?? ''),
    queryFn: () => api.getExecutionHistory(task!.id),
    enabled: !!task,
    staleTime: 10000,
  });
  const executions = Array.isArray(rawExecutions)
    ? rawExecutions
    : (rawExecutions as any)?.executions ?? [];
  const sortedExecutions = useMemo(() => {
    return [...executions].sort((a, b) => {
      const aTime = a?.start_time ? new Date(a.start_time).getTime() : 0;
      const bTime = b?.start_time ? new Date(b.start_time).getTime() : 0;
      if (aTime === bTime) {
        return (b?.id ?? '').localeCompare(a?.id ?? '');
      }
      return bTime - aTime;
    });
  }, [executions]);
  const latestExecution = sortedExecutions[0] ?? null;
  const selectedExecution = sortedExecutions.find(exec => exec.id === selectedExecutionId) || null;

  // Initialize form when task changes
  useEffect(() => {
    if (task) {
      const derivedTarget = (Array.isArray(task.target) && task.target.length > 0)
        ? task.target[0]
        : task.title;
      setTargetName(derivedTarget || '');
      setPriority(task.priority);
      setNotes(task.notes || '');
      const initialMode: SteerMode | 'none' =
        task.auto_steer_profile_id && task.auto_steer_profile_id !== AUTO_STEER_NONE
          ? 'none'
          : (task.steer_mode as SteerMode) || 'none';
      setSteerMode(initialMode);
      setAutoSteerProfileId(task.auto_steer_profile_id || AUTO_STEER_NONE);
      setAutoRequeue(task.auto_requeue || false);
    }
  }, [task]);

  useEffect(() => {
    if (task && autoSteerProfileId !== AUTO_STEER_NONE) {
      refetchAutoSteerState();
    }
  }, [task, autoSteerProfileId, refetchAutoSteerState]);

  // Reset when modal closes
  useEffect(() => {
    if (!open) {
      setActiveTab('details');
      setSelectedExecutionId(null);
    }
  }, [open]);

  useEffect(() => {
    if (autoSteerProfileId !== AUTO_STEER_NONE && steerMode !== 'none') {
      setSteerMode('none');
    }
  }, [autoSteerProfileId, steerMode]);

  useEffect(() => {
    if (sortedExecutions.length > 0 && !selectedExecutionId) {
      const firstId = sortedExecutions.find(exec => exec.id)?.id ?? sortedExecutions[0]?.id ?? null;
      setSelectedExecutionId(firstId);
    } else if (selectedExecutionId && !sortedExecutions.some(exec => exec.id === selectedExecutionId)) {
      const fallbackId = sortedExecutions.find(exec => exec.id)?.id ?? sortedExecutions[0]?.id ?? null;
      setSelectedExecutionId(fallbackId);
    }
  }, [sortedExecutions, selectedExecutionId]);

  const latestExecutionId = latestExecution?.id ?? null;

  const {
    data: selectedExecutionOutput,
    isLoading: isLoadingSelectedOutput,
  } = useQuery({
    queryKey: selectedExecutionId
      ? queryKeys.executions.output(task?.id ?? '', selectedExecutionId)
      : ['executions', 'output', 'inactive', task?.id ?? ''],
    queryFn: () => api.getExecutionOutput(task!.id, selectedExecutionId!),
    enabled: !!task && !!selectedExecutionId,
    staleTime: 15000,
  });

  const {
    data: selectedExecutionPrompt,
    isLoading: isLoadingSelectedPrompt,
  } = useQuery({
    queryKey: selectedExecutionId
      ? queryKeys.executions.prompt(task?.id ?? '', selectedExecutionId)
      : ['executions', 'prompt', 'inactive', task?.id ?? ''],
    queryFn: () => api.getExecutionPrompt(task!.id, selectedExecutionId!),
    enabled: !!task && !!selectedExecutionId,
    staleTime: 15000,
  });

  const {
    data: latestExecutionOutput,
    isLoading: isLoadingLatestOutput,
  } = useQuery({
    queryKey: latestExecutionId
      ? queryKeys.executions.output(task?.id ?? '', latestExecutionId)
      : ['executions', 'output', 'latest', task?.id ?? ''],
    queryFn: () => api.getExecutionOutput(task!.id, latestExecutionId!),
    enabled: !!task && !!latestExecutionId,
    staleTime: 15000,
  });

  const formatDateTime = (value?: string) => {
    if (!value) return '—';
    const parsed = new Date(value);
    return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString();
  };

  const formatDurationMs = (ms: number) => {
    if (!ms || ms < 0) return '—';
    const totalSeconds = Math.round(ms / 1000);
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

  const getStatusTone = (status: ExecutionHistory['status']) => {
    switch (status) {
      case 'completed':
        return 'text-emerald-400';
      case 'failed':
        return 'text-red-400';
      case 'rate_limited':
        return 'text-amber-300';
      default:
        return 'text-yellow-300';
    }
  };

  const extractFinalMessage = (output?: string, maxLength = 600) => {
    if (!output) return '';
    const lines = output
      .split('\n')
      .map(stripLogLine)
      .filter(Boolean);

    if (lines.length === 0) return '';

    // Prefer the last summary/final section if present
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
      return `${tailLines.slice(tailLines.length - maxLength)}`;
    }
    return tailLines;
  };

  const selectedOutputText =
    (selectedExecutionOutput as any)?.output ??
    (selectedExecutionOutput as any)?.content ??
    '';
  const selectedPromptText =
    (selectedExecutionPrompt as any)?.prompt ??
    (selectedExecutionPrompt as any)?.content ??
    '';
  const latestOutputText =
    (latestExecutionOutput as any)?.output ??
    (latestExecutionOutput as any)?.content ??
    '';
  const selectedFinalMessage = useMemo(
    () => extractFinalMessage(selectedOutputText),
    [selectedOutputText],
  );
  const latestFinalMessage = useMemo(
    () => extractFinalMessage(latestOutputText),
    [latestOutputText],
  );

  const assembledPromptHtml = useMemo(() => markdownToHtml(assembledPrompt), [assembledPrompt]);

  if (!task) return null;

  const activeProfile = profiles.find(profile => profile.id === autoSteerProfileId);
  const currentPhaseNumber = autoSteerState ? autoSteerState.current_phase_index + 1 : 0;
  const totalPhases = activeProfile?.phases?.length ?? 0;
  const currentMode =
    activeProfile?.phases?.[autoSteerState?.current_phase_index ?? 0]?.mode ??
    (activeProfile?.phases?.[0]?.mode ?? undefined);
  const autoSteerIteration = autoSteerState?.auto_steer_iteration ?? 0;

  const handleResetAutoSteer = async () => {
    if (!task) return;
    await resetAutoSteer.mutateAsync(task.id);
    await refetchAutoSteerState();
  };

  const handleSave = () => {
    if (!task) return;
    if (canEditTarget && !normalizedTarget) {
      return;
    }

    const updates: UpdateTaskInput = {
      priority,
      notes: notes.trim() || undefined,
      steer_mode: autoSteerProfileId === AUTO_STEER_NONE && steerMode !== 'none' ? steerMode : 'none',
      auto_steer_profile_id: autoSteerProfileId === AUTO_STEER_NONE ? undefined : autoSteerProfileId,
      auto_requeue: autoRequeue,
    };

    if (canEditTarget && normalizedTarget) {
      updates.target = [normalizedTarget];
    }

    updateTask.mutate({
      id: task.id,
      updates,
    }, {
      onSuccess: () => {
        onOpenChange(false);
      },
    });
  };

  const handleDelete = () => {
    if (confirm(`Delete task "${task.title}"?`)) {
      deleteTask.mutate(task.id, {
        onSuccess: () => {
          onOpenChange(false);
        },
      });
    }
  };

  const handleArchive = () => {
    updateTask.mutate({
      id: task.id,
      updates: { status: 'archived' } as any,
    }, {
      onSuccess: () => {
        onOpenChange(false);
      },
    });
  };

  const canEditTarget = task?.operation === 'generator' && task?.status === 'pending';
  const normalizedTarget = targetName.trim();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-5xl md:max-w-6xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Task Details</DialogTitle>
          <DialogDescription>
            {task.type} • {task.operation} • {task.status}
          </DialogDescription>
        </DialogHeader>

        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <TabsList className="grid w-full grid-cols-5">
            <TabsTrigger value="details">Details</TabsTrigger>
            <TabsTrigger value="logs">Logs</TabsTrigger>
            <TabsTrigger value="prompt">Prompt</TabsTrigger>
            <TabsTrigger value="recycler">Recycler</TabsTrigger>
            <TabsTrigger value="executions">Executions</TabsTrigger>
          </TabsList>

          {/* Details Tab */}
          <TabsContent value="details" className="mt-4">
            <div className="grid gap-6 md:grid-cols-[minmax(0,1fr)_minmax(0,0.9fr)]">
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="detail-target">Target</Label>
                  <Input
                    id="detail-target"
                    value={targetName}
                    onChange={(e) => setTargetName(e.target.value)}
                    disabled={!canEditTarget}
                    placeholder="No target recorded"
                  />
                  {!canEditTarget && (
                    <p className="text-xs text-muted-foreground">
                      Target can only be edited for pending generator tasks.
                    </p>
                  )}
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Type</Label>
                    <Input value={task.type} disabled />
                  </div>

                  <div className="space-y-2">
                    <Label>Operation</Label>
                    <Input value={task.operation} disabled />
                  </div>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="detail-priority">Priority</Label>
                  <Select value={priority} onValueChange={(val: string) => setPriority(val as Priority)}>
                    <SelectTrigger id="detail-priority">
                      <SelectValue />
                    </SelectTrigger>
                <SelectContent>
                  {PRIORITIES.map(p => (
                    <SelectItem key={p} value={p}>
                      {p.charAt(0).toUpperCase() + p.slice(1)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {task?.type === 'scenario' && task?.operation === 'improver' && (
              <div className="space-y-2">
                <Label htmlFor="detail-steer-mode">Steering focus (manual)</Label>
                <Select
                  value={steerMode}
                  onValueChange={(val: string) => {
                    const mode = val as SteerMode | 'none';
                    setSteerMode(mode);
                    if (mode !== 'none') {
                      setAutoSteerProfileId(AUTO_STEER_NONE);
                    }
                  }}
                  disabled={autoSteerProfileId !== AUTO_STEER_NONE}
                >
                  <SelectTrigger id="detail-steer-mode">
                    <SelectValue placeholder="None (use Progress)" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">None (use Progress)</SelectItem>
                    {STEER_MODES.map(mode => (
                      <SelectItem key={mode} value={mode}>
                        {mode.toUpperCase()}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-slate-500">
                  Pick a manual focus OR an Auto Steer profile. Selecting a profile disables manual steering until cleared.
                </p>
              </div>
            )}

            {(profiles && profiles.length > 0) && (
              <div className="space-y-2">
                <Label htmlFor="detail-auto-steer">Auto Steer Profile</Label>
                <Select
                  value={autoSteerProfileId}
                  onValueChange={(val: string) => {
                    setAutoSteerProfileId(val);
                    if (val !== AUTO_STEER_NONE) {
                      setSteerMode('none');
                    }
                  }}
                >
                  <SelectTrigger id="detail-auto-steer">
                    <SelectValue placeholder="None" />
                  </SelectTrigger>
                  <SelectContent>
                        <SelectItem value={AUTO_STEER_NONE}>None</SelectItem>
                        {profiles.map(profile => (
                          <SelectItem key={profile.id} value={profile.id}>
                            {profile.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                )}

                {autoSteerProfileId !== AUTO_STEER_NONE && (
                  <div className="rounded-md border border-slate-800 bg-slate-900/50 p-3 text-xs text-slate-200 space-y-2">
                    <div className="flex items-center justify-between gap-2">
                      <div className="text-sm font-medium text-slate-100">Auto Steer status</div>
                      <div className="flex gap-2">
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => refetchAutoSteerState()}
                          disabled={isAutoSteerStateLoading}
                        >
                          Refresh
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={handleResetAutoSteer}
                          disabled={resetAutoSteer.isPending || !task}
                        >
                          Restart
                        </Button>
                      </div>
                    </div>
                    <div className="grid grid-cols-2 gap-2">
                      <div>
                        <div className="text-slate-400">Iteration</div>
                        <div className="font-semibold text-slate-50">{autoSteerIteration}</div>
                      </div>
                      <div>
                        <div className="text-slate-400">Phase</div>
                        <div className="font-semibold text-slate-50">
                          {autoSteerState
                            ? `Phase ${currentPhaseNumber}${totalPhases ? ` of ${totalPhases}` : ''}`
                            : 'Not started'}
                        </div>
                      </div>
                      <div>
                        <div className="text-slate-400">Mode</div>
                        <div className="font-semibold text-slate-50">
                          {currentMode ? currentMode.toUpperCase() : '—'}
                        </div>
                      </div>
                      <div>
                        <div className="text-slate-400">Updated</div>
                        <div className="font-semibold text-slate-50">
                          {autoSteerState?.last_updated
                            ? new Date(autoSteerState.last_updated).toLocaleString()
                            : '—'}
                        </div>
                      </div>
                    </div>
                    {autoSteerStateError && (
                      <div className="text-amber-400 text-[11px]">
                        No active Auto Steer state yet. It will initialize on next run.
                      </div>
                    )}
                  </div>
                )}

                <div className="flex items-center gap-2">
                  <Checkbox
                    id="detail-auto-requeue"
                    checked={autoRequeue}
                    onCheckedChange={(checked: boolean) => setAutoRequeue(checked)}
                  />
                  <Label htmlFor="detail-auto-requeue" className="cursor-pointer">
                    Auto-requeue on completion
                  </Label>
                </div>

                <div className="grid grid-cols-2 gap-4 text-xs text-slate-400">
                  <div>
                    <span className="font-medium">Completions:</span> {task.completion_count ?? 0}
                  </div>
                  <div>
                    <span className="font-medium">Last completed:</span>{' '}
                    {task.last_completed_at ? formatDateTime(task.last_completed_at) : '—'}
                  </div>
                  <div>
                    <span className="font-medium">Created:</span> {new Date(task.created_at).toLocaleString()}
                  </div>
                  <div>
                    <span className="font-medium">Updated:</span> {new Date(task.updated_at).toLocaleString()}
                  </div>
                </div>
              </div>

              <div className="space-y-3">
                <div className="rounded-md border border-white/10 bg-slate-900/70 p-3 shadow-inner">
                  {latestExecution ? (
                    <div className="space-y-2">
                      <div className="flex items-center justify-between gap-3">
                        <div>
                          <div className="text-xs uppercase text-slate-400">Last execution</div>
                          <div className="text-sm font-semibold text-foreground">
                            {formatDateTime(latestExecution.end_time ?? latestExecution.start_time)}
                          </div>
                          <div className="text-xs text-slate-400">
                            Status:{' '}
                            <span className={getStatusTone(latestExecution.status)}>
                              {latestExecution.status}
                            </span>
                            {' • '}
                            Duration: {formatExecutionDuration(latestExecution)}
                          </div>
                          <div className="text-xs text-slate-400">
                            Timeout: {latestExecution.timeout_allowed ?? '—'}
                          </div>
                        </div>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => setActiveTab('executions')}
                        >
                          View details
                        </Button>
                      </div>
                      <div className="space-y-1">
                        <div className="text-[11px] uppercase text-slate-500">Final response</div>
                        {isLoadingLatestOutput ? (
                          <div className="text-xs text-slate-500">Loading output...</div>
                        ) : latestFinalMessage ? (
                          <div className="rounded-md border border-white/5 bg-slate-800/60 p-2 text-sm text-slate-100 whitespace-pre-wrap max-h-28 overflow-y-auto">
                            {latestFinalMessage}
                          </div>
                        ) : (
                          <div className="text-xs text-slate-500">No output captured yet.</div>
                        )}
                      </div>
                    </div>
                  ) : (
                    <div className="text-xs text-slate-500">
                      {isFetchingExecutions ? 'Loading executions...' : 'No executions yet.'}
                    </div>
                  )}
                </div>

                <Label htmlFor="detail-notes">Notes</Label>
                <Textarea
                  id="detail-notes"
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  className="min-h-[240px]"
                />
              </div>
            </div>
          </TabsContent>

          {/* Logs Tab */}
          <TabsContent value="logs" className="mt-4">
            <div className="border rounded-md p-4 max-h-96 overflow-y-auto bg-slate-900 font-mono text-xs">
              {logs.length === 0 ? (
                <div className="text-slate-400 text-center py-8">No logs available</div>
              ) : (
                logs.map((log, idx) => (
                  <div key={idx} className="py-1">
                    <span className="text-slate-500">{log.timestamp}</span>{' '}
                    <span className={`font-semibold ${
                      log.level === 'error' ? 'text-red-400' :
                      log.level === 'warning' ? 'text-yellow-400' :
                      log.level === 'info' ? 'text-blue-400' :
                      'text-slate-300'
                    }`}>
                      [{log.level.toUpperCase()}]
                    </span>{' '}
                    <span className="text-slate-200">{log.message}</span>
                  </div>
                ))
              )}
            </div>
          </TabsContent>

          {/* Prompt Tab */}
          <TabsContent value="prompt" className="mt-4">
            <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)]">
              <div className="border rounded-md p-4 max-h-[420px] overflow-y-auto bg-slate-900 font-mono text-xs whitespace-pre-wrap">
                {assembledPrompt || 'No prompt available'}
              </div>
              <div className="border rounded-md p-4 max-h-[420px] overflow-y-auto bg-card">
                <div className="text-xs uppercase text-slate-400 mb-2">Preview</div>
                {assembledPrompt ? (
                  <div
                    className="text-sm leading-relaxed space-y-3 [&_code]:bg-black/40 [&_code]:px-1.5 [&_code]:py-0.5 [&_code]:rounded [&_pre]:text-xs [&_pre]:leading-relaxed [&_ul]:list-disc [&_ul]:pl-5"
                    dangerouslySetInnerHTML={{ __html: assembledPromptHtml }}
                  />
                ) : (
                  <div className="text-slate-500 text-sm">No prompt available</div>
                )}
              </div>
            </div>
          </TabsContent>

          {/* Recycler Tab */}
          <TabsContent value="recycler" className="mt-4">
            <div className="text-slate-400 text-center py-8">
              Recycler history coming soon
            </div>
          </TabsContent>

          {/* Executions Tab */}
          <TabsContent value="executions" className="mt-4">
            {sortedExecutions.length === 0 ? (
              <div className="text-slate-400 text-center py-8">
                {isFetchingExecutions ? 'Loading executions...' : 'No executions yet'}
              </div>
            ) : (
              <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)]">
                <div className="space-y-3 max-h-96 overflow-y-auto pr-1">
                  {sortedExecutions.map((exec, idx) => {
                    const isSelected = exec.id === selectedExecutionId;
                    return (
                      <div
                        key={exec.id}
                        role="button"
                        tabIndex={0}
                        onClick={() => setSelectedExecutionId(exec.id)}
                        onKeyDown={(event) => {
                          if (event.key === 'Enter' || event.key === ' ') {
                            event.preventDefault();
                            setSelectedExecutionId(exec.id);
                          }
                        }}
                        className={`border rounded-md p-3 transition-colors cursor-pointer focus:outline-none ${
                          isSelected
                            ? 'border-primary/60 bg-slate-800/60 ring-1 ring-primary/40'
                            : 'border-border hover:bg-slate-800/40'
                        }`}
                      >
                        <div className="flex items-center justify-between gap-3">
                          <div>
                            <div className="text-sm font-medium">Execution #{idx + 1}</div>
                            <div className="text-xs text-slate-400">
                              Started: {formatDateTime(exec.start_time)}
                              {exec.end_time && <> • Ended: {formatDateTime(exec.end_time)}</>}
                            </div>
                            <div className="text-xs text-slate-500">
                              Duration: {formatExecutionDuration(exec)}{exec.timeout_allowed ? ` • Timeout: ${exec.timeout_allowed}` : ''}
                            </div>
                          </div>
                          <div className={`text-sm font-semibold ${getStatusTone(exec.status)}`}>
                            {exec.status}
                          </div>
                        </div>
                        {exec.exit_code !== undefined && (
                          <div className="mt-2 text-xs text-slate-400">
                            Exit code: {exec.exit_code}
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>

                <div className="rounded-md border border-primary/30 bg-slate-900/60 p-3 min-h-[220px]">
                  {selectedExecution ? (
                    <div className="space-y-3">
                      <div className="text-xs uppercase text-slate-400">Selected execution</div>
                      <div className="text-sm font-medium text-foreground break-words">ID: {selectedExecution.id}</div>
                      <div className="text-xs text-slate-300">
                        Task: {selectedExecution.task_id} • Status:{' '}
                        <span className={getStatusTone(selectedExecution.status)}>
                          {selectedExecution.status}
                        </span>
                      </div>
                      <div className="text-xs text-slate-400">
                        Started: {formatDateTime(selectedExecution.start_time)}
                        {selectedExecution.end_time && (
                          <> • Ended: {formatDateTime(selectedExecution.end_time)}</>
                        )}
                      </div>
                      <div className="text-xs text-slate-400">
                        Duration: {formatExecutionDuration(selectedExecution)}
                        {selectedExecution.timeout_allowed ? ` • Timeout: ${selectedExecution.timeout_allowed}` : ''}
                      </div>
                      {selectedExecution.exit_code !== undefined && (
                        <div className="text-xs text-slate-400">Exit code: {selectedExecution.exit_code}</div>
                      )}
                      {selectedExecution.exit_reason && (
                        <div className="text-xs text-slate-400">Exit reason: {selectedExecution.exit_reason}</div>
                      )}
                      <div className="space-y-1">
                        <div className="text-[11px] uppercase text-slate-400">Final message</div>
                        {isLoadingSelectedOutput ? (
                          <div className="text-xs text-slate-500">Loading output...</div>
                        ) : selectedFinalMessage ? (
                          <div className="rounded-md border border-white/5 bg-slate-800/70 p-2 text-sm text-slate-100 whitespace-pre-wrap max-h-36 overflow-y-auto">
                            {selectedFinalMessage}
                          </div>
                        ) : (
                          <div className="text-xs text-slate-500">No output captured for this execution.</div>
                        )}
                      </div>
                      <div className="space-y-1">
                        <div className="text-[11px] uppercase text-slate-400">Prompt sent to agent</div>
                        {isLoadingSelectedPrompt ? (
                          <div className="text-xs text-slate-500">Loading prompt...</div>
                        ) : selectedPromptText ? (
                          <div className="rounded-md border border-white/5 bg-slate-900/70 p-2 text-xs text-slate-100 whitespace-pre-wrap max-h-40 overflow-y-auto">
                            {selectedPromptText}
                          </div>
                        ) : (
                          <div className="text-xs text-slate-500">Prompt not captured for this execution.</div>
                        )}
                      </div>
                      {selectedExecution.metadata && Object.keys(selectedExecution.metadata).length > 0 ? (
                        <pre className="text-xs text-slate-200 bg-black/30 border border-white/5 rounded p-2 overflow-x-auto">
                          {JSON.stringify(selectedExecution.metadata, null, 2)}
                        </pre>
                      ) : (
                        <div className="text-xs text-slate-500">No additional metadata</div>
                      )}
                    </div>
                  ) : (
                    <div className="flex h-full items-center justify-center text-slate-500 text-sm">
                      Select an execution to view details
                    </div>
                  )}
                </div>
              </div>
            )}
          </TabsContent>
        </Tabs>

        <DialogFooter className="gap-2">
          <Button
            variant="outline"
            onClick={handleArchive}
            disabled={updateTask.isPending}
          >
            <Archive className="h-4 w-4 mr-2" />
            Archive
          </Button>
          <Button
            variant="outline"
            onClick={handleDelete}
            disabled={deleteTask.isPending}
            className="text-red-400 hover:text-red-300 border-red-400/30 hover:border-red-400/50"
          >
            <Trash2 className="h-4 w-4 mr-2" />
            Delete
          </Button>
          <div className="flex-1" />
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={updateTask.isPending || (canEditTarget && !normalizedTarget)}
          >
            {updateTask.isPending ? (
              'Saving...'
            ) : (
              <>
                <Save className="h-4 w-4 mr-2" />
                Save Changes
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
