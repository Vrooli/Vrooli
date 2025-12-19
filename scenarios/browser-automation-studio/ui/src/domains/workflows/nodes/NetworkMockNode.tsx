import { FC, memo, useMemo } from 'react';
import type { NodeProps } from 'reactflow';
import { Network } from 'lucide-react';
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

// NetworkMockParams interface for V2 native action params
interface NetworkMockParams {
  urlPattern?: string;
  method?: string;
  mockType?: string;
  statusCode?: number;
  delayMs?: number;
  headers?: Record<string, string>;
  body?: string;
  abortReason?: string;
}

const MOCK_TYPES = [
  { label: 'Stub response', value: 'response' },
  { label: 'Abort request', value: 'abort' },
  { label: 'Delay & passthrough', value: 'delay' },
];

const METHODS = ['Any', 'GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS', 'HEAD'];

const ABORT_REASONS = [
  { label: 'Generic failure', value: 'Failed' },
  { label: 'Client blocked', value: 'BlockedByClient' },
  { label: 'Server blocked', value: 'BlockedByResponse' },
  { label: 'Timed out', value: 'TimedOut' },
  { label: 'Aborted', value: 'Aborted' },
  { label: 'Connection refused', value: 'ConnectionRefused' },
];

const headersToText = (headers?: Record<string, string>): string => {
  if (!headers || typeof headers !== 'object') {
    return '';
  }
  return Object.entries(headers)
    .map(([key, value]) => `${key}: ${value ?? ''}`.trim())
    .join('\n');
};

const textToHeaders = (text: string): Record<string, string> | undefined => {
  const lines = text.split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
  if (lines.length === 0) {
    return undefined;
  }
  const headers: Record<string, string> = {};
  lines.forEach((line) => {
    const [key, ...rest] = line.split(':');
    const headerKey = key.trim();
    if (!headerKey) {
      return;
    }
    headers[headerKey] = rest.join(':').trim();
  });
  return Object.keys(headers).length ? headers : undefined;
};

const NetworkMockNode: FC<NodeProps> = ({ selected, id }) => {
  const { params, updateParams } = useActionParams<NetworkMockParams>(id);

  // String fields
  const urlPattern = useSyncedString(params?.urlPattern ?? '', {
    onCommit: (v) => updateParams({ urlPattern: v || undefined }),
  });

  // Select fields
  const method = useSyncedSelect(params?.method ?? '', {
    onCommit: (v) => updateParams({ method: v || undefined }),
  });
  const mockType = useSyncedSelect(params?.mockType ?? 'response', {
    onCommit: (v) => updateParams({ mockType: v }),
  });
  const abortReason = useSyncedSelect(params?.abortReason ?? 'Failed', {
    onCommit: (v) => updateParams({ abortReason: v }),
  });

  // Number fields
  const statusCode = useSyncedNumber(params?.statusCode ?? 200, {
    min: 100,
    max: 599,
    fallback: 200,
    onCommit: (v) => updateParams({ statusCode: v }),
  });
  const delayMs = useSyncedNumber(params?.delayMs ?? 0, {
    min: 0,
    max: 600000,
    onCommit: (v) => updateParams({ delayMs: v || undefined }),
  });

  // Headers as text (special handling for object)
  const headersText = useSyncedString(headersToText(params?.headers), {
    trim: false,
    onCommit: (v) => updateParams({ headers: textToHeaders(v) }),
  });

  // Body field
  const body = useSyncedString(params?.body ?? '', {
    trim: false,
    onCommit: (v) => updateParams({ body: v.trim() ? v : undefined }),
  });

  const isResponse = mockType.value === 'response' || mockType.value === '';
  const isDelay = mockType.value === 'delay';
  const isAbort = mockType.value === 'abort';

  const methodOptions = useMemo(() => METHODS.map((label) => ({ label, value: label === 'Any' ? '' : label })), []);

  return (
    <BaseNode selected={selected} icon={Network} iconClassName="text-fuchsia-300" title="Network Mock">
      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">URL pattern</label>
          <input
            type="text"
            value={urlPattern.value}
            onChange={textInputHandler(urlPattern.setValue)}
            onBlur={urlPattern.commit}
            placeholder="https://api.example.com/* or regex:^https://api/.+"
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
          <p className="text-[10px] text-gray-500 mt-1">Prefix with <code>regex:</code> for regular expressions.</p>
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">HTTP method</label>
            <select
              value={method.value}
              onChange={selectInputHandler(method.setValue, method.commit)}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            >
              {methodOptions.map((option) => (
                <option key={option.label} value={option.value}>{option.label}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Mock type</label>
            <select
              value={mockType.value}
              onChange={selectInputHandler(mockType.setValue, mockType.commit)}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            >
              {MOCK_TYPES.map((option) => (
                <option key={option.value} value={option.value}>{option.label}</option>
              ))}
            </select>
          </div>
        </div>

        {isAbort && (
          <div>
            <label className="text-gray-400 block mb-1">Abort reason</label>
            <select
              value={abortReason.value}
              onChange={selectInputHandler(abortReason.setValue, abortReason.commit)}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            >
              {ABORT_REASONS.map((option) => (
                <option key={option.value} value={option.value}>{option.label}</option>
              ))}
            </select>
          </div>
        )}

        {isResponse && (
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="text-gray-400 block mb-1">Status code</label>
              <input
                type="number"
                min={100}
                max={599}
                value={statusCode.value}
                onChange={numberInputHandler(statusCode.setValue)}
                onBlur={statusCode.commit}
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
            </div>
            <div>
              <label className="text-gray-400 block mb-1">Delay before respond (ms)</label>
              <input
                type="number"
                min={0}
                value={delayMs.value}
                onChange={numberInputHandler(delayMs.setValue)}
                onBlur={delayMs.commit}
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
            </div>
          </div>
        )}

        {isDelay && (
          <div>
            <label className="text-gray-400 block mb-1">Delay before continuing (ms)</label>
            <input
              type="number"
              min={1}
              value={delayMs.value}
              onChange={numberInputHandler(delayMs.setValue)}
              onBlur={delayMs.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
            <p className="text-[10px] text-gray-500 mt-1">Required for delay mocks.</p>
          </div>
        )}

        {isResponse && (
          <>
            <div>
              <label className="text-gray-400 block mb-1">Headers</label>
              <textarea
                rows={3}
                value={headersText.value}
                onChange={textInputHandler(headersText.setValue)}
                onBlur={headersText.commit}
                placeholder={'Content-Type: application/json\nX-Feature: mock'}
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
            </div>
            <div>
              <label className="text-gray-400 block mb-1">Body</label>
              <textarea
                rows={4}
                value={body.value}
                onChange={textInputHandler(body.setValue)}
                onBlur={body.commit}
                placeholder='{"ok": true}'
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
              <p className="text-[10px] text-gray-500 mt-1">Paste raw text or JSON. Stored value is sent as-is.</p>
            </div>
          </>
        )}
      </div>
    </BaseNode>
  );
};

export default memo(NetworkMockNode);
