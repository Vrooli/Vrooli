import { FC, memo, useMemo } from 'react';
import { Handle, NodeProps, Position } from 'reactflow';
import { RefreshCcw } from 'lucide-react';
import { useNodeData } from '@hooks/useNodeData';
import { useSyncedString, useSyncedNumber, useSyncedSelect, textInputHandler } from '@hooks/useSyncedField';

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

  // UI-specific fields using useSyncedField hooks
  const loopType = useSyncedSelect(getValue<string>('loopType') ?? 'foreach', {
    onCommit: (v) => updateData({ loopType: v }),
  });
  const arraySource = useSyncedString(getValue<string>('arraySource') ?? '', {
    onCommit: (v) => updateData({ arraySource: v?.trim() || undefined }),
  });
  const repeatCount = useSyncedNumber(getValue<number>('count') ?? 1, {
    min: 1,
    fallback: 1,
    onCommit: (v) => updateData({ count: v }),
  });
  const maxIterations = useSyncedNumber(getValue<number>('maxIterations') ?? DEFAULT_MAX_ITERATIONS, {
    min: 1,
    max: 1000,
    fallback: DEFAULT_MAX_ITERATIONS,
    onCommit: (v) => updateData({ maxIterations: v }),
  });
  const itemVariable = useSyncedString(getValue<string>('itemVariable') ?? 'loop.item', {
    onCommit: (v) => updateData({ itemVariable: v?.trim() || undefined }),
  });
  const indexVariable = useSyncedString(getValue<string>('indexVariable') ?? 'loop.index', {
    onCommit: (v) => updateData({ indexVariable: v?.trim() || undefined }),
  });
  const conditionType = useSyncedSelect(getValue<string>('conditionType') ?? 'variable', {
    onCommit: (v) => updateData({ conditionType: v }),
  });
  const conditionVariable = useSyncedString(getValue<string>('conditionVariable') ?? '', {
    onCommit: (v) => updateData({ conditionVariable: v?.trim() || undefined }),
  });
  const conditionOperator = useSyncedSelect(getValue<string>('conditionOperator') ?? 'truthy', {
    onCommit: (v) => updateData({ conditionOperator: v }),
  });
  const conditionValue = useSyncedString(getValue<string>('conditionValue') ?? '', {
    onCommit: (v) => updateData({ conditionValue: v }),
  });
  const conditionExpression = useSyncedString(getValue<string>('conditionExpression') ?? '', {
    onCommit: (v) => updateData({ conditionExpression: v?.trim() || undefined }),
  });
  const iterationTimeoutMs = useSyncedNumber(getValue<number>('iterationTimeoutMs') ?? DEFAULT_ITERATION_TIMEOUT_MS, {
    min: MIN_ITERATION_TIMEOUT_MS,
    max: MAX_ITERATION_TIMEOUT_MS,
    fallback: DEFAULT_ITERATION_TIMEOUT_MS,
    onCommit: (v) => updateData({ iterationTimeoutMs: v }),
  });
  const totalTimeoutMs = useSyncedNumber(getValue<number>('totalTimeoutMs') ?? DEFAULT_TOTAL_TIMEOUT_MS, {
    min: MIN_TOTAL_TIMEOUT_MS,
    max: MAX_TOTAL_TIMEOUT_MS,
    fallback: DEFAULT_TOTAL_TIMEOUT_MS,
    onCommit: (v) => updateData({ totalTimeoutMs: v }),
  });

  const showArraySource = loopType.value === 'foreach';
  const showRepeatCount = loopType.value === 'repeat';
  const showConditionConfig = loopType.value === 'while';
  const showVariableCondition = showConditionConfig && conditionType.value === 'variable';
  const showExpressionCondition = showConditionConfig && conditionType.value === 'expression';

  const loopHint = useMemo(() => {
    switch (loopType.value) {
      case 'repeat':
        return 'Execute the body a fixed number of times.';
      case 'while':
        return 'Evaluate a variable or expression before each iteration.';
      default:
        return 'Iterate over each element in a workflow variable array.';
    }
  }, [loopType.value]);

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
            value={loopType.value}
            onChange={(e) => {
              loopType.setValue(e.target.value);
              loopType.commit();
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
              value={arraySource.value}
              onChange={textInputHandler(arraySource.setValue)}
              onBlur={arraySource.commit}
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
              value={repeatCount.value}
              onChange={(e) => repeatCount.setValue(Number(e.target.value))}
              onBlur={repeatCount.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        {showConditionConfig && (
          <div className="space-y-2">
            <div>
              <label className="text-gray-400 block mb-1">Condition Type</label>
              <select
                value={conditionType.value}
                onChange={(e) => {
                  conditionType.setValue(e.target.value);
                  conditionType.commit();
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
                    value={conditionVariable.value}
                    onChange={textInputHandler(conditionVariable.setValue)}
                    onBlur={conditionVariable.commit}
                    placeholder="continueProcessing"
                    className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                  />
                </div>
                <div>
                  <label className="text-gray-400 block mb-1">Operator</label>
                  <select
                    value={conditionOperator.value}
                    onChange={(e) => {
                      conditionOperator.setValue(e.target.value);
                      conditionOperator.commit();
                    }}
                    className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                  >
                    {CONDITION_OPERATORS.map((option) => (
                      <option key={option.value} value={option.value}>{option.label}</option>
                    ))}
                  </select>
                </div>
                {conditionOperator.value !== 'truthy' && (
                  <div>
                    <label className="text-gray-400 block mb-1">Expected Value</label>
                    <input
                      type="text"
                      value={conditionValue.value}
                      onChange={textInputHandler(conditionValue.setValue)}
                      onBlur={conditionValue.commit}
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
                  value={conditionExpression.value}
                  onChange={(e) => conditionExpression.setValue(e.target.value)}
                  onBlur={conditionExpression.commit}
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
              value={maxIterations.value}
              onChange={(e) => maxIterations.setValue(Number(e.target.value))}
              onBlur={maxIterations.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Item Variable</label>
            <input
              type="text"
              value={itemVariable.value}
              onChange={textInputHandler(itemVariable.setValue)}
              onBlur={itemVariable.commit}
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
              value={iterationTimeoutMs.value}
              onChange={(e) => iterationTimeoutMs.setValue(Number(e.target.value))}
              onBlur={iterationTimeoutMs.commit}
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
              value={totalTimeoutMs.value}
              onChange={(e) => totalTimeoutMs.setValue(Number(e.target.value))}
              onBlur={totalTimeoutMs.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
            <p className="text-gray-500 mt-1">Stops the loop if combined time is too high.</p>
          </div>
        </div>

        <div>
          <label className="text-gray-400 block mb-1">Index Variable</label>
          <input
            type="text"
            value={indexVariable.value}
            onChange={textInputHandler(indexVariable.setValue)}
            onBlur={indexVariable.commit}
            placeholder="loop.index"
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>
      </div>
    </div>
  );
};

export default memo(LoopNode);
