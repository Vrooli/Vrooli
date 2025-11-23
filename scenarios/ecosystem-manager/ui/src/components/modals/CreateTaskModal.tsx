/**
 * CreateTaskModal Component
 * Modal for creating new tasks with form validation
 */

import { useState, useEffect } from 'react';
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Checkbox } from '@/components/ui/checkbox';
import { useCreateTask } from '@/hooks/useTaskMutations';
import { useAutoSteerProfiles } from '@/hooks/useAutoSteer';
import { useActiveTargets } from '@/hooks/useActiveTargets';
import type { TaskType, OperationType, Priority, CreateTaskInput } from '@/types/api';

interface CreateTaskModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const PRIORITIES: Priority[] = ['critical', 'high', 'medium', 'low'];
const AUTO_STEER_NONE = 'none';

export function CreateTaskModal({ open, onOpenChange }: CreateTaskModalProps) {
  const [type, setType] = useState<TaskType>('resource');
  const [operation, setOperation] = useState<OperationType>('generator');
  const [title, setTitle] = useState('');
  const [priority, setPriority] = useState<Priority>('medium');
  const [notes, setNotes] = useState('');
  const [autoSteerProfileId, setAutoSteerProfileId] = useState<string>(AUTO_STEER_NONE);
  const [autoRequeue, setAutoRequeue] = useState(false);
  const [selectedTargets, setSelectedTargets] = useState<string[]>([]);

  const createTask = useCreateTask();
  const { data: profiles = [] } = useAutoSteerProfiles();
  const { data: targets = [] } = useActiveTargets(type, operation);

  // Reset form when modal closes
  useEffect(() => {
    if (!open) {
      setType('resource');
      setOperation('generator');
      setTitle('');
      setPriority('medium');
      setNotes('');
      setAutoSteerProfileId(AUTO_STEER_NONE);
      setAutoRequeue(false);
      setSelectedTargets([]);
    }
  }, [open]);

  // Auto-generate title based on type and operation
  useEffect(() => {
    if (!title || title.startsWith('Generate') || title.startsWith('Improve')) {
      if (operation === 'generator') {
        setTitle(`Generate ${type}`);
      } else {
        setTitle(`Improve ${type}`);
      }
    }
  }, [type, operation, title]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!title.trim()) {
      return;
    }

    const taskData: CreateTaskInput = {
      title: title.trim(),
      type,
      operation,
      priority,
      notes: notes.trim() || undefined,
      auto_steer_profile_id: autoSteerProfileId === AUTO_STEER_NONE ? undefined : autoSteerProfileId,
      auto_requeue: autoRequeue,
      target: operation === 'improver' && selectedTargets.length > 0 ? selectedTargets : undefined,
    };

    createTask.mutate(taskData, {
      onSuccess: () => {
        onOpenChange(false);
      },
    });
  };

  const toggleTarget = (target: string) => {
    setSelectedTargets(prev =>
      prev.includes(target)
        ? prev.filter(t => t !== target)
        : [...prev, target]
    );
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
              <Select value={type} onValueChange={(val: string) => setType(val as TaskType)}>
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

          {/* Title */}
          <div className="space-y-2">
            <Label htmlFor="title">Title</Label>
            <Input
              id="title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Task title"
              required
            />
          </div>

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

          {/* Auto Steer Profile */}
          {(profiles && profiles.length > 0) && (
            <div className="space-y-2">
              <Label htmlFor="auto-steer">Auto Steer Profile (Optional)</Label>
              <Select value={autoSteerProfileId} onValueChange={setAutoSteerProfileId}>
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

          {/* Target Multi-Select (Improver only) */}
          {operation === 'improver' && targets && targets.length > 0 && (
            <div className="space-y-2">
              <Label>Targets ({selectedTargets.length} selected)</Label>
              <div className="border rounded-md p-3 max-h-48 overflow-y-auto space-y-2">
                {targets.map(target => (
                  <div key={target} className="flex items-center gap-2">
                    <Checkbox
                      id={`target-${target}`}
                      checked={selectedTargets.includes(target)}
                      onCheckedChange={() => toggleTarget(target)}
                    />
                    <label
                      htmlFor={`target-${target}`}
                      className="text-sm cursor-pointer flex-1"
                    >
                      {target}
                    </label>
                  </div>
                ))}
              </div>
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
            <Button type="submit" disabled={createTask.isPending || !title.trim()}>
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
