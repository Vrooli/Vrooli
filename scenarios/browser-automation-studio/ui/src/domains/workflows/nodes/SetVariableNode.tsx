import { memo, FC } from 'react';
import type { NodeProps } from 'reactflow';
import { Variable, Settings2 } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useElementPicker } from '@hooks/useElementPicker';
import { useUrlInheritance } from '@hooks/useUrlInheritance';
import { useSyncedString, useSyncedNumber, useSyncedBoolean, useSyncedSelect } from '@hooks/useSyncedField';
import type { SetVariableParams } from '@utils/actionParams';
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

  // V2 Native: Use action params as source of truth
  const { params, updateParams } = useActionParams<SetVariableParams>(id);

  // Element picker binding
  const elementPicker = useElementPicker(id);

  // Action params fields using useSyncedField hooks
  const name = useSyncedString(params?.name ?? '', {
    onCommit: (v) => updateParams({ name: v || undefined }),
  });
  const sourceType = useSyncedSelect(params?.sourceType ?? 'static', {
    onCommit: (v) => updateParams({ sourceType: v }),
  });
  const valueType = useSyncedSelect(params?.valueType ?? 'text', {
    onCommit: (v) => updateParams({ valueType: v }),
  });
  const staticValue = useSyncedString(params?.value ?? '', {
    onCommit: (v) => updateParams({ value: v || undefined }),
  });
  const expression = useSyncedString(params?.expression ?? '', {
    onCommit: (v) => updateParams({ expression: v || undefined }),
  });
  const selector = useSyncedString(params?.selector ?? '', {
    onCommit: (v) => updateParams({ selector: v || undefined }),
  });
  const extractType = useSyncedSelect(params?.extractType ?? 'text', {
    onCommit: (v) => updateParams({ extractType: v }),
  });
  const attribute = useSyncedString(params?.attribute ?? '', {
    onCommit: (v) => updateParams({ attribute: v || undefined }),
  });
  const storeAs = useSyncedString(params?.storeAs ?? '', {
    onCommit: (v) => updateParams({ storeAs: v || undefined }),
  });
  const timeoutMs = useSyncedNumber(params?.timeoutMs ?? 0, {
    min: 0,
    onCommit: (v) => updateParams({ timeoutMs: v || undefined }),
  });
  const allMatches = useSyncedBoolean(coerceBoolean(params?.allMatches), {
    onCommit: (v) => updateParams({ allMatches: v }),
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
