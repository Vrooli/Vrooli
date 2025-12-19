import { FC, memo } from 'react';
import type { NodeProps } from 'reactflow';
import { Cookie } from 'lucide-react';
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

// CookieParams interface for V2 native action params
interface CookieParams {
  name?: string;
  value?: string;
  url?: string;
  domain?: string;
  path?: string;
  sameSite?: string;
  secure?: boolean;
  httpOnly?: boolean;
  ttlSeconds?: number;
  expiresAt?: string;
  timeoutMs?: number;
  waitForMs?: number;
}

const SAME_SITE_OPTIONS = [
  { label: 'Browser default', value: '' },
  { label: 'Lax', value: 'lax' },
  { label: 'Strict', value: 'strict' },
  { label: 'None', value: 'none' },
];

const MIN_TIMEOUT = 100;

const SetCookieNode: FC<NodeProps> = ({ selected, id }) => {
  const { params, updateParams } = useActionParams<CookieParams>(id);

  // String fields with trim normalization
  const name = useSyncedString(params?.name ?? '', {
    onCommit: (v) => updateParams({ name: v || undefined }),
  });
  const value = useSyncedString(params?.value ?? '', {
    trim: false, // Don't trim cookie value
    onCommit: (v) => updateParams({ value: v }),
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
  const expiresAt = useSyncedString(params?.expiresAt ?? '', {
    onCommit: (v) => updateParams({ expiresAt: v || undefined }),
  });

  // Select field (commits immediately on change)
  const sameSite = useSyncedSelect(params?.sameSite ?? '', {
    onCommit: (v) => updateParams({ sameSite: v || undefined }),
  });

  // Boolean fields (commit immediately on change)
  const secure = useSyncedBoolean(params?.secure ?? false, {
    onCommit: (v) => updateParams({ secure: v || undefined }),
  });
  const httpOnly = useSyncedBoolean(params?.httpOnly ?? false, {
    onCommit: (v) => updateParams({ httpOnly: v || undefined }),
  });

  // Number fields with validation
  const ttlSeconds = useSyncedNumber(params?.ttlSeconds ?? 0, {
    min: 0,
    onCommit: (v) => updateParams({ ttlSeconds: v || undefined }),
  });
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
    <BaseNode selected={selected} icon={Cookie} iconClassName="text-amber-300" title="Set Cookie">
      <div className="space-y-3 text-xs">
        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Cookie name</label>
            <input
              type="text"
              value={name.value}
              onChange={textInputHandler(name.setValue)}
              onBlur={name.commit}
              placeholder="sessionId"
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
              placeholder="/"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        </div>

        <div>
          <label className="text-gray-400 block mb-1">Value</label>
          <textarea
            rows={2}
            value={value.value}
            onChange={textInputHandler(value.setValue)}
            onBlur={value.commit}
            placeholder="Cookie payload"
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Target URL</label>
            <input
              type="text"
              value={url.value}
              onChange={textInputHandler(url.setValue)}
              onBlur={url.commit}
              placeholder="https://example.com/app"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
            <p className="text-[10px] text-gray-500 mt-1">Provide URL or domain to scope the cookie.</p>
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

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">SameSite</label>
            <select
              value={sameSite.value}
              onChange={selectInputHandler(sameSite.setValue, sameSite.commit)}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            >
              {SAME_SITE_OPTIONS.map((option) => (
                <option key={option.value || 'default'} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>
          <div className="flex items-center gap-4 pt-5">
            <label className="flex items-center gap-2 text-gray-400">
              <input
                type="checkbox"
                checked={secure.value}
                onChange={checkboxInputHandler(secure.setValue, secure.commit)}
              />
              Secure
            </label>
            <label className="flex items-center gap-2 text-gray-400">
              <input
                type="checkbox"
                checked={httpOnly.value}
                onChange={checkboxInputHandler(httpOnly.setValue, httpOnly.commit)}
              />
              HTTP only
            </label>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">TTL (seconds)</label>
            <input
              type="number"
              min={0}
              value={ttlSeconds.value}
              onChange={numberInputHandler(ttlSeconds.setValue)}
              onBlur={ttlSeconds.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Expires at (RFC3339)</label>
            <input
              type="text"
              value={expiresAt.value}
              onChange={textInputHandler(expiresAt.setValue)}
              onBlur={expiresAt.commit}
              placeholder="2025-12-31T23:59:59Z"
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

export default memo(SetCookieNode);
