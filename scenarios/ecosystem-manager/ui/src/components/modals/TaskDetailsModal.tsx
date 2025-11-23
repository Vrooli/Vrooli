/**
 * TaskDetailsModal Component
 * Modal for viewing and editing task details with 5 tabs
 */

import { useState, useEffect } from 'react';
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
import { useAutoSteerProfiles } from '@/hooks/useAutoSteer';
import { api } from '@/lib/api';
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

  const updateTask = useUpdateTask();
  const deleteTask = useDeleteTask();
  const { data: profiles = [] } = useAutoSteerProfiles();

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

  // Reset when modal closes
  useEffect(() => {
    if (!open) {
      setActiveTab('details');
    }
  }, [open]);

  if (!task) return null;

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
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
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
          <TabsContent value="details" className="space-y-4 mt-4">
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

            <div className="space-y-2">
              <Label htmlFor="detail-notes">Notes</Label>
              <Textarea
                id="detail-notes"
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                rows={6}
              />
            </div>

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
            <div className="border rounded-md p-4 max-h-96 overflow-y-auto bg-slate-900 font-mono text-xs whitespace-pre-wrap">
              {assembledPrompt || 'No prompt available'}
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
              <div className="space-y-2">
                {executions.map((exec, idx) => (
                  <div key={exec.id} className="border rounded-md p-3 hover:bg-slate-800/50">
                    <div className="flex items-center justify-between">
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
                ))}
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
