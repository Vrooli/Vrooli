import { FC, memo, useCallback, useEffect, useMemo, useState } from 'react';
import { Handle, NodeProps, Position } from 'reactflow';
import { GitBranch, ToggleLeft } from 'lucide-react';
import { useNodeData } from '@hooks/useNodeData';

const CONDITION_TYPES = [
  { value: 'expression', label: 'Expression (JS)' },
  { value: 'element', label: 'Element Visibility' },
  { value: 'variable', label: 'Workflow Variable' },
];

const VARIABLE_OPERATORS = [
  { value: 'equals', label: 'Equals' },
  { value: 'not_equals', label: 'Not Equals' },
  { value: 'contains', label: 'Contains' },
  { value: 'starts_with', label: 'Starts With' },
  { value: 'ends_with', label: 'Ends With' },
  { value: 'gt', label: 'Greater Than' },
  { value: 'gte', label: 'Greater Than or Equal' },
  { value: 'lt', label: 'Less Than' },
  { value: 'lte', label: 'Less Than or Equal' },
];

const DEFAULT_TIMEOUT = 10000;
const DEFAULT_POLL_INTERVAL = 250;

const ConditionalNode: FC<NodeProps> = ({ id, selected }) => {
  const { getValue, updateData } = useNodeData(id);

  const [conditionType, setConditionType] = useState<string>(getValue<string>('conditionType') ?? 'expression');
  const [expression, setExpression] = useState<string>(getValue<string>('expression') ?? '');
  const [selector, setSelector] = useState<string>(getValue<string>('selector') ?? '');
  const [variableName, setVariableName] = useState<string>(getValue<string>('variable') ?? '');
  const [operator, setOperator] = useState<string>(getValue<string>('operator') ?? 'equals');
  const [comparisonValue, setComparisonValue] = useState<string>(getValue<string>('value') ?? '');
  const [negate, setNegate] = useState<boolean>(getValue<boolean>('negate') ?? false);
  const [timeoutMs, setTimeoutMs] = useState<number>(getValue<number>('timeoutMs') ?? DEFAULT_TIMEOUT);
  const [pollIntervalMs, setPollIntervalMs] = useState<number>(getValue<number>('pollIntervalMs') ?? DEFAULT_POLL_INTERVAL);

  useEffect(() => {
    setConditionType(getValue<string>('conditionType') ?? 'expression');
  }, [getValue]);

  useEffect(() => {
    setExpression(getValue<string>('expression') ?? '');
  }, [getValue]);

  useEffect(() => {
    setSelector(getValue<string>('selector') ?? '');
  }, [getValue]);

  useEffect(() => {
    setVariableName(getValue<string>('variable') ?? '');
  }, [getValue]);

  useEffect(() => {
    setOperator(getValue<string>('operator') ?? 'equals');
  }, [getValue]);

  useEffect(() => {
    setComparisonValue(getValue<string>('value') ?? '');
  }, [getValue]);

  useEffect(() => {
    setNegate(getValue<boolean>('negate') ?? false);
  }, [getValue]);

  useEffect(() => {
    setTimeoutMs(getValue<number>('timeoutMs') ?? DEFAULT_TIMEOUT);
  }, [getValue]);

  useEffect(() => {
    setPollIntervalMs(getValue<number>('pollIntervalMs') ?? DEFAULT_POLL_INTERVAL);
  }, [getValue]);

  const showExpressionField = conditionType === 'expression';
  const showSelectorField = conditionType === 'element';
  const showVariableFields = conditionType === 'variable';

  const conditionDescription = useMemo(() => {
    switch (conditionType) {
      case 'element':
        return 'Check whether a selector exists on the page (polls until timeout).';
      case 'variable':
        return 'Compare a workflow variable captured earlier in the run.';
      default:
        return 'Evaluate JavaScript inside the page context and branch based on the result.';
    }
  }, [conditionType]);

  const handleTimeoutCommit = useCallback(() => {
    const normalized = Math.min(Math.max(Math.round(timeoutMs) || DEFAULT_TIMEOUT, 100), 60000);
    setTimeoutMs(normalized);
    updateData({ timeoutMs: normalized });
  }, [timeoutMs, updateData]);

  const handlePollIntervalCommit = useCallback(() => {
    const normalized = Math.min(Math.max(Math.round(pollIntervalMs) || DEFAULT_POLL_INTERVAL, 50), 2000);
    setPollIntervalMs(normalized);
    updateData({ pollIntervalMs: normalized });
  }, [pollIntervalMs, updateData]);

  const toggleNegate = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const next = event.target.checked;
    setNegate(next);
    updateData({ negate: next });
  }, [updateData]);

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />
      <div className="flex items-center gap-2 mb-3">
        <GitBranch size={16} className="text-indigo-300" />
        <span className="font-semibold text-sm">Conditional</span>
      </div>

      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">Condition Type</label>
          <select
            value={conditionType}
            onChange={(event) => {
              const value = event.target.value;
              setConditionType(value);
              updateData({ conditionType: value });
            }}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          >
            {CONDITION_TYPES.map((option) => (
              <option key={option.value} value={option.value}>{option.label}</option>
            ))}
          </select>
          <p className="text-gray-500 mt-1">{conditionDescription}</p>
        </div>

        {showExpressionField && (
          <div>
            <label className="text-gray-400 block mb-1">Expression</label>
            <textarea
              rows={3}
              value={expression}
              onChange={(event) => setExpression(event.target.value)}
              onBlur={() => updateData({ expression: expression.trim() })}
              placeholder="return document.querySelector('#status') !== null;"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        {showSelectorField && (
          <div>
            <label className="text-gray-400 block mb-1">Element Selector</label>
            <input
              type="text"
              value={selector}
              onChange={(event) => setSelector(event.target.value)}
              onBlur={() => updateData({ selector: selector.trim() })}
              placeholder="#status-badge"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        {showVariableFields && (
          <div className="space-y-2">
            <div>
              <label className="text-gray-400 block mb-1">Variable Name</label>
              <input
                type="text"
                value={variableName}
                onChange={(event) => setVariableName(event.target.value)}
                onBlur={() => updateData({ variable: variableName.trim() })}
                placeholder="loginStatus"
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div>
                <label className="text-gray-400 block mb-1">Operator</label>
                <select
                  value={operator}
                  onChange={(event) => {
                    const next = event.target.value;
                    setOperator(next);
                    updateData({ operator: next });
                  }}
                  className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                >
                  {VARIABLE_OPERATORS.map((option) => (
                    <option key={option.value} value={option.value}>{option.label}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-gray-400 block mb-1">Compare To</label>
                <input
                  type="text"
                  value={comparisonValue}
                  onChange={(event) => setComparisonValue(event.target.value)}
                  onBlur={() => updateData({ value: comparisonValue })}
                  placeholder="success"
                  className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                />
              </div>
            </div>
          </div>
        )}

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Timeout (ms)</label>
            <input
              type="number"
              min={100}
              max={60000}
              value={timeoutMs}
              onChange={(event) => setTimeoutMs(Number(event.target.value))}
              onBlur={handleTimeoutCommit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Poll Interval (ms)</label>
            <input
              type="number"
              min={50}
              max={2000}
              value={pollIntervalMs}
              onChange={(event) => setPollIntervalMs(Number(event.target.value))}
              onBlur={handlePollIntervalCommit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        </div>

        <label className="flex items-center gap-2 text-gray-400">
          <input type="checkbox" className="accent-flow-accent" checked={negate} onChange={toggleNegate} />
          <span>Negate result (swap true/false)</span>
        </label>

        <div className="p-2 rounded border border-gray-700 bg-flow-bg">
          <div className="flex items-center gap-2 text-gray-300 text-xs">
            <ToggleLeft size={14} className="text-flow-accent" />
            <span>Branch handles</span>
          </div>
          <p className="text-[11px] text-gray-500 mt-1">
            Connect from the green handle for IF TRUE and the red handle for IF FALSE. Workflow Builder will label the edges automatically.
          </p>
        </div>
      </div>

      <div className="relative h-6 mt-3">
        <Handle type="source" position={Position.Bottom} id="ifTrue" className="node-handle" style={{ left: '30%', background: '#4ade80' }} />
        <Handle type="source" position={Position.Bottom} id="ifFalse" className="node-handle" style={{ left: '70%', background: '#f87171' }} />
      </div>
    </div>
  );
};

export default memo(ConditionalNode);
