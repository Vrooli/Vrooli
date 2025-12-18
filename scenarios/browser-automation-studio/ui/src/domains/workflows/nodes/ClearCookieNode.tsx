import { FC, memo, useCallback, useEffect, useState } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { Cookie } from 'lucide-react';

const MIN_TIMEOUT = 100;

const sanitizeNumber = (value: number, fallback = 0): number => {
  if (!Number.isFinite(value)) {
    return fallback;
  }
  return Math.max(0, Math.round(value));
};

const ClearCookieNode: FC<NodeProps> = ({ data, selected, id }) => {
  const nodeData = (data ?? {}) as Record<string, unknown>;
  const { getNodes, setNodes } = useReactFlow();

  const [clearAll, setClearAll] = useState<boolean>(Boolean(nodeData.clearAll));
  const [name, setName] = useState<string>(String(nodeData.name ?? ''));
  const [url, setUrl] = useState<string>(String(nodeData.url ?? ''));
  const [domain, setDomain] = useState<string>(String(nodeData.domain ?? ''));
  const [path, setPath] = useState<string>(String(nodeData.path ?? ''));
  const [timeoutMs, setTimeoutMs] = useState<number>(sanitizeNumber(Number(nodeData.timeoutMs ?? 30000), 30000));
  const [waitForMs, setWaitForMs] = useState<number>(sanitizeNumber(Number(nodeData.waitForMs ?? 0)));

  useEffect(() => setClearAll(Boolean(nodeData.clearAll)), [nodeData.clearAll]);
  useEffect(() => setName(String(nodeData.name ?? '')), [nodeData.name]);
  useEffect(() => setUrl(String(nodeData.url ?? '')), [nodeData.url]);
  useEffect(() => setDomain(String(nodeData.domain ?? '')), [nodeData.domain]);
  useEffect(() => setPath(String(nodeData.path ?? '')), [nodeData.path]);
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

  const commitNumber = useCallback((field: 'timeoutMs' | 'waitForMs', value: number) => {
    const normalized = sanitizeNumber(value, field === 'timeoutMs' ? MIN_TIMEOUT : 0);
    if (field === 'timeoutMs') {
      const safe = Math.max(MIN_TIMEOUT, normalized);
      setTimeoutMs(safe);
      updateNodeData({ timeoutMs: safe });
      return;
    }
    setWaitForMs(normalized);
    updateNodeData({ waitForMs: normalized || undefined });
  }, [updateNodeData]);

  const disabledFields = clearAll;

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />

      <div className="flex items-center gap-2 mb-3">
        <Cookie size={16} className="text-rose-300" />
        <span className="font-semibold text-sm">Clear Cookie</span>
      </div>

      <div className="space-y-3 text-xs">
        <label className="flex items-center gap-2 text-gray-400">
          <input
            type="checkbox"
            checked={clearAll}
            onChange={(event) => {
              const next = event.target.checked;
              setClearAll(next);
              updateNodeData({ clearAll: next || undefined });
            }}
          />
          Remove every cookie in the current context
        </label>

        {!disabledFields && (
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="text-gray-400 block mb-1">Cookie name</label>
              <input
                type="text"
                value={name}
                onChange={(event) => setName(event.target.value)}
                onBlur={() => updateNodeData({ name: name.trim() })}
                placeholder="authToken"
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
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
        )}

        {!disabledFields && (
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="text-gray-400 block mb-1">Target URL</label>
              <input
                type="text"
                value={url}
                onChange={(event) => setUrl(event.target.value)}
                onBlur={() => updateNodeData({ url: url.trim() })}
                placeholder="https://example.com"
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
              value={timeoutMs}
              onChange={(event) => setTimeoutMs(Number(event.target.value))}
              onBlur={() => commitNumber('timeoutMs', timeoutMs)}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Post-clear wait (ms)</label>
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

export default memo(ClearCookieNode);
