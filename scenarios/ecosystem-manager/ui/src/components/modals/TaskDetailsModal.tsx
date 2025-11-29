/**
 * TaskDetailsModal Component
 * Modal for viewing and editing task details with 5 tabs
 */

import { useState, useEffect, useMemo, useRef } from 'react';
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
import { Save, Archive, Trash2, RefreshCw, ChevronsUpDown, AlertCircle, Database, Calendar, Loader2, ExternalLink, RotateCcw } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { useUpdateTask, useDeleteTask } from '@/hooks/useTaskMutations';
import { useAutoSteerProfiles, useAutoSteerExecutionState, useResetAutoSteerExecution, useSeekAutoSteerExecution } from '@/hooks/useAutoSteer';
import { usePhaseNames } from '@/hooks/usePromptFiles';
import { api } from '@/lib/api';
import { markdownToHtml } from '@/lib/markdown';
import { queryKeys } from '@/lib/queryKeys';
import { ExecutionDetailCard } from '@/components/executions/ExecutionDetailCard';
import { SteerFocusBadge, getExecutionSteerFocus } from '@/components/steer/SteerFocusBadge';
import type { SteerFocusInfo } from '@/components/steer/SteerFocusBadge';
import { AutoSteerProfileEditorModal } from '@/components/modals/AutoSteerProfileEditorModal';
import { InsightsTab } from '@/components/insights/InsightsTab';
import type { Task, Priority, ExecutionHistory, UpdateTaskInput, SteerMode, Campaign } from '@/types/api';

interface TaskDetailsModalProps {
  task: Task | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialTab?: 'details' | 'prompt' | 'executions' | 'insights' | 'campaigns';
}

const PRIORITIES: Priority[] = ['critical', 'high', 'medium', 'low'];
const AUTO_STEER_NONE = 'none';

// Campaigns Tab Component
function CampaignsTab({ task }: { task: Task }) {
  const targetPath = Array.isArray(task?.target) && task.target.length > 0 ? task.target[0] : '';

  const { data: rawCampaigns, isLoading, error, refetch } = useQuery({
    queryKey: ['campaigns', targetPath],
    queryFn: () => targetPath ? api.getCampaignsForTarget(targetPath) : Promise.resolve([]),
    enabled: !!targetPath,
    staleTime: 30000,
  });

  // Ensure campaigns is always an array
  const campaigns = Array.isArray(rawCampaigns) ? rawCampaigns : [];

  const handleDelete = async (campaignId: string) => {
    if (!confirm('Delete this campaign? All visit history will be lost.')) return;

    try {
      await api.deleteCampaign(campaignId);
      refetch();
    } catch (err) {
      alert(`Failed to delete campaign: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  const handleReset = async (campaignId: string) => {
    if (!confirm('Reset this campaign? All visit counts and history will be cleared, but the campaign structure will remain.')) return;

    try {
      await api.resetCampaign(campaignId);
      refetch();
    } catch (err) {
      alert(`Failed to reset campaign: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  if (!targetPath) {
    return (
      <div className="text-center py-12 text-slate-400">
        <AlertCircle className="h-12 w-12 mx-auto mb-3 opacity-50" />
        <p className="text-sm">No target path configured for this task.</p>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="text-center py-12 text-slate-400">
        <Loader2 className="h-8 w-8 mx-auto mb-3 animate-spin" />
        <p className="text-sm">Loading campaigns...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center py-12 text-slate-400">
        <AlertCircle className="h-12 w-12 mx-auto mb-3 opacity-50" />
        <p className="text-sm mb-4">Failed to load campaigns</p>
        <Button size="sm" variant="outline" onClick={() => refetch()}>
          <RefreshCw className="h-4 w-4 mr-2" />
          Retry
        </Button>
      </div>
    );
  }

  const handleOpenVisitedTracker = async () => {
    try {
      const { url } = await api.getVisitedTrackerUIPort();
      window.open(url, '_blank');
    } catch (err) {
      alert('Failed to open visited-tracker. Make sure the scenario is running.');
    }
  };

  return (
    <div className="space-y-4">
      {/* Explanation Header */}
      <div className="rounded-md border border-blue-500/30 bg-blue-500/10 p-4 text-sm text-slate-200">
        <div className="flex items-start gap-3">
          <Database className="h-5 w-5 text-blue-400 mt-0.5 flex-shrink-0" />
          <div className="flex-1">
            <h4 className="font-semibold text-slate-100 mb-1">What are campaigns?</h4>
            <p className="text-slate-300">
              Campaigns track which files the AI has visited during improvement loops, ensuring systematic coverage without repetition.
              They store visit counts, staleness scores, and notes to help agents prioritize work across multiple sessions.
            </p>
          </div>
          <Button
            size="sm"
            variant="outline"
            onClick={handleOpenVisitedTracker}
            className="flex-shrink-0"
          >
            <ExternalLink className="h-4 w-4 mr-2" />
            Open Tracker
          </Button>
        </div>
      </div>

      {campaigns.length === 0 ? (
        <div className="text-center py-8 text-slate-400">
          <Database className="h-12 w-12 mx-auto mb-3 opacity-30" />
          <p className="text-sm">No campaigns found for this target.</p>
          <p className="text-xs mt-2 text-slate-500">
            Campaigns are created automatically when agents use visited-tracker CLI during improvement loops.
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          {campaigns.map((campaign) => (
            <div
              key={campaign.id}
              className="rounded-md border border-white/10 bg-slate-900/70 p-4 space-y-3"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="flex-1 min-w-0">
                  <h4 className="font-semibold text-slate-100 truncate">{campaign.name}</h4>
                  {campaign.description && (
                    <p className="text-xs text-slate-400 mt-1">{campaign.description}</p>
                  )}
                  {campaign.tag && (
                    <span className="inline-block mt-2 px-2 py-0.5 rounded text-xs bg-slate-800 text-slate-300">
                      {campaign.tag}
                    </span>
                  )}
                </div>
                <div className="flex gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => handleReset(campaign.id)}
                    className="text-amber-400 hover:text-amber-300 border-amber-400/30 hover:border-amber-400/50"
                  >
                    <RotateCcw className="h-4 w-4 mr-1.5" />
                    Reset
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => handleDelete(campaign.id)}
                    className="text-red-400 hover:text-red-300 border-red-400/30 hover:border-red-400/50"
                  >
                    <Trash2 className="h-4 w-4 mr-1.5" />
                    Delete
                  </Button>
                </div>
              </div>

              <div className="grid grid-cols-3 gap-4 text-xs">
                <div>
                  <div className="text-slate-500">Coverage</div>
                  <div className="font-semibold text-slate-100">
                    {typeof campaign.coverage_percent === 'number'
                      ? `${campaign.coverage_percent.toFixed(1)}%`
                      : '0.0%'}
                  </div>
                </div>
                <div>
                  <div className="text-slate-500">Files Tracked</div>
                  <div className="font-semibold text-slate-100">
                    {campaign.total_files ?? 0}
                  </div>
                </div>
                <div>
                  <div className="text-slate-500">Visited</div>
                  <div className="font-semibold text-slate-100">
                    {campaign.visited_files ?? 0}
                  </div>
                </div>
              </div>

              {campaign.created_at && (
                <div className="flex items-center gap-2 text-xs text-slate-500 pt-2 border-t border-white/5">
                  <Calendar className="h-3 w-3" />
                  <span>Created {new Date(campaign.created_at).toLocaleString()}</span>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export function TaskDetailsModal({ task, open, onOpenChange, initialTab = 'details' }: TaskDetailsModalProps) {
  const [activeTab, setActiveTab] = useState(initialTab);

  // Form state
  const [targetName, setTargetName] = useState('');
  const [priority, setPriority] = useState<Priority>('medium');
  const [notes, setNotes] = useState('');
  const [notesDirty, setNotesDirty] = useState(false);
  const [steerMode, setSteerMode] = useState<SteerMode | 'none'>('none');
  const [autoSteerProfileId, setAutoSteerProfileId] = useState<string>(AUTO_STEER_NONE);
  const [autoRequeue, setAutoRequeue] = useState(false);
  const [selectedExecutionId, setSelectedExecutionId] = useState<string | null>(null);
  const [autoSteerExpanded, setAutoSteerExpanded] = useState(false);
  const [phaseDraft, setPhaseDraft] = useState(0);
  const [iterationDraft, setIterationDraft] = useState(0);
  const [profileModalOpen, setProfileModalOpen] = useState(false);
  const lastTaskIdRef = useRef<string | null>(null);
  const lastSyncedNotesRef = useRef<string>('');

  const updateTask = useUpdateTask();
  const deleteTask = useDeleteTask();
  const { data: profiles = [] } = useAutoSteerProfiles();
  const autoSteerProfilesById = useMemo(
    () => Object.fromEntries((profiles ?? []).map((profile) => [profile.id, profile])),
    [profiles],
  );
  const { data: phaseNames = [], isLoading: phasesLoading } = usePhaseNames();
  const {
    data: autoSteerState,
    isFetching: isAutoSteerStateLoading,
    isError: autoSteerStateError,
    refetch: refetchAutoSteerState,
  } = useAutoSteerExecutionState(task && autoSteerProfileId !== AUTO_STEER_NONE ? task.id : undefined);
  const resetAutoSteer = useResetAutoSteerExecution();
  const seekAutoSteer = useSeekAutoSteerExecution();

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
  const { data: rawExecutions = [], isFetching: isFetchingExecutions, refetch: refetchExecutions } = useQuery({
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
  const canSeekAutoSteer = Boolean(task && autoSteerState);

  // Initialize form when the task identity changes (avoid clobbering in-flight edits)
  useEffect(() => {
    if (!task) return;

    const derivedTarget = (Array.isArray(task.target) && task.target.length > 0)
      ? task.target[0]
      : task.title;
    setTargetName(derivedTarget || '');
    setPriority(task.priority);

    const initialMode: SteerMode | 'none' =
      task.auto_steer_profile_id && task.auto_steer_profile_id !== AUTO_STEER_NONE
        ? 'none'
        : (task.steer_mode as SteerMode) || 'none';
    setSteerMode(initialMode);
    setAutoSteerProfileId(task.auto_steer_profile_id || AUTO_STEER_NONE);
    setAutoRequeue(task.auto_requeue || false);
  }, [task?.id]);

  // Keep notes in sync without clobbering in-progress edits
  useEffect(() => {
    if (!task) return;
    const incomingNotes = task.notes || '';
    const lastTaskId = lastTaskIdRef.current;
    const isNewTask = task.id !== lastTaskId;

    if (isNewTask) {
      setNotes(incomingNotes);
      setNotesDirty(false);
      lastTaskIdRef.current = task.id;
      lastSyncedNotesRef.current = incomingNotes;
      return;
    }

    if (!notesDirty && incomingNotes !== lastSyncedNotesRef.current) {
      setNotes(incomingNotes);
      lastSyncedNotesRef.current = incomingNotes;
    }
  }, [task, notesDirty]);

  useEffect(() => {
    if (task && autoSteerProfileId !== AUTO_STEER_NONE) {
      refetchAutoSteerState();
    }
  }, [task, autoSteerProfileId, refetchAutoSteerState]);

  // Reset when modal closes
  useEffect(() => {
    if (!open) {
      setActiveTab(initialTab);
      setSelectedExecutionId(null);
      setNotesDirty(false);
      lastTaskIdRef.current = null;
      lastSyncedNotesRef.current = '';
      setAutoSteerExpanded(false);
    }
  }, [open, initialTab]);

  // Update active tab when initialTab changes and modal is open
  useEffect(() => {
    if (open) {
      setActiveTab(initialTab);
    }
  }, [initialTab, open]);

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

  const formatExecutionStatusLabel = (status: ExecutionHistory['status']) => {
    switch (status) {
      case 'completed':
        return 'Completed';
      case 'failed':
        return 'Failed';
      case 'rate_limited':
        return 'Rate limited';
      default:
        return 'Running';
    }
  };

  const formatSteerSummary = (focus?: SteerFocusInfo) => {
    if (!focus) return '';
    if (focus.autoSteerProfileName) {
      return focus.phaseMode ? `${focus.autoSteerProfileName} • ${focus.phaseMode}` : focus.autoSteerProfileName;
    }
    return focus.manualSteerMode ?? '';
  };

  const formatExecutionSummary = (execution: ExecutionHistory, focus?: SteerFocusInfo) => {
    const parts = [formatExecutionStatusLabel(execution.status)];
    const duration = formatExecutionDuration(execution);
    if (duration && duration !== '—') {
      parts.push(duration);
    }
    const steerLabel = formatSteerSummary(focus);
    if (steerLabel) {
      parts.push(`Steer: ${steerLabel}`);
    }
    return parts.join(' · ');
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
  const assembledPromptHtml = useMemo(() => markdownToHtml(assembledPrompt), [assembledPrompt]);

  const activeProfile = profiles.find(profile => profile.id === autoSteerProfileId);
  const toNumber = (val: unknown): number =>
    typeof val === 'number' ? val : Number(val ?? 0);

  const currentPhaseNumber = autoSteerState ? toNumber(autoSteerState.current_phase_index) + 1 : 0;
  const totalPhases = activeProfile?.phases?.length ?? 0;
  const currentMode =
    activeProfile?.phases?.[autoSteerState?.current_phase_index ?? 0]?.mode ??
    (activeProfile?.phases?.[0]?.mode ?? undefined);
  const completedPhaseIterations =
    autoSteerState?.phase_history?.reduce((sum, phase) => sum + toNumber(phase?.iterations), 0) ?? 0;
  const currentPhaseIteration = toNumber(autoSteerState?.current_phase_iteration);
  const derivedTotalIterations = completedPhaseIterations + currentPhaseIteration;
  const historyIterationCount = autoSteerState ? sortedExecutions.length : 0;
  const latestAutoSteerIteration = toNumber(latestExecution?.auto_steer_iteration);
  const latestPhaseIteration = toNumber(latestExecution?.steer_phase_iteration);
  const autoSteerIteration = autoSteerState
    ? Math.max(
      toNumber(autoSteerState?.auto_steer_iteration),
      derivedTotalIterations,
      latestAutoSteerIteration,
      historyIterationCount,
    )
    : 0;
  const phaseIteration = autoSteerState
    ? currentPhaseIteration + 1
    : latestPhaseIteration || (historyIterationCount > 0 ? 1 : 0);
  const currentPhaseIndex = autoSteerState?.current_phase_index ?? 0;
  const currentPhaseIterationRaw = autoSteerState?.current_phase_iteration ?? 0;
  const selectedPhase = activeProfile?.phases?.[phaseDraft];
  const selectedPhaseMaxIterations = selectedPhase?.max_iterations ?? 1;

  useEffect(() => {
    if (!autoSteerState) return;
    const clampedPhase = Math.min(
      Math.max(autoSteerState.current_phase_index ?? 0, 0),
      Math.max((activeProfile?.phases?.length ?? 1) - 1, 0)
    );
    setPhaseDraft(clampedPhase);
    setIterationDraft(autoSteerState.current_phase_iteration ?? 0);
  }, [autoSteerState, activeProfile?.phases?.length]);

  useEffect(() => {
    if (!selectedPhase) return;
    if (iterationDraft > selectedPhaseMaxIterations) {
      setIterationDraft(selectedPhaseMaxIterations);
    }
  }, [selectedPhase, selectedPhaseMaxIterations, iterationDraft]);

  if (!task) return null;

  const handleResetAutoSteer = async () => {
    if (!task) return;
    await resetAutoSteer.mutateAsync(task.id);
    await refetchAutoSteerState();
  };

  const handleSave = async () => {
    if (!task) return;
    if (canEditTarget && !normalizedTarget) {
      return;
    }

    const updates: UpdateTaskInput = {
      priority,
      notes: notes.trim(),
      steer_mode: autoSteerProfileId === AUTO_STEER_NONE && steerMode !== 'none' ? steerMode : undefined,
      auto_steer_profile_id: autoSteerProfileId === AUTO_STEER_NONE ? undefined : autoSteerProfileId,
      auto_requeue: autoRequeue,
    };

    if (canEditTarget && normalizedTarget) {
      updates.target = [normalizedTarget];
    }

    const desiredPhaseIteration = Math.min(iterationDraft, selectedPhaseMaxIterations);
    const seekNeeded =
      canSeekAutoSteer &&
      autoSteerState &&
      (phaseDraft !== currentPhaseIndex || desiredPhaseIteration !== currentPhaseIterationRaw);

    try {
      if (seekNeeded) {
        const scenarioName = Array.isArray(normalizedTarget)
          ? normalizedTarget[0]
          : normalizedTarget;
        await seekAutoSteer.mutateAsync({
          taskId: task.id,
          phaseIndex: phaseDraft,
          phaseIteration: desiredPhaseIteration,
          profileId: autoSteerProfileId !== AUTO_STEER_NONE ? autoSteerProfileId : undefined,
          scenarioName: scenarioName || undefined,
        });
        await refetchAutoSteerState();
      }

      await updateTask.mutateAsync({
        id: task.id,
        updates,
      });

      lastSyncedNotesRef.current = updates.notes ?? notes.trim();
      setNotesDirty(false);
      onOpenChange(false);
    } catch (err) {
      console.error('Failed to save task or Auto Steer state', err);
      alert('Unable to save changes right now. Please try again.');
    }
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
    <>
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
            <TabsTrigger value="prompt">Prompt</TabsTrigger>
            <TabsTrigger value="executions">Executions</TabsTrigger>
            <TabsTrigger value="insights">Insights</TabsTrigger>
            <TabsTrigger value="campaigns">Campaigns</TabsTrigger>
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
                    <SelectValue placeholder={phasesLoading ? 'Loading phases...' : 'None (use Progress)'} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">None (use Progress)</SelectItem>
                    {phaseNames.map(phase => (
                      <SelectItem key={phase.name} value={phase.name}>
                        {phase.name.split('-').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ')}
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
                  <div className="rounded-md border border-slate-800 bg-slate-900/50 p-3 text-xs text-slate-200 space-y-3">
                    <div className="flex items-start justify-between gap-3">
                      <div className="space-y-1">
                        <button
                          type="button"
                          onClick={() => activeProfile && setProfileModalOpen(true)}
                          disabled={!activeProfile}
                          className="text-sm font-semibold text-slate-100 hover:text-primary transition-colors disabled:opacity-60 disabled:cursor-not-allowed"
                        >
                          {activeProfile?.name ?? 'Auto Steer profile'}
                        </button>
                        <div className="text-slate-400 flex flex-col sm:flex-row sm:items-center sm:gap-2">
                          <span>
                            Phase {Number.isFinite(currentPhaseNumber) && currentPhaseNumber > 0 ? currentPhaseNumber : '—'}{totalPhases ? ` / ${totalPhases}` : ''}
                          </span>
                          <span className="hidden sm:inline text-slate-500">•</span>
                          <span>
                            Iteration {Number.isFinite(phaseIteration) ? phaseIteration : '—'}{selectedPhaseMaxIterations ? ` / ${selectedPhaseMaxIterations}` : ''}
                          </span>
                          {currentMode && (
                            <>
                              <span className="hidden sm:inline text-slate-500">•</span>
                              <span className="uppercase tracking-wide text-[11px]">{currentMode}</span>
                            </>
                          )}
                        </div>
                      </div>
                      <div className="flex gap-2">
                        <Button
                          size="sm"
                          variant="ghost"
                          className="p-2"
                          onClick={async () => {
                            if (task) {
                              await Promise.allSettled([
                                refetchAutoSteerState(),
                                refetchExecutions?.(),
                              ]);
                            } else {
                              await refetchAutoSteerState();
                            }
                          }}
                          disabled={isAutoSteerStateLoading || isFetchingExecutions}
                          aria-label="Refresh Auto Steer status"
                        >
                          <RefreshCw className="h-4 w-4" />
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          className="p-2"
                          onClick={() => setAutoSteerExpanded((v) => !v)}
                          aria-label={autoSteerExpanded ? 'Collapse Auto Steer details' : 'Expand Auto Steer details'}
                        >
                          <ChevronsUpDown className={`h-4 w-4 transition-transform ${autoSteerExpanded ? 'rotate-180' : ''}`} />
                        </Button>
                      </div>
                    </div>

                    {!autoSteerExpanded && (
                      <div className="grid grid-cols-2 gap-2">
                        <div>
                          <div className="text-slate-400">Overall iterations</div>
                          <div className="font-semibold text-slate-50">{autoSteerIteration}</div>
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
                    )}

                    {autoSteerExpanded && activeProfile && (
                      <div className="space-y-3">
                        <div className="flex flex-wrap gap-2">
                          {activeProfile.phases.map((phase, idx) => {
                            const isActive = idx === phaseDraft;
                            return (
                              <button
                                key={phase.id ?? `${phase.mode}-${idx}`}
                                type="button"
                                className={`
                                  px-3 py-2 rounded-md border text-left transition-all
                                  ${isActive ? 'border-primary bg-primary/10 text-primary' : 'border-white/10 bg-slate-800/40 hover:border-primary/40'}
                                `}
                                onClick={() => {
                                  setPhaseDraft(idx);
                                  setIterationDraft(0);
                                }}
                              >
                                <div className="text-[11px] uppercase text-slate-400">Phase {idx + 1}</div>
                                <div className="text-sm font-semibold">{phase.mode.toUpperCase()}</div>
                                <div className="text-[11px] text-slate-500">
                                  Max iterations: {phase.max_iterations}
                                </div>
                              </button>
                            );
                          })}
                        </div>

                        <div className="space-y-1.5">
                          <div className="flex items-center justify-between text-[11px] text-slate-400">
                            <span>Phase iteration</span>
                            <span>
                              {iterationDraft} / {selectedPhaseMaxIterations}
                            </span>
                          </div>
                          <input
                            type="range"
                            min={0}
                            max={selectedPhaseMaxIterations}
                            value={Math.min(iterationDraft, selectedPhaseMaxIterations)}
                            onChange={(e) => setIterationDraft(Number(e.target.value))}
                            disabled={!canSeekAutoSteer}
                            className="w-full accent-primary"
                          />
                        </div>

                        <div className="flex flex-wrap items-center gap-2">
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={!canSeekAutoSteer || seekAutoSteer.isPending}
                            onClick={async () => {
                              if (!task) return;
                              const scenarioName = Array.isArray(normalizedTarget)
                                ? normalizedTarget[0]
                                : (normalizedTarget || task.target);
                              await seekAutoSteer.mutateAsync({
                                taskId: task.id,
                                phaseIndex: phaseDraft,
                                phaseIteration: Math.min(iterationDraft, selectedPhaseMaxIterations),
                                profileId: autoSteerProfileId !== AUTO_STEER_NONE ? autoSteerProfileId : undefined,
                                scenarioName: scenarioName || undefined,
                              });
                              await Promise.allSettled([
                                refetchAutoSteerState(),
                                refetchExecutions?.(),
                              ]);
                            }}
                          >
                            Jump to selection
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            disabled={resetAutoSteer.isPending || !task}
                            onClick={handleResetAutoSteer}
                          >
                            Reset to start
                          </Button>
                          {autoSteerState?.last_updated && (
                            <span className="text-[11px] text-slate-500">
                              Updated {new Date(autoSteerState.last_updated).toLocaleString()}
                            </span>
                          )}
                        </div>
                      </div>
                    )}

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
                        <div className="text-[11px] uppercase text-slate-500">Output</div>
                        {isLoadingLatestOutput ? (
                          <div className="text-xs text-slate-500">Loading output...</div>
                        ) : latestOutputText ? (
                          <pre className="rounded-md border border-white/5 bg-slate-800/60 p-2 text-xs text-slate-100 whitespace-pre-wrap max-h-52 overflow-y-auto">
                            {latestOutputText}
                          </pre>
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
                  onChange={(e) => {
                    setNotes(e.target.value);
                    setNotesDirty(true);
                  }}
                  className="min-h-[240px]"
                />
              </div>
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
                    const steerFocus = getExecutionSteerFocus(exec, autoSteerProfilesById);
                    const summaryLabel = formatExecutionSummary(exec, steerFocus);
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
                          <div className="flex-1 min-w-0">
                            <div className="text-sm font-medium text-white line-clamp-2">
                              {summaryLabel}
                            </div>
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
                        {(steerFocus.autoSteerProfileName || steerFocus.manualSteerMode) && (
                          <div className="mt-3">
                            <SteerFocusBadge {...steerFocus} />
                          </div>
                        )}
                        {exec.exit_code !== undefined && (
                          <div className="mt-2 text-xs text-slate-400">
                            Exit code: {exec.exit_code}
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>

                <ExecutionDetailCard
                  execution={selectedExecution}
                  promptText={selectedPromptText}
                  outputText={selectedOutputText}
                  isLoadingPrompt={isLoadingSelectedPrompt}
                  isLoadingOutput={isLoadingSelectedOutput}
                />
              </div>
            )}
          </TabsContent>

          {/* Insights Tab */}
          <TabsContent value="insights" className="mt-4">
            <InsightsTab task={task} />
          </TabsContent>

          {/* Campaigns Tab */}
          <TabsContent value="campaigns" className="mt-4">
            <CampaignsTab task={task} />
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

      <AutoSteerProfileEditorModal
        open={profileModalOpen}
        onOpenChange={setProfileModalOpen}
        profile={activeProfile ?? null}
      />
    </>
  );
}
