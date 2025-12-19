import { FC, memo } from 'react';
import type { NodeProps } from 'reactflow';
import { HardDriveDownload } from 'lucide-react';
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

// GetStorageParams interface for V2 native action params
interface GetStorageParams {
  storageType?: string;
  key?: string;
  resultFormat?: string;
  storeResult?: string;
  timeoutMs?: number;
  waitForMs?: number;
}

const STORAGE_OPTIONS = [
  { label: 'localStorage', value: 'localStorage' },
  { label: 'sessionStorage', value: 'sessionStorage' },
];

const RESULT_OPTIONS = [
  { label: 'Text', value: 'text' },
  { label: 'JSON (parse)', value: 'json' },
];

const MIN_TIMEOUT = 100;

const GetStorageNode: FC<NodeProps> = ({ selected, id }) => {
  const { params, updateParams } = useActionParams<GetStorageParams>(id);

  // Select fields
  const storageType = useSyncedSelect(params?.storageType ?? 'localStorage', {
    onCommit: (v) => updateParams({ storageType: v }),
  });
  const resultFormat = useSyncedSelect(params?.resultFormat ?? 'text', {
    onCommit: (v) => updateParams({ resultFormat: v }),
  });

  // String fields
  const keyValue = useSyncedString(params?.key ?? '', {
    onCommit: (v) => updateParams({ key: v || undefined }),
  });
  const storeAs = useSyncedString(params?.storeResult ?? '', {
    onCommit: (v) => updateParams({ storeResult: v || undefined }),
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
    <BaseNode selected={selected} icon={HardDriveDownload} iconClassName="text-teal-300" title="Get Storage">
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

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Result format</label>
            <select
              value={resultFormat.value}
              onChange={selectInputHandler(resultFormat.setValue, resultFormat.commit)}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            >
              {RESULT_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Store as</label>
            <input
              type="text"
              value={storeAs.value}
              onChange={textInputHandler(storeAs.setValue)}
              onBlur={storeAs.commit}
              placeholder="profileData"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
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
            <label className="text-gray-400 block mb-1">Post-read wait (ms)</label>
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

export default memo(GetStorageNode);
