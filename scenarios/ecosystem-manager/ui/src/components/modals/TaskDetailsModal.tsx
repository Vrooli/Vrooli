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
import type { Task, Priority } from '@/types/api';

interface TaskDetailsModalProps {
  task: Task | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const PRIORITIES: Priority[] = ['critical', 'high', 'medium', 'low'];
const AUTO_STEER_NONE = 'none';

export function TaskDetailsModal({ task, open, onOpenChange }: TaskDetailsModalProps) {
  const [activeTab, setActiveTab] = useState('details');

  // Form state
  const [title, setTitle] = useState('');
  const [priority, setPriority] = useState<Priority>('medium');
  const [notes, setNotes] = useState('');
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
  const logs = Array.isArray(rawLogs) ? rawLogs : (rawLogs as any)?.entries ?? [];

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
  const { data: rawExecutions = [] } = useQuery({
    queryKey: queryKeys.tasks.executions(task?.id ?? ''),
    queryFn: () => api.getExecutionHistory(task!.id),
    enabled: !!task && activeTab === 'executions',
  });
  const executions = Array.isArray(rawExecutions)
    ? rawExecutions
    : (rawExecutions as any)?.executions ?? [];
  const selectedExecution = executions.find(exec => exec.id === selectedExecutionId) || null;

  // Initialize form when task changes
  useEffect(() => {
    if (task) {
      setTitle(task.title);
      setPriority(task.priority);
      setNotes(task.notes || '');
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
    if (executions.length > 0 && !selectedExecutionId) {
      const firstId = executions.find(exec => exec.id)?.id ?? executions[0]?.id ?? null;
      setSelectedExecutionId(firstId);
    } else if (selectedExecutionId && !executions.some(exec => exec.id === selectedExecutionId)) {
      const fallbackId = executions.find(exec => exec.id)?.id ?? executions[0]?.id ?? null;
      setSelectedExecutionId(fallbackId);
    }
  }, [executions, selectedExecutionId]);

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
    updateTask.mutate({
      id: task.id,
      updates: {
        title: title.trim(),
        priority,
        notes: notes.trim() || undefined,
        auto_steer_profile_id: autoSteerProfileId === AUTO_STEER_NONE ? undefined : autoSteerProfileId,
        auto_requeue: autoRequeue,
      },
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
                  <Label htmlFor="detail-title">Title</Label>
                  <Input
                    id="detail-title"
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                  />
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

                {task.target && task.target.length > 0 && (
                  <div className="space-y-2">
                    <Label>Targets</Label>
                    <div className="border rounded-md p-3 max-h-48 overflow-y-auto">
                      {task.target.map((t, idx) => (
                        <div key={idx} className="text-sm py-1">
                          • {t}
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {(profiles && profiles.length > 0) && (
                  <div className="space-y-2">
                    <Label htmlFor="detail-auto-steer">Auto Steer Profile</Label>
                    <Select value={autoSteerProfileId} onValueChange={setAutoSteerProfileId}>
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
                    <span className="font-medium">Created:</span> {new Date(task.created_at).toLocaleString()}
                  </div>
                  <div>
                    <span className="font-medium">Updated:</span> {new Date(task.updated_at).toLocaleString()}
                  </div>
                </div>
              </div>

              <div className="space-y-2">
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
            {executions.length === 0 ? (
              <div className="text-slate-400 text-center py-8">No executions yet</div>
            ) : (
              <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)]">
                <div className="space-y-3 max-h-96 overflow-y-auto pr-1">
                  {executions.map((exec, idx) => {
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
                              Started: {new Date(exec.start_time).toLocaleString()}
                              {exec.end_time && (
                                <> • Ended: {new Date(exec.end_time).toLocaleString()}</>
                              )}
                            </div>
                          </div>
                          <div className={`text-sm font-semibold ${
                            exec.status === 'completed' ? 'text-green-400' :
                            exec.status === 'failed' ? 'text-red-400' :
                            'text-yellow-400'
                          }`}>
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
                    <div className="space-y-2">
                      <div className="text-xs uppercase text-slate-400">Selected execution</div>
                      <div className="text-sm font-medium text-foreground break-words">ID: {selectedExecution.id}</div>
                      <div className="text-xs text-slate-300">
                        Task: {selectedExecution.task_id} • Status: {selectedExecution.status}
                      </div>
                      <div className="text-xs text-slate-400">
                        Started: {new Date(selectedExecution.start_time).toLocaleString()}
                        {selectedExecution.end_time && (
                          <> • Ended: {new Date(selectedExecution.end_time).toLocaleString()}</>
                        )}
                      </div>
                      {selectedExecution.exit_code !== undefined && (
                        <div className="text-xs text-slate-400">Exit code: {selectedExecution.exit_code}</div>
                      )}
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
            disabled={updateTask.isPending || !title.trim()}
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
