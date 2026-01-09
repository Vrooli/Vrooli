import { memo, FC, useCallback, useEffect } from 'react';
import type { NodeProps } from 'reactflow';
import { ListFilter } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import { useResiliencePanelProps } from '@hooks/useResiliencePanel';
import {
  useSyncedString,
  useSyncedNumber,
  useSyncedBoolean,
  useSyncedSelect,
  useSyncedArrayAsText,
  textInputHandler,
} from '@hooks/useSyncedField';
import type { SelectParams } from '@utils/actionBuilder';
import BaseNode from './BaseNode';
import ResiliencePanel from './ResiliencePanel';
import { NodeTextField, NodeNumberField, FieldRow, TimeoutFields } from './fields';

const SELECT_BY_OPTIONS = [
  { value: 'value', label: 'Option value' },
  { value: 'text', label: 'Displayed text' },
  { value: 'index', label: 'Option index' },
];

const SelectNode: FC<NodeProps> = ({ selected, id }) => {
  // Node data hook for UI-specific fields
  const { getValue, updateData } = useNodeData(id);

  // V2 Native: Use action params as source of truth
  const { params, updateParams } = useActionParams<SelectParams>(id);

  // Action params fields
  const selector = useSyncedString(params?.selector ?? '', {
    onCommit: (v) => updateParams({ selector: v || undefined }),
  });
  const optionValue = useSyncedString(params?.value ?? '', {
    onCommit: (v) => updateParams({ value: v || undefined }),
  });
  const label = useSyncedString(params?.label ?? '', {
    onCommit: (v) => updateParams({ label: v || undefined }),
  });
  const index = useSyncedNumber(params?.index ?? 0, {
    min: 0,
    onCommit: (v) => updateParams({ index: v }),
  });
  const timeoutMs = useSyncedNumber(params?.timeoutMs ?? 5000, {
    min: 100,
    fallback: 5000,
    onCommit: (v) => updateParams({ timeoutMs: v }),
  });

  // UI-specific fields (stored in node.data)
  const selectBy = useSyncedSelect(getValue<string>('selectBy') ?? 'value', {
    onCommit: (v) => updateData({ selectBy: v }),
  });
  const multiple = useSyncedBoolean(getValue<boolean>('multiple') ?? false, {
    onCommit: (v) => updateData({ multiple: v }),
  });
  const waitForMs = useSyncedNumber(getValue<number>('waitForMs') ?? 0, {
    min: 0,
    onCommit: (v) => updateData({ waitForMs: v || undefined }),
  });

  // Array field using new hook (replaces manual useState + useEffect + parse/format)
  const valuesField = useSyncedArrayAsText(getValue<string[]>('values') ?? [], {
    onCommit: (v) => updateData({ values: v.length > 0 ? v : undefined }),
  });

  const resilience = useResiliencePanelProps(id);

  // If multiple mode and index is selected, switch to value
  useEffect(() => {
    if (multiple.value && selectBy.value === 'index') {
      selectBy.setValue('value');
      updateData({ selectBy: 'value' });
    }
  }, [multiple.value, selectBy, updateData]);

  const renderSelectionField = () => {
    if (multiple.value) {
      // Textarea for multiple values using useSyncedArrayAsText
      return (
        <div>
          <label className="text-gray-400 block mb-1 text-xs">Values (one per line)</label>
          <textarea
            rows={3}
            value={valuesField.value}
            onChange={textInputHandler(valuesField.setValue)}
            onBlur={valuesField.commit}
            placeholder={selectBy.value === 'text' ? 'Primary\nSecondary' : 'value-a\nvalue-b'}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 text-xs focus:border-flow-accent focus:outline-none resize-none"
          />
          <p className="text-[10px] text-gray-500 mt-1">All matching options will be selected.</p>
        </div>
      );
    }

    if (selectBy.value === 'index') {
      return (
        <NodeNumberField
          field={index}
          label="Option index"
          min={0}
          description="Uses zero-based indexing (0 = first option)."
        />
      );
    }

    if (selectBy.value === 'text') {
      return (
        <NodeTextField
          field={label}
          label="Text to match"
          placeholder="Contains..."
          description="Matches options whose text includes this value."
        />
      );
    }

    return (
      <NodeTextField
        field={optionValue}
        label="Option value"
        placeholder="Option value attribute"
      />
    );
  };

  const toggleMultiple = useCallback(() => {
    const next = !multiple.value;
    multiple.setValue(next);
    updateData({ multiple: next });
  }, [multiple, updateData]);

  return (
    <BaseNode selected={selected} icon={ListFilter} iconClassName="text-teal-300" title="Select Option">
      <div className="space-y-3 text-xs">
        <NodeTextField
          field={selector}
          label="Target selector"
          placeholder='select[name="country"]'
        />

        <FieldRow>
          {/* Custom select since it needs disabled option based on multiple state */}
          <div className="flex-1 min-w-0">
            <label className="text-gray-400 block mb-1 text-xs">Select by</label>
            <select
              value={selectBy.value}
              onChange={(e) => {
                selectBy.setValue(e.target.value);
                selectBy.commit();
              }}
              className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
            >
              {SELECT_BY_OPTIONS.map((option) => (
                <option
                  key={option.value}
                  value={option.value}
                  disabled={option.value === 'index' && multiple.value}
                >
                  {option.label}
                </option>
              ))}
            </select>
          </div>
          <div className="flex-1 min-w-0">
            <label className="text-gray-400 block mb-1 text-xs">Allow multiple</label>
            <button
              type="button"
              onClick={toggleMultiple}
              className={`w-full px-2 py-1 rounded border text-xs transition-colors ${
                multiple.value
                  ? 'border-teal-400 text-teal-200 bg-teal-400/10'
                  : 'border-gray-700 text-gray-300 hover:border-gray-500'
              }`}
            >
              {multiple.value ? 'Enabled' : 'Disabled'}
            </button>
          </div>
        </FieldRow>

        {renderSelectionField()}

        <TimeoutFields
          timeoutMs={timeoutMs}
          waitForMs={waitForMs}
          waitLabel="Post-change wait (ms)"
        />

        <p className="text-[11px] text-gray-500">
          Dispatches change/input events after updating the element so downstream nodes can rely on
          the new value.
        </p>
      </div>

      <ResiliencePanel {...resilience} />
    </BaseNode>
  );
};

export default memo(SelectNode);
