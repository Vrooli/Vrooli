import { FC, memo, useCallback, useEffect, useState } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { Trash2 } from 'lucide-react';

const STORAGE_OPTIONS = [
  { label: 'localStorage', value: 'localStorage' },
  { label: 'sessionStorage', value: 'sessionStorage' },
];

const MIN_TIMEOUT = 100;

const sanitizeNumber = (value: number, fallback = 0): number => {
  if (!Number.isFinite(value)) {
    return fallback;
  }
  return Math.max(0, Math.round(value));
};

const ClearStorageNode: FC<NodeProps> = ({ data, selected, id }) => {
  const nodeData = (data ?? {}) as Record<string, unknown>;
  const { getNodes, setNodes } = useReactFlow();

  const [storageType, setStorageType] = useState<string>(String(nodeData.storageType ?? 'localStorage'));
  const [clearAll, setClearAll] = useState<boolean>(Boolean(nodeData.clearAll));
  const [keyValue, setKeyValue] = useState<string>(String(nodeData.key ?? ''));
  const [timeoutMs, setTimeoutMs] = useState<number>(sanitizeNumber(Number(nodeData.timeoutMs ?? 15000), 15000));
  const [waitForMs, setWaitForMs] = useState<number>(sanitizeNumber(Number(nodeData.waitForMs ?? 0)));

  useEffect(() => setStorageType(String(nodeData.storageType ?? 'localStorage')), [nodeData.storageType]);
  useEffect(() => setClearAll(Boolean(nodeData.clearAll)), [nodeData.clearAll]);
  useEffect(() => setKeyValue(String(nodeData.key ?? '')), [nodeData.key]);
  useEffect(() => setTimeoutMs(sanitizeNumber(Number(nodeData.timeoutMs ?? 15000), 15000)), [nodeData.timeoutMs]);
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

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />

      <div className="flex items-center gap-2 mb-3">
        <Trash2 size={16} className="text-red-300" />
        <span className="font-semibold text-sm">Clear Storage</span>
      </div>

      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">Storage</label>
          <select
            value={storageType}
            onChange={(event) => {
              setStorageType(event.target.value);
              updateNodeData({ storageType: event.target.value });
            }}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          >
            {STORAGE_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>{option.label}</option>
            ))}
          </select>
        </div>

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
          Remove every entry
        </label>

        {!clearAll && (
          <div>
            <label className="text-gray-400 block mb-1">Key</label>
            <input
              type="text"
              value={keyValue}
              onChange={(event) => setKeyValue(event.target.value)}
              onBlur={() => updateNodeData({ key: keyValue.trim() })}
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

export default memo(ClearStorageNode);
