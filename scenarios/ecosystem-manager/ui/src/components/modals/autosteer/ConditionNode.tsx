/**
 * ConditionNode Component
 * Recursive component for rendering simple or compound conditions
 */

import { X, Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { StopCondition } from '@/types/api';

interface ConditionNodeProps {
  condition: StopCondition;
  path: number[]; // Array of indices representing path in tree
  onChange: (updated: StopCondition) => void;
  onRemove: () => void;
  depth: number;
}

export const AVAILABLE_METRICS = [
  { value: 'phase_loops', label: 'Phase Iterations' },
  { value: 'total_loops', label: 'Total Iterations' },
  { value: 'operational_targets_percentage', label: 'Operational Targets %' },
  { value: 'operational_targets_passing', label: 'Operational Targets Passing' },
  { value: 'operational_targets_total', label: 'Operational Targets Total' },
  { value: 'build_status', label: 'Build Status (1 = passing)' },
  { value: 'unit_test_coverage', label: 'Unit Test Coverage' },
  { value: 'integration_test_coverage', label: 'Integration Test Coverage' },
  { value: 'ui_test_coverage', label: 'UI Test Coverage' },
  { value: 'accessibility_score', label: 'Accessibility Score' },
  { value: 'responsive_breakpoints', label: 'Responsive Breakpoints' },
  { value: 'user_flows_implemented', label: 'User Flows Implemented' },
  { value: 'loading_states_count', label: 'Loading States Count' },
  { value: 'error_handling_coverage', label: 'Error Handling Coverage' },
  { value: 'tidiness_score', label: 'Tidiness Score' },
  { value: 'cyclomatic_complexity_avg', label: 'Cyclomatic Complexity (avg)' },
  { value: 'duplication_percentage', label: 'Duplication %' },
  { value: 'standards_violations', label: 'Standards Violations' },
  { value: 'tech_debt_items', label: 'Tech Debt Items' },
  { value: 'edge_cases_covered', label: 'Edge Cases Covered' },
  { value: 'flaky_tests', label: 'Flaky Tests' },
  { value: 'test_quality_score', label: 'Test Quality Score' },
  { value: 'bundle_size_kb', label: 'Bundle Size (KB)' },
  { value: 'initial_load_time_ms', label: 'Initial Load Time (ms)' },
  { value: 'lcp_ms', label: 'LCP (ms)' },
  { value: 'fid_ms', label: 'FID/Blocking Time (ms)' },
  { value: 'cls_score', label: 'CLS Score' },
  { value: 'vulnerability_count', label: 'Vulnerability Count' },
  { value: 'input_validation_coverage', label: 'Input Validation Coverage' },
  { value: 'auth_implementation_score', label: 'Auth Implementation Score' },
  { value: 'security_scan_score', label: 'Security Scan Score' },
];

const COMPARE_OPERATORS = ['>', '<', '>=', '<=', '==', '!='];

const LOGIC_OPERATORS = [
  { value: 'AND', label: 'AND (all must be true)' },
  { value: 'OR', label: 'OR (any must be true)' },
];

export function ConditionNode({
  condition,
  path,
  onChange,
  onRemove,
  depth,
}: ConditionNodeProps) {
  const indent = depth * 24;

  // Handle simple condition
  if (condition.type === 'simple') {
    return (
      <div
        className="flex items-center gap-2 p-3 bg-slate-800 rounded border border-slate-700"
        style={{ marginLeft: `${indent}px` }}
      >
        <Select
          value={condition.metric}
          onValueChange={(value) => onChange({ ...condition, metric: value })}
        >
          <SelectTrigger className="w-[220px] bg-slate-900/70 border-slate-700 text-slate-100">
            <SelectValue placeholder="Select metric..." className="truncate">
              {AVAILABLE_METRICS.find((m) => m.value === condition.metric)?.label ||
                condition.metric ||
                'Select metric...'}
            </SelectValue>
          </SelectTrigger>
          <SelectContent>
            {AVAILABLE_METRICS.map((metric) => (
              <SelectItem key={metric.value} value={metric.value}>
                {metric.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select
          value={condition.compare_operator}
          onValueChange={(value) => onChange({ ...condition, compare_operator: value })}
        >
          <SelectTrigger className="w-[90px] bg-slate-900/70 border-slate-700 text-slate-100">
            <SelectValue placeholder="Op" className="text-center">
              {condition.compare_operator || 'Op'}
            </SelectValue>
          </SelectTrigger>
          <SelectContent>
            {COMPARE_OPERATORS.map((op) => (
              <SelectItem key={op} value={op}>
                {op}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Input
          type="number"
          className="w-[120px]"
          value={condition.value}
          onChange={(e) =>
            onChange({ ...condition, value: parseFloat(e.target.value) || 0 })
          }
          step="any"
          placeholder="Value"
        />

        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={onRemove}
          className="text-red-400 hover:text-red-300"
        >
          <X className="h-4 w-4" />
        </Button>
      </div>
    );
  }

  // Handle compound condition (group)
  if (condition.type === 'compound') {
    const subConditions = condition.conditions || [];

    const handleUpdateSubCondition = (index: number, updated: StopCondition) => {
      const newSubConditions = [...subConditions];
      newSubConditions[index] = updated;
      onChange({ ...condition, conditions: newSubConditions });
    };

    const handleRemoveSubCondition = (index: number) => {
      const newSubConditions = subConditions.filter((_: StopCondition, i: number) => i !== index);
      onChange({ ...condition, conditions: newSubConditions });
    };

    const handleAddSubCondition = (type: 'simple' | 'compound') => {
      if (type === 'simple') {
        const newCondition: StopCondition = {
          type: 'simple',
          metric: '',
          compare_operator: '>',
          value: 0,
        };
        onChange({ ...condition, conditions: [...subConditions, newCondition] });
      } else {
        const newCondition: StopCondition = {
          type: 'compound',
          operator: 'AND',
          conditions: [],
        };
        onChange({ ...condition, conditions: [...subConditions, newCondition] });
      }
    };

    return (
      <div
        className="border-l-2 border-blue-500 pl-4 space-y-3"
        style={{ marginLeft: `${indent}px` }}
      >
        {/* Group Header */}
        <div className="flex items-center gap-2 p-3 bg-blue-900/20 rounded border border-blue-700">
          <Select
            value={condition.operator}
            onValueChange={(value) =>
              onChange({ ...condition, operator: value as 'AND' | 'OR' })
            }
          >
            <SelectTrigger className="w-[200px] bg-slate-900/70 border-slate-700 text-slate-100">
              <SelectValue placeholder="Logic">
                {LOGIC_OPERATORS.find((op) => op.value === condition.operator)?.label ||
                  condition.operator ||
                  'Logic'}
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              {LOGIC_OPERATORS.map((op) => (
                <SelectItem key={op.value} value={op.value}>
                  {op.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <span className="text-xs text-slate-400 flex-1">
            {subConditions.length} condition{subConditions.length !== 1 ? 's' : ''}
          </span>

          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => handleAddSubCondition('simple')}
          >
            <Plus className="h-3 w-3 mr-1" />
            Condition
          </Button>

          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => handleAddSubCondition('compound')}
          >
            <Plus className="h-3 w-3 mr-1" />
            Group
          </Button>

          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={onRemove}
            className="text-red-400 hover:text-red-300"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>

        {/* Sub-conditions (recursive) */}
        {subConditions.length === 0 ? (
          <div className="text-center py-4 text-sm text-slate-500 border border-dashed border-slate-700 rounded">
            Empty group - add conditions
          </div>
        ) : (
          <div className="space-y-3">
            {subConditions.map((subCondition: StopCondition, index: number) => (
              <ConditionNode
                key={index}
                condition={subCondition}
                path={[...path, index]}
                onChange={(updated) => handleUpdateSubCondition(index, updated)}
                onRemove={() => handleRemoveSubCondition(index)}
                depth={depth + 1}
              />
            ))}
          </div>
        )}
      </div>
    );
  }

  return null;
}
