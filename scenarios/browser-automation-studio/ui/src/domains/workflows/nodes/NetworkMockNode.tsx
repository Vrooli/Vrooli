import { FC, memo, useCallback, useEffect, useMemo, useState } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { Network } from 'lucide-react';

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

const sanitizeNumber = (value: unknown, fallback = 0, min = 0, max = 1_000_000): number => {
  const numeric = Number(value);
  if (!Number.isFinite(numeric)) {
    return fallback;
  }
  return Math.min(Math.max(Math.round(numeric), min), max);
};

const headersToText = (headers: unknown): string => {
  if (!headers || typeof headers !== 'object') {
    return '';
  }
  return Object.entries(headers as Record<string, unknown>)
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

const NetworkMockNode: FC<NodeProps> = ({ data, selected, id }) => {
  const nodeData = (data ?? {}) as Record<string, unknown>;
  const { getNodes, setNodes } = useReactFlow();

  const [urlPattern, setUrlPattern] = useState<string>(String(nodeData.urlPattern ?? ''));
  const [method, setMethod] = useState<string>(String(nodeData.method ?? ''));
  const [mockType, setMockType] = useState<string>(String(nodeData.mockType ?? 'response'));
  const [statusCode, setStatusCode] = useState<number>(sanitizeNumber(nodeData.statusCode ?? 200, 200, 100, 599));
  const [delayMs, setDelayMs] = useState<number>(sanitizeNumber(nodeData.delayMs ?? 0));
  const [headersText, setHeadersText] = useState<string>(headersToText(nodeData.headers));
  const [body, setBody] = useState<string>(String(nodeData.body ?? ''));
  const [abortReason, setAbortReason] = useState<string>(String(nodeData.abortReason ?? 'Failed'));

  useEffect(() => setUrlPattern(String(nodeData.urlPattern ?? '')), [nodeData.urlPattern]);
  useEffect(() => setMethod(String(nodeData.method ?? '')), [nodeData.method]);
  useEffect(() => setMockType(String(nodeData.mockType ?? 'response')), [nodeData.mockType]);
  useEffect(() => setStatusCode(sanitizeNumber(nodeData.statusCode ?? 200, 200, 100, 599)), [nodeData.statusCode]);
  useEffect(() => setDelayMs(sanitizeNumber(nodeData.delayMs ?? 0)), [nodeData.delayMs]);
  useEffect(() => setHeadersText(headersToText(nodeData.headers)), [nodeData.headers]);
  useEffect(() => setBody(String(nodeData.body ?? '')), [nodeData.body]);
  useEffect(() => setAbortReason(String(nodeData.abortReason ?? 'Failed')), [nodeData.abortReason]);

  const updateNodeData = useCallback((updates: Record<string, unknown>) => {
    const nodes = getNodes();
    setNodes(nodes.map((node) => {
      if (node.id !== id) {
        return node;
      }
      const nextData = { ...(node.data ?? {}) } as Record<string, unknown>;
      Object.entries(updates).forEach(([key, value]) => {
        if (value === undefined || value === null || value === '' || (typeof value === 'number' && Number.isNaN(value))) {
          delete nextData[key];
        } else {
          nextData[key] = value;
        }
      });
      return { ...node, data: nextData };
    }));
  }, [getNodes, setNodes, id]);

  const isResponse = mockType === 'response' || mockType === '';
  const isDelay = mockType === 'delay';
  const isAbort = mockType === 'abort';

  const methodOptions = useMemo(() => METHODS.map((label) => ({ label, value: label === 'Any' ? '' : label })), []);

  const handleHeadersBlur = useCallback((value: string) => {
    const parsed = textToHeaders(value);
    updateNodeData({ headers: parsed });
  }, [updateNodeData]);

  const handleBodyBlur = useCallback((value: string) => {
    updateNodeData({ body: value.trim() ? value : undefined });
  }, [updateNodeData]);

  const handleStatusBlur = useCallback((value: number) => {
    const normalized = sanitizeNumber(value, 200, 100, 599);
    setStatusCode(normalized);
    updateNodeData({ statusCode: normalized });
  }, [updateNodeData]);

  const handleDelayBlur = useCallback((value: number) => {
    const normalized = sanitizeNumber(value, 0, 0, 600000);
    setDelayMs(normalized);
    updateNodeData({ delayMs: normalized || undefined });
  }, [updateNodeData]);

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />

      <div className="flex items-center gap-2 mb-3">
        <Network size={16} className="text-fuchsia-300" />
        <span className="font-semibold text-sm">Network Mock</span>
      </div>

      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">URL pattern</label>
          <input
            type="text"
            value={urlPattern}
            onChange={(event) => setUrlPattern(event.target.value)}
            onBlur={() => updateNodeData({ urlPattern: urlPattern.trim() })}
            placeholder="https://api.example.com/* or regex:^https://api/.+"
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
          <p className="text-[10px] text-gray-500 mt-1">Prefix with <code>regex:</code> for regular expressions.</p>
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">HTTP method</label>
            <select
              value={method}
              onChange={(event) => {
                setMethod(event.target.value);
                updateNodeData({ method: event.target.value || undefined });
              }}
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
              value={mockType}
              onChange={(event) => {
                const value = event.target.value || 'response';
                setMockType(value);
                updateNodeData({ mockType: value });
              }}
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
              value={abortReason}
              onChange={(event) => {
                setAbortReason(event.target.value);
                updateNodeData({ abortReason: event.target.value });
              }}
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
                value={statusCode}
                onChange={(event) => setStatusCode(Number(event.target.value))}
                onBlur={(event) => handleStatusBlur(Number(event.target.value))}
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
            </div>
            <div>
              <label className="text-gray-400 block mb-1">Delay before respond (ms)</label>
              <input
                type="number"
                min={0}
                value={delayMs}
                onChange={(event) => setDelayMs(Number(event.target.value))}
                onBlur={(event) => handleDelayBlur(Number(event.target.value))}
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
              value={delayMs}
              onChange={(event) => setDelayMs(Number(event.target.value))}
              onBlur={(event) => handleDelayBlur(Number(event.target.value))}
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
                value={headersText}
                onChange={(event) => setHeadersText(event.target.value)}
                onBlur={(event) => handleHeadersBlur(event.target.value)}
                placeholder={'Content-Type: application/json\nX-Feature: mock'}
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
            </div>
            <div>
              <label className="text-gray-400 block mb-1">Body</label>
              <textarea
                rows={4}
                value={body}
                onChange={(event) => setBody(event.target.value)}
                onBlur={(event) => handleBodyBlur(event.target.value)}
                placeholder='{"ok": true}'
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
              <p className="text-[10px] text-gray-500 mt-1">Paste raw text or JSON. Stored value is sent as-is.</p>
            </div>
          </>
        )}
      </div>

      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(NetworkMockNode);
