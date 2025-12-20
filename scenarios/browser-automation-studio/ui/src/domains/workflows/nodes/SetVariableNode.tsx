import { memo, FC } from 'react';
import type { NodeProps } from 'reactflow';
import { Variable, Settings2 } from 'lucide-react';
import { useElementPicker } from '@hooks/useElementPicker';
import { useNodeData } from '@hooks/useNodeData';
import { useUrlInheritance } from '@hooks/useUrlInheritance';
import { useSyncedString, useSyncedNumber, useSyncedBoolean, useSyncedSelect } from '@hooks/useSyncedField';
import BaseNode from './BaseNode';
import { NodeTextField, NodeTextArea, NodeSelectField, NodeCheckbox, NodeUrlField, NodeSelectorField, FieldRow } from './fields';

const SOURCE_OPTIONS = [
  { value: 'static', label: 'Static Value' },
  { value: 'expression', label: 'JavaScript Expression' },
  { value: 'extract', label: 'Extract from Selector' },
];

const VALUE_TYPE_OPTIONS = [
  { value: 'text', label: 'Text' },
  { value: 'number', label: 'Number' },
  { value: 'boolean', label: 'Boolean' },
  { value: 'json', label: 'JSON' },
];

const EXTRACT_TYPE_OPTIONS = [
  { value: 'text', label: 'Text Content' },
  { value: 'attribute', label: 'Attribute' },
  { value: 'value', label: 'Input Value' },
  { value: 'html', label: 'Inner HTML' },
];

function coerceBoolean(value: unknown): boolean {
  if (typeof value === 'boolean') {
    return value;
  }
  if (typeof value === 'string') {
    return value.toLowerCase() === 'true';
  }
  return Boolean(value);
}

const SetVariableNode: FC<NodeProps> = ({ selected, id }) => {
  // URL inheritance for element picker (extract mode)
  const { effectiveUrl } = useUrlInheritance(id);

  // Node data hook for fields (set variable doesn't have a proto type, so all stored in node.data)
  const { getValue, updateData } = useNodeData(id);

  // Element picker binding (data-only mode since this node doesn't use action params)
  const elementPicker = useElementPicker(id, { mode: 'data-only' });

  // UI-specific fields using useSyncedField hooks
  const name = useSyncedString(getValue<string>('name') ?? '', {
    onCommit: (v) => updateData({ name: v || undefined }),
  });
  const sourceType = useSyncedSelect(getValue<string>('sourceType') ?? 'static', {
    onCommit: (v) => updateData({ sourceType: v }),
  });
  const valueType = useSyncedSelect(getValue<string>('valueType') ?? 'text', {
    onCommit: (v) => updateData({ valueType: v }),
  });
  const staticValue = useSyncedString(getValue<string>('value') ?? '', {
    onCommit: (v) => updateData({ value: v || undefined }),
  });
  const expression = useSyncedString(getValue<string>('expression') ?? '', {
    onCommit: (v) => updateData({ expression: v || undefined }),
  });
  const selector = useSyncedString(getValue<string>('selector') ?? '', {
    onCommit: (v) => updateData({ selector: v || undefined }),
  });
  const extractType = useSyncedSelect(getValue<string>('extractType') ?? 'text', {
    onCommit: (v) => updateData({ extractType: v }),
  });
  const attribute = useSyncedString(getValue<string>('attribute') ?? '', {
    onCommit: (v) => updateData({ attribute: v || undefined }),
  });
  const storeAs = useSyncedString(getValue<string>('storeAs') ?? '', {
    onCommit: (v) => updateData({ storeAs: v || undefined }),
  });
  const timeoutMs = useSyncedNumber(getValue<number>('timeoutMs') ?? 0, {
    min: 0,
    onCommit: (v) => updateData({ timeoutMs: v || undefined }),
  });
  const allMatches = useSyncedBoolean(coerceBoolean(getValue<boolean>('allMatches')), {
    onCommit: (v) => updateData({ allMatches: v }),
  });

  const renderSourceFields = () => {
    if (sourceType.value === 'static') {
      return (
        <div className="space-y-2">
          <NodeSelectField field={valueType} label="Value Type" options={VALUE_TYPE_OPTIONS} />
          <NodeTextArea
            field={staticValue}
            label="Value"
            rows={3}
            placeholder={valueType.value === 'json' ? '{"key": "value"}' : 'Value to store'}
          />
        </div>
      );
    }

    if (sourceType.value === 'expression') {
      return (
        <NodeTextArea
          field={expression}
          label="Expression"
          rows={4}
          placeholder="return document.querySelector('#title').textContent;"
          description="Runs inside the page context. Return a value to store in the variable."
        />
      );
    }

    // Extract mode - uses NodeUrlField and NodeSelectorField
    return (
      <div className="space-y-2">
        <NodeUrlField nodeId={id} />
        <NodeSelectorField
          field={selector}
          effectiveUrl={effectiveUrl}
          onElementSelect={elementPicker.onSelect}
          placeholder="CSS selector..."
          className=""
        />
        <NodeSelectField field={extractType} label="Extract Type" options={EXTRACT_TYPE_OPTIONS} />
        {extractType.value === 'attribute' && (
          <NodeTextField field={attribute} label="Attribute Name" placeholder="data-testid" />
        )}
        <FieldRow>
          <NodeCheckbox field={allMatches} label="Capture all matches" />
          <div className="flex items-center gap-1 text-xs text-gray-400">
            <span>Timeout</span>
            <input
              type="number"
              min={0}
              className="w-20 px-2 py-0.5 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              value={timeoutMs.value}
              onChange={(e) => {
                const next = Math.max(0, Number(e.target.value));
                timeoutMs.setValue(next);
              }}
              onBlur={timeoutMs.commit}
            />
          </div>
        </FieldRow>
      </div>
    );
  };

  return (
    <BaseNode selected={selected} icon={Variable} iconClassName="text-emerald-300" title="Set Variable">
      <div className="space-y-2 text-xs">
        <NodeTextField field={name} label="Variable Name" placeholder="variableName" />
        <NodeTextField field={storeAs} label="Store As (optional alias)" placeholder="friendly alias" />
        <div>
          <label className="text-[11px] font-semibold text-gray-400 flex items-center gap-1">
            <Settings2 size={12} /> Source
          </label>
          <select
            className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            value={sourceType.value}
            onChange={(e) => {
              sourceType.setValue(e.target.value);
              sourceType.commit();
            }}
          >
            {SOURCE_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </div>
        {renderSourceFields()}
      </div>
    </BaseNode>
  );
};

export default memo(SetVariableNode);
