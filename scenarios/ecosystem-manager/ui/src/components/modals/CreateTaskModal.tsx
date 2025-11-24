/**
 * CreateTaskModal Component
 * Modal for creating new tasks with form validation
 */

import { useEffect, useMemo, useState } from 'react';
import { Plus } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useCreateTask } from '@/hooks/useTaskMutations';
import { useAutoSteerProfiles } from '@/hooks/useAutoSteer';
import { useActiveTargets } from '@/hooks/useActiveTargets';
import { useDiscoveryStore, refreshDiscovery } from '@/stores/discoveryStore';
import type { TaskType, OperationType, Priority, CreateTaskInput, SteerMode } from '@/types/api';
import { STEER_MODES } from '@/types/api';

interface CreateTaskModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const PRIORITIES: Priority[] = ['critical', 'high', 'medium', 'low'];
const AUTO_STEER_NONE = 'none';

export function CreateTaskModal({ open, onOpenChange }: CreateTaskModalProps) {
  const [type, setType] = useState<TaskType>('resource');
  const [operation, setOperation] = useState<OperationType>('generator');
  const [priority, setPriority] = useState<Priority>('medium');
  const [notes, setNotes] = useState('');
  const [autoSteerProfileId, setAutoSteerProfileId] = useState<string>(AUTO_STEER_NONE);
  const [steerMode, setSteerMode] = useState<SteerMode | 'none'>('none');
  const [autoRequeue, setAutoRequeue] = useState(false);
  const [selectedTarget, setSelectedTarget] = useState('');
  const [generatorTarget, setGeneratorTarget] = useState('');

  const createTask = useCreateTask();
  const { data: profiles = [] } = useAutoSteerProfiles();
  const { resources, scenarios, loading: discoveryLoading } = useDiscoveryStore(state => ({
    resources: state.resources,
    scenarios: state.scenarios,
    loading: state.loading,
  }));
  const activeType = open ? type : undefined;
  const { data: activeImproverTargets = [] } = useActiveTargets(activeType, 'improver');
  const { data: activeGeneratorTargets = [] } = useActiveTargets(activeType, 'generator');

  const occupiedTargetSet = useMemo(() => {
    const combined = [...(activeImproverTargets || []), ...(activeGeneratorTargets || [])];
    const set = new Set<string>();
    combined.forEach(t => {
      if (t?.target) {
        set.add(t.target.toLowerCase());
      }
    });
    return set;
  }, [activeGeneratorTargets, activeImproverTargets]);

  const discoveryOptions = useMemo(() => (type === 'resource' ? resources : scenarios), [resources, scenarios, type]);
  const normalizedGeneratorTarget = generatorTarget.trim();
  const normalizedSelectedTarget = selectedTarget.trim();
  const nameSet = useMemo(() => {
    const list = type === 'resource' ? resources : scenarios;
    const set = new Set<string>();
    list.forEach(item => {
      if (item.name) {
        set.add(item.name.toLowerCase());
      }
      if (item.display_name) {
        set.add(item.display_name.toLowerCase());
      }
    });
    return set;
  }, [resources, scenarios, type]);

  const isTargetOccupied = (name: string) => {
    const key = name.trim().toLowerCase();
    return key !== '' && occupiedTargetSet.has(key);
  };

  const generatorConflictsExisting = normalizedGeneratorTarget !== '' && nameSet.has(normalizedGeneratorTarget.toLowerCase());
  const generatorConflictsActive = normalizedGeneratorTarget !== '' && isTargetOccupied(normalizedGeneratorTarget);
  const generatorChecking = discoveryLoading && normalizedGeneratorTarget.length > 0;
  const generatorError = !generatorChecking && (generatorConflictsExisting
    ? 'That name already exists.'
    : generatorConflictsActive
      ? 'An active task already targets this name.'
      : '');
  const generatorSuccess = !generatorChecking && !generatorConflictsExisting && !generatorConflictsActive && normalizedGeneratorTarget.length > 0;
  const targetValue = operation === 'improver' ? normalizedSelectedTarget : normalizedGeneratorTarget;
  const submitDisabled =
    createTask.isPending ||
    targetValue === '' ||
    (operation === 'improver' && isTargetOccupied(targetValue)) ||
    (operation === 'generator' && (generatorConflictsExisting || generatorConflictsActive || generatorChecking));

  // Reset form when modal closes
  useEffect(() => {
    if (!open) {
      setType('resource');
      setOperation('generator');
      setGeneratorTarget('');
      setSelectedTarget('');
      setPriority('medium');
      setNotes('');
      setSteerMode('none');
      setAutoSteerProfileId(AUTO_STEER_NONE);
      setAutoRequeue(false);
    }
  }, [open]);

  useEffect(() => {
    if (operation === 'improver') {
      setGeneratorTarget('');
    } else {
      setSelectedTarget('');
    }
  }, [operation, type]);

  useEffect(() => {
    if (open) {
      refreshDiscovery();
    }
  }, [open, type, operation]);

  // Clear steering choices when task shape changes away from scenario improver
  useEffect(() => {
    if (!(type === 'scenario' && operation === 'improver')) {
      setAutoSteerProfileId(AUTO_STEER_NONE);
      setSteerMode('none');
    }
  }, [type, operation]);

  // Keep steer mode cleared when a profile is selected (guard against stale state on reopen)
  useEffect(() => {
    if (autoSteerProfileId !== AUTO_STEER_NONE && steerMode !== 'none') {
      setSteerMode('none');
    }
  }, [autoSteerProfileId, steerMode]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!targetValue) {
      return;
    }

    if (operation === 'improver' && isTargetOccupied(targetValue)) {
      return;
    }

    if (operation === 'generator' && (generatorConflictsExisting || generatorConflictsActive)) {
      return;
    }

    const taskData: CreateTaskInput = {
      type,
      operation,
      priority,
      notes: notes.trim() || undefined,
      steer_mode: autoSteerProfileId === AUTO_STEER_NONE && steerMode !== 'none' ? steerMode : undefined,
      auto_steer_profile_id: autoSteerProfileId === AUTO_STEER_NONE ? undefined : autoSteerProfileId,
      auto_requeue: autoRequeue,
      target: targetValue ? [targetValue] : undefined,
    };

    createTask.mutate(taskData, {
      onSuccess: () => {
        onOpenChange(false);
      },
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Create New Task</DialogTitle>
          <DialogDescription>
            Configure and submit a new task to the Ecosystem Manager queue
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Type & Operation */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="type">Type</Label>
              <Select value={type} onValueChange={(val: string) => {
                setType(val as TaskType);
                refreshDiscovery();
              }}>
                <SelectTrigger id="type">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="resource">Resource</SelectItem>
                  <SelectItem value="scenario">Scenario</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="operation">Operation</Label>
              <Select value={operation} onValueChange={(val: string) => setOperation(val as OperationType)}>
                <SelectTrigger id="operation">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="generator">Generator</SelectItem>
                  <SelectItem value="improver">Improver</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Target */}
          {operation === 'improver' ? (
            <div className="space-y-2">
              <Label htmlFor="target-select">Target</Label>
              <Select
                value={selectedTarget}
                onValueChange={(val: string) => setSelectedTarget(val)}
                disabled={discoveryLoading}
              >
                <SelectTrigger id="target-select">
                  <SelectValue placeholder={discoveryLoading ? 'Loading targets...' : 'Select target'} />
                </SelectTrigger>
                <SelectContent>
                  {discoveryOptions.map(option => {
                    const optionValue = option.name;
                    const disabled = isTargetOccupied(optionValue);
                    const label =
                      option.display_name && option.display_name !== option.name
                        ? `${option.display_name} (${option.name})`
                        : option.name;
                    return (
                      <SelectItem key={optionValue} value={optionValue} disabled={disabled}>
                        {label} {disabled ? '• in queue' : ''}
                      </SelectItem>
                    );
                  })}
                </SelectContent>
              </Select>
              {discoveryLoading && discoveryOptions.length === 0 && (
                <p className="text-xs text-amber-500">Loading available targets…</p>
              )}
              {selectedTarget && isTargetOccupied(selectedTarget) && (
                <p className="text-xs text-amber-500">An active task already targets this item.</p>
              )}
            </div>
          ) : (
            <div className="space-y-2">
              <Label htmlFor="target-name">Target</Label>
              <Input
                id="target-name"
                value={generatorTarget}
                onChange={(e) => setGeneratorTarget(e.target.value)}
                placeholder={`Enter new ${type} name`}
                required
              />
              {generatorChecking && (
                <p className="text-xs text-amber-500">Checking if name already exists...</p>
              )}
              {generatorError && (
                <p className="text-xs text-red-500">
                  {generatorError}
                </p>
              )}
              {generatorSuccess && (
                <p className="text-xs text-emerald-500">Name is available.</p>
              )}
            </div>
          )}

          {/* Priority */}
          <div className="space-y-2">
            <Label htmlFor="priority">Priority</Label>
            <Select value={priority} onValueChange={(val: string) => setPriority(val as Priority)}>
              <SelectTrigger id="priority">
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

          {type === 'scenario' && operation === 'improver' && (
            <div className="space-y-2">
              <Label htmlFor="steer-mode">Steering focus (optional)</Label>
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
                <SelectTrigger id="steer-mode">
                  <SelectValue placeholder="None (defaults to Progress)" />
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
                Choose a manual focus OR an Auto Steer profile. Selecting a profile disables manual steering until cleared.
              </p>
            </div>
          )}

          {/* Auto Steer Profile */}
          {type === 'scenario' && operation === 'improver' && profiles && profiles.length > 0 && (
            <div className="space-y-2">
              <Label htmlFor="auto-steer">Auto Steer Profile (Optional)</Label>
              <Select
                value={autoSteerProfileId}
                onValueChange={(val: string) => {
                  setAutoSteerProfileId(val);
                  if (val !== AUTO_STEER_NONE) {
                    setSteerMode('none');
                  }
                }}
              >
                <SelectTrigger id="auto-steer">
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

          {/* Notes */}
          <div className="space-y-2">
            <Label htmlFor="notes">Notes (Optional)</Label>
            <Textarea
              id="notes"
              value={notes}
              onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setNotes(e.target.value)}
              placeholder="Additional context or requirements..."
              rows={4}
            />
          </div>

          {/* Auto-Requeue */}
          <div className="flex items-center gap-2">
            <Checkbox
              id="auto-requeue"
              checked={autoRequeue}
              onCheckedChange={(checked: boolean) => setAutoRequeue(checked)}
            />
            <Label htmlFor="auto-requeue" className="cursor-pointer">
              Auto-requeue on completion
            </Label>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={submitDisabled}>
              {createTask.isPending ? (
                'Creating...'
              ) : (
                <>
                  <Plus className="h-4 w-4 mr-2" />
                  Create Task
                </>
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
