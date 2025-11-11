import { FC, memo, useCallback, useEffect, useMemo, useState } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { RefreshCcw } from 'lucide-react';

const LOOP_TYPES = [
  { value: 'foreach', label: 'For Each (array variable)' },
  { value: 'repeat', label: 'Repeat (fixed count)' },
  { value: 'while', label: 'While (boolean condition)' },
];

const CONDITION_TYPES = [
  { value: 'variable', label: 'Variable' },
  { value: 'expression', label: 'Expression (JS)' },
];

const CONDITION_OPERATORS = [
  { value: 'truthy', label: 'Is truthy' },
  { value: 'equals', label: 'Equals' },
  { value: 'not_equals', label: 'Not equals' },
  { value: 'contains', label: 'Contains' },
];

const DEFAULT_MAX_ITERATIONS = 100;

const LoopNode: FC<NodeProps> = ({ id, data, selected }) => {
  const { getNodes, setNodes } = useReactFlow();
  const nodeData = (data ?? {}) as Record<string, unknown>;

  const [loopType, setLoopType] = useState(() => String(nodeData.loopType ?? 'foreach'));
  const [arraySource, setArraySource] = useState(() => String(nodeData.arraySource ?? ''));
  const [repeatCount, setRepeatCount] = useState(() => Number(nodeData.count ?? 1) || 1);
  const [maxIterations, setMaxIterations] = useState(() => Number(nodeData.maxIterations ?? DEFAULT_MAX_ITERATIONS) || DEFAULT_MAX_ITERATIONS);
  const [itemVariable, setItemVariable] = useState(() => String(nodeData.itemVariable ?? 'loop.item'));
  const [indexVariable, setIndexVariable] = useState(() => String(nodeData.indexVariable ?? 'loop.index'));
  const [conditionType, setConditionType] = useState(() => String(nodeData.conditionType ?? 'variable'));
  const [conditionVariable, setConditionVariable] = useState(() => String(nodeData.conditionVariable ?? ''));
  const [conditionOperator, setConditionOperator] = useState(() => String(nodeData.conditionOperator ?? 'truthy'));
  const [conditionValue, setConditionValue] = useState(() => String(nodeData.conditionValue ?? ''));
  const [conditionExpression, setConditionExpression] = useState(() => String(nodeData.conditionExpression ?? ''));

  useEffect(() => setLoopType(String(nodeData.loopType ?? 'foreach')), [nodeData.loopType]);
  useEffect(() => setArraySource(String(nodeData.arraySource ?? '')), [nodeData.arraySource]);
  useEffect(() => setRepeatCount(Number(nodeData.count ?? 1) || 1), [nodeData.count]);
  useEffect(() => setMaxIterations(Number(nodeData.maxIterations ?? DEFAULT_MAX_ITERATIONS) || DEFAULT_MAX_ITERATIONS), [nodeData.maxIterations]);
  useEffect(() => setItemVariable(String(nodeData.itemVariable ?? 'loop.item')), [nodeData.itemVariable]);
  useEffect(() => setIndexVariable(String(nodeData.indexVariable ?? 'loop.index')), [nodeData.indexVariable]);
  useEffect(() => setConditionType(String(nodeData.conditionType ?? 'variable')), [nodeData.conditionType]);
  useEffect(() => setConditionVariable(String(nodeData.conditionVariable ?? '')), [nodeData.conditionVariable]);
  useEffect(() => setConditionOperator(String(nodeData.conditionOperator ?? 'truthy')), [nodeData.conditionOperator]);
  useEffect(() => setConditionValue(String(nodeData.conditionValue ?? '')), [nodeData.conditionValue]);
  useEffect(() => setConditionExpression(String(nodeData.conditionExpression ?? '')), [nodeData.conditionExpression]);

  const updateNode = useCallback((updates: Record<string, unknown>) => {
    const nodes = getNodes();
    setNodes(nodes.map((node) => {
      if (node.id !== id) {
        return node;
      }
      return { ...node, data: { ...(node.data ?? {}), ...updates } };
    }));
  }, [getNodes, setNodes, id]);

  const showArraySource = loopType === 'foreach';
  const showRepeatCount = loopType === 'repeat';
  const showConditionConfig = loopType === 'while';
  const showVariableCondition = showConditionConfig && conditionType === 'variable';
  const showExpressionCondition = showConditionConfig && conditionType === 'expression';

  const loopHint = useMemo(() => {
    switch (loopType) {
      case 'repeat':
        return 'Execute the body a fixed number of times.';
      case 'while':
        return 'Evaluate a variable or expression before each iteration.';
      default:
        return 'Iterate over each element in a workflow variable array.';
    }
  }, [loopType]);

  const commitMaxIterations = useCallback(() => {
    const normalized = Math.max(1, Math.min(1000, Math.round(maxIterations) || DEFAULT_MAX_ITERATIONS));
    setMaxIterations(normalized);
    updateNode({ maxIterations: normalized });
  }, [maxIterations, updateNode]);

  const commitRepeatCount = useCallback(() => {
    const normalized = Math.max(1, Math.round(repeatCount) || 1);
    setRepeatCount(normalized);
    updateNode({ count: normalized });
  }, [repeatCount, updateNode]);

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />
      <Handle type="source" position={Position.Bottom} id="loopBody" className="node-handle" style={{ left: '30%', background: '#38bdf8' }} />
      <Handle type="source" position={Position.Bottom} id="loopAfter" className="node-handle" style={{ left: '70%', background: '#7c3aed' }} />
      <Handle type="target" position={Position.Left} id="loopContinue" className="node-handle" style={{ background: '#22c55e' }} />
      <Handle type="target" position={Position.Right} id="loopBreak" className="node-handle" style={{ background: '#f87171' }} />

      <div className="flex items-center gap-2 mb-3">
        <RefreshCcw size={16} className="text-slate-200" />
        <span className="font-semibold text-sm">Loop</span>
      </div>

      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">Loop Type</label>
          <select
            value={loopType}
            onChange={(event) => {
              const value = event.target.value;
              setLoopType(value);
              updateNode({ loopType: value });
            }}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          >
            {LOOP_TYPES.map((option) => (
              <option key={option.value} value={option.value}>{option.label}</option>
            ))}
          </select>
          <p className="text-gray-500 mt-1">{loopHint}</p>
        </div>

        {showArraySource && (
          <div>
            <label className="text-gray-400 block mb-1">Array Variable</label>
            <input
              type="text"
              value={arraySource}
              onChange={(event) => setArraySource(event.target.value)}
              onBlur={() => updateNode({ arraySource: arraySource.trim() })}
              placeholder="rows"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        {showRepeatCount && (
          <div>
            <label className="text-gray-400 block mb-1">Repeat Count</label>
            <input
              type="number"
              min={1}
              value={repeatCount}
              onChange={(event) => setRepeatCount(Number(event.target.value))}
              onBlur={commitRepeatCount}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        {showConditionConfig && (
          <div className="space-y-2">
            <div>
              <label className="text-gray-400 block mb-1">Condition Type</label>
              <select
                value={conditionType}
                onChange={(event) => {
                  const value = event.target.value;
                  setConditionType(value);
                  updateNode({ conditionType: value });
                }}
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              >
                {CONDITION_TYPES.map((option) => (
                  <option key={option.value} value={option.value}>{option.label}</option>
                ))}
              </select>
            </div>

            {showVariableCondition && (
              <div className="space-y-2">
                <div>
                  <label className="text-gray-400 block mb-1">Variable Name</label>
                  <input
                    type="text"
                    value={conditionVariable}
                    onChange={(event) => setConditionVariable(event.target.value)}
                    onBlur={() => updateNode({ conditionVariable: conditionVariable.trim() })}
                    placeholder="continueProcessing"
                    className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                  />
                </div>
                <div>
                  <label className="text-gray-400 block mb-1">Operator</label>
                  <select
                    value={conditionOperator}
                    onChange={(event) => {
                      const value = event.target.value;
                      setConditionOperator(value);
                      updateNode({ conditionOperator: value });
                    }}
                    className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                  >
                    {CONDITION_OPERATORS.map((option) => (
                      <option key={option.value} value={option.value}>{option.label}</option>
                    ))}
                  </select>
                </div>
                {conditionOperator !== 'truthy' && (
                  <div>
                    <label className="text-gray-400 block mb-1">Expected Value</label>
                    <input
                      type="text"
                      value={conditionValue}
                      onChange={(event) => setConditionValue(event.target.value)}
                      onBlur={() => updateNode({ conditionValue: conditionValue })}
                      className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                    />
                  </div>
                )}
              </div>
            )}

            {showExpressionCondition && (
              <div>
                <label className="text-gray-400 block mb-1">Expression (returns boolean)</label>
                <textarea
                  rows={3}
                  value={conditionExpression}
                  onChange={(event) => setConditionExpression(event.target.value)}
                  onBlur={() => updateNode({ conditionExpression: conditionExpression.trim() })}
                  placeholder="return window.isReady === true;"
                  className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                />
              </div>
            )}
          </div>
        )}

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Max Iterations</label>
            <input
              type="number"
              min={1}
              value={maxIterations}
              onChange={(event) => setMaxIterations(Number(event.target.value))}
              onBlur={commitMaxIterations}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Item Variable</label>
            <input
              type="text"
              value={itemVariable}
              onChange={(event) => setItemVariable(event.target.value)}
              onBlur={() => updateNode({ itemVariable: itemVariable.trim() })}
              placeholder="loop.item"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        </div>

        <div>
          <label className="text-gray-400 block mb-1">Index Variable</label>
          <input
            type="text"
            value={indexVariable}
            onChange={(event) => setIndexVariable(event.target.value)}
            onBlur={() => updateNode({ indexVariable: indexVariable.trim() })}
            placeholder="loop.index"
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>
      </div>
    </div>
  );
};

export default memo(LoopNode);
