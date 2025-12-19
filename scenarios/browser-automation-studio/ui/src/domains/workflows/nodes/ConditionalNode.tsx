import { FC, memo, useMemo } from 'react';
import { Handle, NodeProps, Position } from 'reactflow';
import { GitBranch, ToggleLeft } from 'lucide-react';
import { useNodeData } from '@hooks/useNodeData';
import { useSyncedString, useSyncedNumber, useSyncedBoolean, useSyncedSelect, textInputHandler } from '@hooks/useSyncedField';

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

  // UI-specific fields using useSyncedField hooks
  const conditionType = useSyncedSelect(getValue<string>('conditionType') ?? 'expression', {
    onCommit: (v) => updateData({ conditionType: v }),
  });
  const expression = useSyncedString(getValue<string>('expression') ?? '', {
    onCommit: (v) => updateData({ expression: v?.trim() || undefined }),
  });
  const selector = useSyncedString(getValue<string>('selector') ?? '', {
    onCommit: (v) => updateData({ selector: v?.trim() || undefined }),
  });
  const variableName = useSyncedString(getValue<string>('variable') ?? '', {
    onCommit: (v) => updateData({ variable: v?.trim() || undefined }),
  });
  const operator = useSyncedSelect(getValue<string>('operator') ?? 'equals', {
    onCommit: (v) => updateData({ operator: v }),
  });
  const comparisonValue = useSyncedString(getValue<string>('value') ?? '', {
    onCommit: (v) => updateData({ value: v }),
  });
  const negate = useSyncedBoolean(getValue<boolean>('negate') ?? false, {
    onCommit: (v) => updateData({ negate: v }),
  });
  const timeoutMs = useSyncedNumber(getValue<number>('timeoutMs') ?? DEFAULT_TIMEOUT, {
    min: 100,
    max: 60000,
    fallback: DEFAULT_TIMEOUT,
    onCommit: (v) => updateData({ timeoutMs: v }),
  });
  const pollIntervalMs = useSyncedNumber(getValue<number>('pollIntervalMs') ?? DEFAULT_POLL_INTERVAL, {
    min: 50,
    max: 2000,
    fallback: DEFAULT_POLL_INTERVAL,
    onCommit: (v) => updateData({ pollIntervalMs: v }),
  });

  const showExpressionField = conditionType.value === 'expression';
  const showSelectorField = conditionType.value === 'element';
  const showVariableFields = conditionType.value === 'variable';

  const conditionDescription = useMemo(() => {
    switch (conditionType.value) {
      case 'element':
        return 'Check whether a selector exists on the page (polls until timeout).';
      case 'variable':
        return 'Compare a workflow variable captured earlier in the run.';
      default:
        return 'Evaluate JavaScript inside the page context and branch based on the result.';
    }
  }, [conditionType.value]);

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
          <p className="text-gray-500 mt-1">{conditionDescription}</p>
        </div>

        {showExpressionField && (
          <div>
            <label className="text-gray-400 block mb-1">Expression</label>
            <textarea
              rows={3}
              value={expression.value}
              onChange={(e) => expression.setValue(e.target.value)}
              onBlur={expression.commit}
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
              value={selector.value}
              onChange={textInputHandler(selector.setValue)}
              onBlur={selector.commit}
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
                value={variableName.value}
                onChange={textInputHandler(variableName.setValue)}
                onBlur={variableName.commit}
                placeholder="loginStatus"
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div>
                <label className="text-gray-400 block mb-1">Operator</label>
                <select
                  value={operator.value}
                  onChange={(e) => {
                    operator.setValue(e.target.value);
                    operator.commit();
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
                  value={comparisonValue.value}
                  onChange={textInputHandler(comparisonValue.setValue)}
                  onBlur={comparisonValue.commit}
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
              value={timeoutMs.value}
              onChange={(e) => timeoutMs.setValue(Number(e.target.value))}
              onBlur={timeoutMs.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Poll Interval (ms)</label>
            <input
              type="number"
              min={50}
              max={2000}
              value={pollIntervalMs.value}
              onChange={(e) => pollIntervalMs.setValue(Number(e.target.value))}
              onBlur={pollIntervalMs.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        </div>

        <label className="flex items-center gap-2 text-gray-400">
          <input
            type="checkbox"
            className="accent-flow-accent"
            checked={negate.value}
            onChange={(e) => {
              negate.setValue(e.target.checked);
              negate.commit();
            }}
          />
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
