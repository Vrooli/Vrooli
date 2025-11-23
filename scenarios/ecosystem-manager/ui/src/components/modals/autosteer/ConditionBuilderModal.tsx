/**
 * ConditionBuilderModal Component
 * Modal for building complex stop conditions with AND/OR logic
 */

import { Save, X, Plus } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import type { StopCondition } from '@/types/api';
import { ConditionTree } from './ConditionTree';

interface ConditionBuilderModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  conditions: StopCondition[];
  onChange: (conditions: StopCondition[]) => void;
}

export function ConditionBuilderModal({
  open,
  onOpenChange,
  conditions,
  onChange,
}: ConditionBuilderModalProps) {
  const handleAddCondition = (type: 'simple' | 'compound') => {
    if (type === 'simple') {
      const newCondition: StopCondition = {
        type: 'simple',
        metric: '',
        compare_operator: '>',
        value: 0,
      };
      onChange([...conditions, newCondition]);
    } else {
      const newCondition: StopCondition = {
        type: 'compound',
        logic_operator: 'AND',
        conditions: [],
      };
      onChange([...conditions, newCondition]);
    }
  };

  const handleSave = () => {
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Configure Stop Conditions</DialogTitle>
          <DialogDescription>
            Build complex conditions using AND/OR logic groups
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="flex items-start gap-3 rounded-lg border border-blue-600/40 bg-blue-950/40 p-4 text-sm text-slate-100">
            <div className="mt-0.5 h-6 w-6 rounded-full bg-blue-500/20 border border-blue-500/60 flex items-center justify-center text-blue-200 text-xs font-semibold">
              i
            </div>
            <div className="space-y-2">
              <p className="font-semibold text-blue-100">How to use stop conditions</p>
              <ul className="list-disc list-inside space-y-1 text-slate-200">
                <li><strong>Condition</strong>: single metric check (iteration count, success rate, etc.).</li>
                <li><strong>Group</strong>: AND/OR bundle of conditions; nest groups for complex logic.</li>
                <li>Evaluate top-down: a condition/group stops the phase when it becomes true.</li>
                <li>You can reopen this builder from the phase card anytime to adjust.</li>
              </ul>
            </div>
          </div>

          {/* Add Buttons */}
          <div className="flex gap-2">
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => handleAddCondition('simple')}
            >
              <Plus className="h-4 w-4 mr-2" />
              Add Condition
            </Button>
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => handleAddCondition('compound')}
            >
              <Plus className="h-4 w-4 mr-2" />
              Add Group (AND/OR)
            </Button>
          </div>

          {/* Condition Tree */}
          {conditions.length === 0 ? (
            <div className="text-center py-12 border border-dashed border-slate-700 rounded-lg">
              <p className="text-slate-400 mb-2">No conditions yet</p>
              <p className="text-sm text-slate-500">
                Add a condition or group to get started
              </p>
            </div>
          ) : (
            <div className="border border-slate-700 rounded-lg p-4 bg-slate-900/50">
              <ConditionTree conditions={conditions} onChange={onChange} />
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            <X className="h-4 w-4 mr-2" />
            Cancel
          </Button>
          <Button onClick={handleSave}>
            <Save className="h-4 w-4 mr-2" />
            Save Conditions
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
