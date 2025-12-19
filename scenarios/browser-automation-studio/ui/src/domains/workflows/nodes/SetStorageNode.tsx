import { FC, memo } from 'react';
import type { NodeProps } from 'reactflow';
import { HardDriveUpload } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import {
  useSyncedString,
  useSyncedNumber,
  useSyncedSelect,
  textInputHandler,
  numberInputHandler,
  selectInputHandler,
} from '@hooks/useSyncedField';
import BaseNode from './BaseNode';

// SetStorageParams interface for V2 native action params
interface SetStorageParams {
  storageType?: string;
  key?: string;
  value?: string;
  valueType?: string;
  timeoutMs?: number;
  waitForMs?: number;
}

const STORAGE_OPTIONS = [
  { label: 'localStorage', value: 'localStorage' },
  { label: 'sessionStorage', value: 'sessionStorage' },
];

const VALUE_TYPE_OPTIONS = [
  { label: 'Text', value: 'text' },
  { label: 'JSON', value: 'json' },
];

const MIN_TIMEOUT = 100;

const SetStorageNode: FC<NodeProps> = ({ selected, id }) => {
  const { params, updateParams } = useActionParams<SetStorageParams>(id);

  // Select fields
  const storageType = useSyncedSelect(params?.storageType ?? 'localStorage', {
    onCommit: (v) => updateParams({ storageType: v }),
  });
  const valueType = useSyncedSelect(params?.valueType ?? 'text', {
    onCommit: (v) => updateParams({ valueType: v }),
  });

  // String fields
  const keyValue = useSyncedString(params?.key ?? '', {
    onCommit: (v) => updateParams({ key: v || undefined }),
  });
  const value = useSyncedString(params?.value ?? '', {
    trim: false, // Don't trim storage value
    onCommit: (v) => updateParams({ value: v }),
  });

  // Number fields
  const timeoutMs = useSyncedNumber(params?.timeoutMs ?? 15000, {
    min: MIN_TIMEOUT,
    fallback: 15000,
    onCommit: (v) => updateParams({ timeoutMs: v }),
  });
  const waitForMs = useSyncedNumber(params?.waitForMs ?? 0, {
    min: 0,
    onCommit: (v) => updateParams({ waitForMs: v || undefined }),
  });

  return (
    <BaseNode selected={selected} icon={HardDriveUpload} iconClassName="text-violet-300" title="Set Storage">
      <div className="space-y-3 text-xs">
        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Storage</label>
            <select
              value={storageType.value}
              onChange={selectInputHandler(storageType.setValue, storageType.commit)}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            >
              {STORAGE_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Key</label>
            <input
              type="text"
              value={keyValue.value}
              onChange={textInputHandler(keyValue.setValue)}
              onBlur={keyValue.commit}
              placeholder="profile"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        </div>

        <div>
          <label className="text-gray-400 block mb-1">Value type</label>
          <select
            value={valueType.value}
            onChange={selectInputHandler(valueType.setValue, valueType.commit)}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          >
            {VALUE_TYPE_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="text-gray-400 block mb-1">Value</label>
          <textarea
            rows={3}
            value={value.value}
            onChange={textInputHandler(value.setValue)}
            onBlur={value.commit}
            placeholder={valueType.value === 'json' ? '{"theme":"dark"}' : 'value'}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
          {valueType.value === 'json' && (
            <p className="text-[10px] text-gray-500 mt-1">
              Must be valid JSON and will be stringified before storage.
            </p>
          )}
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Timeout (ms)</label>
            <input
              type="number"
              min={MIN_TIMEOUT}
              value={timeoutMs.value}
              onChange={numberInputHandler(timeoutMs.setValue)}
              onBlur={timeoutMs.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Post-set wait (ms)</label>
            <input
              type="number"
              min={0}
              value={waitForMs.value}
              onChange={numberInputHandler(waitForMs.setValue)}
              onBlur={waitForMs.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        </div>
      </div>
    </BaseNode>
  );
};

export default memo(SetStorageNode);
