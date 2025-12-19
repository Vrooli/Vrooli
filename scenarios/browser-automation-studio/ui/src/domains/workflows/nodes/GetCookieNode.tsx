import { FC, memo } from 'react';
import type { NodeProps } from 'reactflow';
import { Cookie } from 'lucide-react';
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

// GetCookieParams interface for V2 native action params
interface GetCookieParams {
  name?: string;
  url?: string;
  domain?: string;
  path?: string;
  resultFormat?: string;
  storeResult?: string;
  timeoutMs?: number;
  waitForMs?: number;
}

const RESULT_OPTIONS = [
  { label: 'Store value only', value: 'value' },
  { label: 'Store full cookie object', value: 'object' },
];

const MIN_TIMEOUT = 100;

const GetCookieNode: FC<NodeProps> = ({ selected, id }) => {
  const { params, updateParams } = useActionParams<GetCookieParams>(id);

  // String fields
  const name = useSyncedString(params?.name ?? '', {
    onCommit: (v) => updateParams({ name: v || undefined }),
  });
  const url = useSyncedString(params?.url ?? '', {
    onCommit: (v) => updateParams({ url: v || undefined }),
  });
  const domain = useSyncedString(params?.domain ?? '', {
    onCommit: (v) => updateParams({ domain: v || undefined }),
  });
  const path = useSyncedString(params?.path ?? '', {
    onCommit: (v) => updateParams({ path: v || undefined }),
  });
  const storeAs = useSyncedString(params?.storeResult ?? '', {
    onCommit: (v) => updateParams({ storeResult: v || undefined }),
  });

  // Select field
  const resultFormat = useSyncedSelect(params?.resultFormat ?? 'value', {
    onCommit: (v) => updateParams({ resultFormat: v }),
  });

  // Number fields
  const timeoutMs = useSyncedNumber(params?.timeoutMs ?? 30000, {
    min: MIN_TIMEOUT,
    fallback: 30000,
    onCommit: (v) => updateParams({ timeoutMs: v }),
  });
  const waitForMs = useSyncedNumber(params?.waitForMs ?? 0, {
    min: 0,
    onCommit: (v) => updateParams({ waitForMs: v || undefined }),
  });

  return (
    <BaseNode selected={selected} icon={Cookie} iconClassName="text-lime-300" title="Get Cookie">
      <div className="space-y-3 text-xs">
        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Cookie name</label>
            <input
              type="text"
              value={name.value}
              onChange={textInputHandler(name.setValue)}
              onBlur={name.commit}
              placeholder="authToken"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Store result as</label>
            <input
              type="text"
              value={storeAs.value}
              onChange={textInputHandler(storeAs.setValue)}
              onBlur={storeAs.commit}
              placeholder="authToken"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Target URL</label>
            <input
              type="text"
              value={url.value}
              onChange={textInputHandler(url.setValue)}
              onBlur={url.commit}
              placeholder="https://example.com"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Domain</label>
            <input
              type="text"
              value={domain.value}
              onChange={textInputHandler(domain.setValue)}
              onBlur={domain.commit}
              placeholder=".example.com"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        </div>

        <div>
          <label className="text-gray-400 block mb-1">Path (optional)</label>
          <input
            type="text"
            value={path.value}
            onChange={textInputHandler(path.setValue)}
            onBlur={path.commit}
            placeholder="/dashboard"
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>

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
            <label className="text-gray-400 block mb-1">Post-fetch wait (ms)</label>
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

export default memo(GetCookieNode);
