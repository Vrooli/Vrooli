import { FC, memo } from 'react';
import type { NodeProps } from 'reactflow';
import { Trash2 } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import {
  useSyncedString,
  useSyncedNumber,
  useSyncedBoolean,
  useSyncedSelect,
  textInputHandler,
  numberInputHandler,
  checkboxInputHandler,
  selectInputHandler,
} from '@hooks/useSyncedField';
import BaseNode from './BaseNode';

// ClearStorageParams interface for V2 native action params
interface ClearStorageParams {
  storageType?: string;
  clearAll?: boolean;
  key?: string;
  timeoutMs?: number;
  waitForMs?: number;
}

const STORAGE_OPTIONS = [
  { label: 'localStorage', value: 'localStorage' },
  { label: 'sessionStorage', value: 'sessionStorage' },
];

const MIN_TIMEOUT = 100;

const ClearStorageNode: FC<NodeProps> = ({ selected, id }) => {
  const { params, updateParams } = useActionParams<ClearStorageParams>(id);

  // Select field
  const storageType = useSyncedSelect(params?.storageType ?? 'localStorage', {
    onCommit: (v) => updateParams({ storageType: v }),
  });

  // Boolean field
  const clearAll = useSyncedBoolean(params?.clearAll ?? false, {
    onCommit: (v) => updateParams({ clearAll: v || undefined }),
  });

  // String field
  const keyValue = useSyncedString(params?.key ?? '', {
    onCommit: (v) => updateParams({ key: v || undefined }),
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
    <BaseNode selected={selected} icon={Trash2} iconClassName="text-red-300" title="Clear Storage">
      <div className="space-y-3 text-xs">
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

        <label className="flex items-center gap-2 text-gray-400">
          <input
            type="checkbox"
            checked={clearAll.value}
            onChange={checkboxInputHandler(clearAll.setValue, clearAll.commit)}
          />
          Remove every entry
        </label>

        {!clearAll.value && (
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
        )}

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
            <label className="text-gray-400 block mb-1">Post-clear wait (ms)</label>
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

export default memo(ClearStorageNode);
