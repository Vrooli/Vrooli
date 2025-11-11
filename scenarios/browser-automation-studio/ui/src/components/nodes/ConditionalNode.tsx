import { FC, memo, useCallback, useEffect, useMemo, useState } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { GitBranch, ToggleLeft } from 'lucide-react';

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

type ConditionalData = Record<string, unknown>;

const ConditionalNode: FC<NodeProps> = ({ id, data, selected }) => {
  const nodeData = (data ?? {}) as ConditionalData;
  const { getNodes, setNodes } = useReactFlow();

  const [conditionType, setConditionType] = useState(() => String(nodeData.conditionType ?? 'expression'));
  const [expression, setExpression] = useState(() => String(nodeData.expression ?? ''));
  const [selector, setSelector] = useState(() => String(nodeData.selector ?? ''));
  const [variableName, setVariableName] = useState(() => String(nodeData.variable ?? ''));
  const [operator, setOperator] = useState(() => String(nodeData.operator ?? 'equals'));
  const [comparisonValue, setComparisonValue] = useState(() => String(nodeData.value ?? ''));
  const [negate, setNegate] = useState(() => Boolean(nodeData.negate));
  const [timeoutMs, setTimeoutMs] = useState(() => Number(nodeData.timeoutMs ?? DEFAULT_TIMEOUT) || DEFAULT_TIMEOUT);
  const [pollIntervalMs, setPollIntervalMs] = useState(() => Number(nodeData.pollIntervalMs ?? DEFAULT_POLL_INTERVAL) || DEFAULT_POLL_INTERVAL);

  useEffect(() => {
    setConditionType(String(nodeData.conditionType ?? 'expression'));
  }, [nodeData.conditionType]);

  useEffect(() => {
    setExpression(String(nodeData.expression ?? ''));
  }, [nodeData.expression]);

  useEffect(() => {
    setSelector(String(nodeData.selector ?? ''));
  }, [nodeData.selector]);

  useEffect(() => {
    setVariableName(String(nodeData.variable ?? ''));
  }, [nodeData.variable]);

  useEffect(() => {
    setOperator(String(nodeData.operator ?? 'equals'));
  }, [nodeData.operator]);

  useEffect(() => {
    setComparisonValue(String(nodeData.value ?? ''));
  }, [nodeData.value]);

  useEffect(() => {
    setNegate(Boolean(nodeData.negate));
  }, [nodeData.negate]);

  useEffect(() => {
    setTimeoutMs(Number(nodeData.timeoutMs ?? DEFAULT_TIMEOUT) || DEFAULT_TIMEOUT);
  }, [nodeData.timeoutMs]);

  useEffect(() => {
    setPollIntervalMs(Number(nodeData.pollIntervalMs ?? DEFAULT_POLL_INTERVAL) || DEFAULT_POLL_INTERVAL);
  }, [nodeData.pollIntervalMs]);

  const updateNodeData = useCallback((updates: Record<string, unknown>) => {
    const nodes = getNodes();
    setNodes(nodes.map((node) => {
      if (node.id !== id) {
        return node;
      }
      const nextData = { ...(node.data ?? {}) } as ConditionalData;
      Object.entries(updates).forEach(([key, value]) => {
        if (value === undefined || value === null || value === '') {
          delete nextData[key];
        } else {
          nextData[key] = value;
        }
      });
      return { ...node, data: nextData };
    }));
  }, [getNodes, setNodes, id]);

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
    updateNodeData({ timeoutMs: normalized });
  }, [timeoutMs, updateNodeData]);

  const handlePollIntervalCommit = useCallback(() => {
    const normalized = Math.min(Math.max(Math.round(pollIntervalMs) || DEFAULT_POLL_INTERVAL, 50), 2000);
    setPollIntervalMs(normalized);
    updateNodeData({ pollIntervalMs: normalized });
  }, [pollIntervalMs, updateNodeData]);

  const toggleNegate = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const next = event.target.checked;
    setNegate(next);
    updateNodeData({ negate: next });
  }, [updateNodeData]);

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
              updateNodeData({ conditionType: value });
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
              onBlur={() => updateNodeData({ expression: expression.trim() })}
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
              onBlur={() => updateNodeData({ selector: selector.trim() })}
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
                onBlur={() => updateNodeData({ variable: variableName.trim() })}
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
                    updateNodeData({ operator: next });
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
                  onBlur={() => updateNodeData({ value: comparisonValue })}
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
