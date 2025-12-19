import { FC, memo } from 'react';
import type { NodeProps } from 'reactflow';
import { Cookie } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import {
  useSyncedString,
  useSyncedNumber,
  useSyncedBoolean,
  textInputHandler,
  numberInputHandler,
  checkboxInputHandler,
} from '@hooks/useSyncedField';
import BaseNode from './BaseNode';

// ClearCookieParams interface for V2 native action params
interface ClearCookieParams {
  clearAll?: boolean;
  name?: string;
  url?: string;
  domain?: string;
  path?: string;
  timeoutMs?: number;
  waitForMs?: number;
}

const MIN_TIMEOUT = 100;

const ClearCookieNode: FC<NodeProps> = ({ selected, id }) => {
  const { params, updateParams } = useActionParams<ClearCookieParams>(id);

  // Boolean field
  const clearAll = useSyncedBoolean(params?.clearAll ?? false, {
    onCommit: (v) => updateParams({ clearAll: v || undefined }),
  });

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

  const disabledFields = clearAll.value;

  return (
    <BaseNode selected={selected} icon={Cookie} iconClassName="text-rose-300" title="Clear Cookie">
      <div className="space-y-3 text-xs">
        <label className="flex items-center gap-2 text-gray-400">
          <input
            type="checkbox"
            checked={clearAll.value}
            onChange={checkboxInputHandler(clearAll.setValue, clearAll.commit)}
          />
          Remove every cookie in the current context
        </label>

        {!disabledFields && (
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
        )}

        {!disabledFields && (
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
              <label className="text-gray-400 block mb-1">Path</label>
              <input
                type="text"
                value={path.value}
                onChange={textInputHandler(path.setValue)}
                onBlur={path.commit}
                placeholder="/app"
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
            </div>
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

export default memo(ClearCookieNode);
