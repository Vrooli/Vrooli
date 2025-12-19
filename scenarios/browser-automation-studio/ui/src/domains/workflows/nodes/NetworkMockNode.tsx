import { FC, memo } from 'react';
import type { NodeProps } from 'reactflow';
import { Network } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useSyncedString, useSyncedNumber, useSyncedSelect } from '@hooks/useSyncedField';
import BaseNode from './BaseNode';
import { NodeTextField, NodeTextArea, NodeNumberField, NodeSelectField, FieldRow } from './fields';

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
  { value: 'response', label: 'Stub response' },
  { value: 'abort', label: 'Abort request' },
  { value: 'delay', label: 'Delay & passthrough' },
];

const METHOD_OPTIONS = [
  { value: '', label: 'Any' },
  { value: 'GET', label: 'GET' },
  { value: 'POST', label: 'POST' },
  { value: 'PUT', label: 'PUT' },
  { value: 'PATCH', label: 'PATCH' },
  { value: 'DELETE', label: 'DELETE' },
  { value: 'OPTIONS', label: 'OPTIONS' },
  { value: 'HEAD', label: 'HEAD' },
];

const ABORT_REASONS = [
  { value: 'Failed', label: 'Generic failure' },
  { value: 'BlockedByClient', label: 'Client blocked' },
  { value: 'BlockedByResponse', label: 'Server blocked' },
  { value: 'TimedOut', label: 'Timed out' },
  { value: 'Aborted', label: 'Aborted' },
  { value: 'ConnectionRefused', label: 'Connection refused' },
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

  return (
    <BaseNode selected={selected} icon={Network} iconClassName="text-fuchsia-300" title="Network Mock">
      <div className="space-y-3 text-xs">
        <NodeTextField
          field={urlPattern}
          label="URL pattern"
          placeholder="https://api.example.com/* or regex:^https://api/.+"
          description="Prefix with regex: for regular expressions."
        />

        <FieldRow>
          <NodeSelectField field={method} label="HTTP method" options={METHOD_OPTIONS} />
          <NodeSelectField field={mockType} label="Mock type" options={MOCK_TYPES} />
        </FieldRow>

        {isAbort && (
          <NodeSelectField field={abortReason} label="Abort reason" options={ABORT_REASONS} />
        )}

        {isResponse && (
          <FieldRow>
            <NodeNumberField field={statusCode} label="Status code" min={100} max={599} />
            <NodeNumberField field={delayMs} label="Delay before respond (ms)" min={0} />
          </FieldRow>
        )}

        {isDelay && (
          <NodeNumberField
            field={delayMs}
            label="Delay before continuing (ms)"
            min={1}
            description="Required for delay mocks."
          />
        )}

        {isResponse && (
          <>
            <NodeTextArea
              field={headersText}
              label="Headers"
              rows={3}
              placeholder={'Content-Type: application/json\nX-Feature: mock'}
            />
            <NodeTextArea
              field={body}
              label="Body"
              rows={4}
              placeholder='{"ok": true}'
              description="Paste raw text or JSON. Stored value is sent as-is."
            />
          </>
        )}
      </div>
    </BaseNode>
  );
};

export default memo(NetworkMockNode);
