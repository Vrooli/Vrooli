/**
 * ConditionTree Component
 * Renders the tree of conditions (recursive structure)
 */

import type { StopCondition } from '@/types/api';
import { ConditionNode } from './ConditionNode';

interface ConditionTreeProps {
  conditions: StopCondition[];
  onChange: (conditions: StopCondition[]) => void;
}

export function ConditionTree({ conditions, onChange }: ConditionTreeProps) {
  const handleUpdateCondition = (index: number, updated: StopCondition) => {
    const newConditions = [...conditions];
    newConditions[index] = updated;
    onChange(newConditions);
  };

  const handleRemoveCondition = (index: number) => {
    onChange(conditions.filter((_, i) => i !== index));
  };

  return (
    <div className="space-y-3">
      {conditions.map((condition, index) => (
        <ConditionNode
          key={index}
          condition={condition}
          path={[index]}
          onChange={(updated) => handleUpdateCondition(index, updated)}
          onRemove={() => handleRemoveCondition(index)}
          depth={0}
        />
      ))}
    </div>
  );
}
