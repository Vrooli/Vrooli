import { FC, memo, useCallback, useEffect, useMemo, useState } from 'react';
import { Handle, NodeProps, Position } from 'reactflow';
import { RefreshCcw } from 'lucide-react';
import { useNodeData } from '@hooks/useNodeData';

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
const DEFAULT_ITERATION_TIMEOUT_MS = 45000;
const DEFAULT_TOTAL_TIMEOUT_MS = 300000;
const MIN_ITERATION_TIMEOUT_MS = 250;
const MAX_ITERATION_TIMEOUT_MS = 600000;
const MIN_TOTAL_TIMEOUT_MS = 1000;
const MAX_TOTAL_TIMEOUT_MS = 1800000;

const LoopNode: FC<NodeProps> = ({ id, selected }) => {
  const { getValue, updateData } = useNodeData(id);

  const [loopType, setLoopType] = useState<string>(getValue<string>('loopType') ?? 'foreach');
  const [arraySource, setArraySource] = useState<string>(getValue<string>('arraySource') ?? '');
  const [repeatCount, setRepeatCount] = useState<number>(getValue<number>('count') ?? 1);
  const [maxIterations, setMaxIterations] = useState<number>(getValue<number>('maxIterations') ?? DEFAULT_MAX_ITERATIONS);
  const [itemVariable, setItemVariable] = useState<string>(getValue<string>('itemVariable') ?? 'loop.item');
  const [indexVariable, setIndexVariable] = useState<string>(getValue<string>('indexVariable') ?? 'loop.index');
  const [conditionType, setConditionType] = useState<string>(getValue<string>('conditionType') ?? 'variable');
  const [conditionVariable, setConditionVariable] = useState<string>(getValue<string>('conditionVariable') ?? '');
  const [conditionOperator, setConditionOperator] = useState<string>(getValue<string>('conditionOperator') ?? 'truthy');
  const [conditionValue, setConditionValue] = useState<string>(getValue<string>('conditionValue') ?? '');
  const [conditionExpression, setConditionExpression] = useState<string>(getValue<string>('conditionExpression') ?? '');
  const [iterationTimeoutMs, setIterationTimeoutMs] = useState<number>(getValue<number>('iterationTimeoutMs') ?? DEFAULT_ITERATION_TIMEOUT_MS);
  const [totalTimeoutMs, setTotalTimeoutMs] = useState<number>(getValue<number>('totalTimeoutMs') ?? DEFAULT_TOTAL_TIMEOUT_MS);

  useEffect(() => setLoopType(getValue<string>('loopType') ?? 'foreach'), [getValue]);
  useEffect(() => setArraySource(getValue<string>('arraySource') ?? ''), [getValue]);
  useEffect(() => setRepeatCount(getValue<number>('count') ?? 1), [getValue]);
  useEffect(() => setMaxIterations(getValue<number>('maxIterations') ?? DEFAULT_MAX_ITERATIONS), [getValue]);
  useEffect(() => setItemVariable(getValue<string>('itemVariable') ?? 'loop.item'), [getValue]);
  useEffect(() => setIndexVariable(getValue<string>('indexVariable') ?? 'loop.index'), [getValue]);
  useEffect(() => setConditionType(getValue<string>('conditionType') ?? 'variable'), [getValue]);
  useEffect(() => setConditionVariable(getValue<string>('conditionVariable') ?? ''), [getValue]);
  useEffect(() => setConditionOperator(getValue<string>('conditionOperator') ?? 'truthy'), [getValue]);
  useEffect(() => setConditionValue(getValue<string>('conditionValue') ?? ''), [getValue]);
  useEffect(() => setConditionExpression(getValue<string>('conditionExpression') ?? ''), [getValue]);
  useEffect(() => setIterationTimeoutMs(getValue<number>('iterationTimeoutMs') ?? DEFAULT_ITERATION_TIMEOUT_MS), [getValue]);
  useEffect(() => setTotalTimeoutMs(getValue<number>('totalTimeoutMs') ?? DEFAULT_TOTAL_TIMEOUT_MS), [getValue]);

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
    updateData({ maxIterations: normalized });
  }, [maxIterations, updateData]);

  const commitRepeatCount = useCallback(() => {
    const normalized = Math.max(1, Math.round(repeatCount) || 1);
    setRepeatCount(normalized);
    updateData({ count: normalized });
  }, [repeatCount, updateData]);

  const commitIterationTimeout = useCallback(() => {
    const normalized = Math.max(
      MIN_ITERATION_TIMEOUT_MS,
      Math.min(MAX_ITERATION_TIMEOUT_MS, Math.round(iterationTimeoutMs) || DEFAULT_ITERATION_TIMEOUT_MS),
    );
    setIterationTimeoutMs(normalized);
    updateData({ iterationTimeoutMs: normalized });
  }, [iterationTimeoutMs, updateData]);

  const commitTotalTimeout = useCallback(() => {
    const normalized = Math.max(
      MIN_TOTAL_TIMEOUT_MS,
      Math.min(MAX_TOTAL_TIMEOUT_MS, Math.round(totalTimeoutMs) || DEFAULT_TOTAL_TIMEOUT_MS),
    );
    setTotalTimeoutMs(normalized);
    updateData({ totalTimeoutMs: normalized });
  }, [totalTimeoutMs, updateData]);

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
              updateData({ loopType: value });
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
              onBlur={() => updateData({ arraySource: arraySource.trim() })}
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
                  updateData({ conditionType: value });
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
                    onBlur={() => updateData({ conditionVariable: conditionVariable.trim() })}
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
                      updateData({ conditionOperator: value });
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
                      onBlur={() => updateData({ conditionValue: conditionValue })}
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
                  onBlur={() => updateData({ conditionExpression: conditionExpression.trim() })}
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
              onBlur={() => updateData({ itemVariable: itemVariable.trim() })}
              placeholder="loop.item"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Iteration Timeout (ms)</label>
            <input
              type="number"
              min={MIN_ITERATION_TIMEOUT_MS}
              max={MAX_ITERATION_TIMEOUT_MS}
              value={iterationTimeoutMs}
              onChange={(event) => setIterationTimeoutMs(Number(event.target.value))}
              onBlur={commitIterationTimeout}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
            <p className="text-gray-500 mt-1">Fails if one pass exceeds this duration.</p>
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Total Timeout (ms)</label>
            <input
              type="number"
              min={MIN_TOTAL_TIMEOUT_MS}
              max={MAX_TOTAL_TIMEOUT_MS}
              value={totalTimeoutMs}
              onChange={(event) => setTotalTimeoutMs(Number(event.target.value))}
              onBlur={commitTotalTimeout}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
            <p className="text-gray-500 mt-1">Stops the loop if combined time is too high.</p>
          </div>
        </div>

        <div>
          <label className="text-gray-400 block mb-1">Index Variable</label>
          <input
            type="text"
            value={indexVariable}
            onChange={(event) => setIndexVariable(event.target.value)}
            onBlur={() => updateData({ indexVariable: indexVariable.trim() })}
            placeholder="loop.index"
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>
      </div>
    </div>
  );
};

export default memo(LoopNode);
