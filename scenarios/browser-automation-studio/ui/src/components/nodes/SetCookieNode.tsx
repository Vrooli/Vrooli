import { FC, memo, useCallback, useEffect, useState } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { Cookie } from 'lucide-react';

const SAME_SITE_OPTIONS = [
  { label: 'Browser default', value: '' },
  { label: 'Lax', value: 'lax' },
  { label: 'Strict', value: 'strict' },
  { label: 'None', value: 'none' },
];

const MIN_TIMEOUT = 100;

const sanitizeNumber = (value: number, fallback = 0): number => {
  if (!Number.isFinite(value)) {
    return fallback;
  }
  return Math.max(0, Math.round(value));
};

const SetCookieNode: FC<NodeProps> = ({ data, selected, id }) => {
  const nodeData = (data ?? {}) as Record<string, unknown>;
  const { getNodes, setNodes } = useReactFlow();

  const [name, setName] = useState<string>(String(nodeData.name ?? ''));
  const [value, setValue] = useState<string>(String(nodeData.value ?? ''));
  const [url, setUrl] = useState<string>(String(nodeData.url ?? ''));
  const [domain, setDomain] = useState<string>(String(nodeData.domain ?? ''));
  const [path, setPath] = useState<string>(String(nodeData.path ?? ''));
  const [sameSite, setSameSite] = useState<string>(String(nodeData.sameSite ?? ''));
  const [secure, setSecure] = useState<boolean>(Boolean(nodeData.secure));
  const [httpOnly, setHttpOnly] = useState<boolean>(Boolean(nodeData.httpOnly));
  const [ttlSeconds, setTtlSeconds] = useState<number>(sanitizeNumber(Number(nodeData.ttlSeconds ?? 0)));
  const [expiresAt, setExpiresAt] = useState<string>(String(nodeData.expiresAt ?? ''));
  const [timeoutMs, setTimeoutMs] = useState<number>(sanitizeNumber(Number(nodeData.timeoutMs ?? 30000), 30000));
  const [waitForMs, setWaitForMs] = useState<number>(sanitizeNumber(Number(nodeData.waitForMs ?? 0)));

  useEffect(() => setName(String(nodeData.name ?? '')), [nodeData.name]);
  useEffect(() => setValue(String(nodeData.value ?? '')), [nodeData.value]);
  useEffect(() => setUrl(String(nodeData.url ?? '')), [nodeData.url]);
  useEffect(() => setDomain(String(nodeData.domain ?? '')), [nodeData.domain]);
  useEffect(() => setPath(String(nodeData.path ?? '')), [nodeData.path]);
  useEffect(() => setSameSite(String(nodeData.sameSite ?? '')), [nodeData.sameSite]);
  useEffect(() => setSecure(Boolean(nodeData.secure)), [nodeData.secure]);
  useEffect(() => setHttpOnly(Boolean(nodeData.httpOnly)), [nodeData.httpOnly]);
  useEffect(() => setTtlSeconds(sanitizeNumber(Number(nodeData.ttlSeconds ?? 0))), [nodeData.ttlSeconds]);
  useEffect(() => setExpiresAt(String(nodeData.expiresAt ?? '')), [nodeData.expiresAt]);
  useEffect(() => setTimeoutMs(sanitizeNumber(Number(nodeData.timeoutMs ?? 30000), 30000)), [nodeData.timeoutMs]);
  useEffect(() => setWaitForMs(sanitizeNumber(Number(nodeData.waitForMs ?? 0))), [nodeData.waitForMs]);

  const updateNodeData = useCallback((updates: Record<string, unknown>) => {
    const nodes = getNodes();
    setNodes(nodes.map((node) => {
      if (node.id !== id) {
        return node;
      }
      const nextData = { ...(node.data ?? {}) } as Record<string, unknown>;
      Object.entries(updates).forEach(([key, value]) => {
        if (value === undefined || value === null || value === '') {
          delete nextData[key];
        } else {
          nextData[key] = value;
        }
      });
      return { ...node, data: nextData };
    }));
  }, [getNodes, setNodes, id]);

  const commitNumber = useCallback((field: 'ttlSeconds' | 'timeoutMs' | 'waitForMs', value: number, fallback = 0) => {
    const normalized = sanitizeNumber(value, fallback);
    switch (field) {
      case 'ttlSeconds':
        setTtlSeconds(normalized);
        updateNodeData({ ttlSeconds: normalized || undefined });
        return;
      case 'timeoutMs': {
        const safe = Math.max(MIN_TIMEOUT, normalized || MIN_TIMEOUT);
        setTimeoutMs(safe);
        updateNodeData({ timeoutMs: safe });
        return;
      }
      default:
        setWaitForMs(normalized);
        updateNodeData({ waitForMs: normalized || undefined });
    }
  }, [updateNodeData]);

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />

      <div className="flex items-center gap-2 mb-3">
        <Cookie size={16} className="text-amber-300" />
        <span className="font-semibold text-sm">Set Cookie</span>
      </div>

      <div className="space-y-3 text-xs">
        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Cookie name</label>
            <input
              type="text"
              value={name}
              onChange={(event) => setName(event.target.value)}
              onBlur={() => updateNodeData({ name: name.trim() })}
              placeholder="sessionId"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Path</label>
            <input
              type="text"
              value={path}
              onChange={(event) => setPath(event.target.value)}
              onBlur={() => updateNodeData({ path: path.trim() })}
              placeholder="/"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        </div>

        <div>
          <label className="text-gray-400 block mb-1">Value</label>
          <textarea
            rows={2}
            value={value}
            onChange={(event) => setValue(event.target.value)}
            onBlur={() => updateNodeData({ value })}
            placeholder="Cookie payload"
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Target URL</label>
            <input
              type="text"
              value={url}
              onChange={(event) => setUrl(event.target.value)}
              onBlur={() => updateNodeData({ url: url.trim() })}
              placeholder="https://example.com/app"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
            <p className="text-[10px] text-gray-500 mt-1">Provide URL or domain to scope the cookie.</p>
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Domain</label>
            <input
              type="text"
              value={domain}
              onChange={(event) => setDomain(event.target.value)}
              onBlur={() => updateNodeData({ domain: domain.trim() })}
              placeholder=".example.com"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">SameSite</label>
            <select
              value={sameSite}
              onChange={(event) => {
                setSameSite(event.target.value);
                updateNodeData({ sameSite: event.target.value || undefined });
              }}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            >
              {SAME_SITE_OPTIONS.map((option) => (
                <option key={option.value || 'default'} value={option.value}>{option.label}</option>
              ))}
            </select>
          </div>
          <div className="flex items-center gap-4 pt-5">
            <label className="flex items-center gap-2 text-gray-400">
              <input
                type="checkbox"
                checked={secure}
                onChange={(event) => {
                  setSecure(event.target.checked);
                  updateNodeData({ secure: event.target.checked || undefined });
                }}
              />
              Secure
            </label>
            <label className="flex items-center gap-2 text-gray-400">
              <input
                type="checkbox"
                checked={httpOnly}
                onChange={(event) => {
                  setHttpOnly(event.target.checked);
                  updateNodeData({ httpOnly: event.target.checked || undefined });
                }}
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
              value={ttlSeconds}
              onChange={(event) => setTtlSeconds(Number(event.target.value))}
              onBlur={() => commitNumber('ttlSeconds', ttlSeconds)}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Expires at (RFC3339)</label>
            <input
              type="text"
              value={expiresAt}
              onChange={(event) => setExpiresAt(event.target.value)}
              onBlur={() => updateNodeData({ expiresAt: expiresAt.trim() })}
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
              value={timeoutMs}
              onChange={(event) => setTimeoutMs(Number(event.target.value))}
              onBlur={() => commitNumber('timeoutMs', timeoutMs, MIN_TIMEOUT)}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Post-set wait (ms)</label>
            <input
              type="number"
              min={0}
              value={waitForMs}
              onChange={(event) => setWaitForMs(Number(event.target.value))}
              onBlur={() => commitNumber('waitForMs', waitForMs)}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        </div>
      </div>

      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(SetCookieNode);
