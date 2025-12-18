import { FC, memo, useCallback, useEffect, useMemo, useState } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { UploadCloud, FileWarning } from 'lucide-react';
import { selectors } from '@constants/selectors';

const normalizeTimeout = (value: unknown, fallback: number): number => {
  const numeric = Number(value);
  if (Number.isFinite(numeric) && numeric > 0) {
    return Math.round(numeric);
  }
  return fallback;
};

const normalizeWait = (value: unknown): number => {
  const numeric = Number(value);
  if (Number.isFinite(numeric) && numeric >= 0) {
    return Math.round(numeric);
  }
  return 0;
};

const deriveFileText = (nodeData: Record<string, unknown>): string => {
  const rawPaths = Array.isArray(nodeData.filePaths)
    ? (nodeData.filePaths as unknown[]).filter((entry): entry is string => typeof entry === 'string')
    : [];
  if (rawPaths.length > 0) {
    return rawPaths.join('\n');
  }
  if (typeof nodeData.filePath === 'string') {
    return nodeData.filePath;
  }
  return '';
};

const parseFilePaths = (value: string): string[] => {
  return value
    .split(/\r?\n|,/)
    .map((entry) => entry.trim())
    .filter(Boolean);
};

const UploadFileNode: FC<NodeProps> = ({ id, data, selected }) => {
  const nodeData = (data ?? {}) as Record<string, unknown>;
  const { getNodes, setNodes } = useReactFlow();

  const [selector, setSelector] = useState<string>(typeof nodeData.selector === 'string' ? nodeData.selector : '');
  const [fileText, setFileText] = useState<string>(() => deriveFileText(nodeData));
  const [timeoutMs, setTimeoutMs] = useState<number>(() => normalizeTimeout(nodeData.timeoutMs, 30000));
  const [waitForMs, setWaitForMs] = useState<number>(() => normalizeWait(nodeData.waitForMs));

  useEffect(() => {
    setSelector(typeof nodeData.selector === 'string' ? nodeData.selector : '');
  }, [nodeData.selector]);

  useEffect(() => {
    setFileText(deriveFileText(nodeData));
  }, [nodeData.filePaths, nodeData.filePath]);

  useEffect(() => {
    setTimeoutMs(normalizeTimeout(nodeData.timeoutMs, 30000));
  }, [nodeData.timeoutMs]);

  useEffect(() => {
    setWaitForMs(normalizeWait(nodeData.waitForMs));
  }, [nodeData.waitForMs]);

  const updateNodeData = useCallback((updates: Record<string, unknown>) => {
    const nodes = getNodes();
    setNodes(nodes.map((node) => {
      if (node.id !== id) {
        return node;
      }
      const nextData = { ...(node.data ?? {}) } as Record<string, unknown>;
      Object.entries(updates).forEach(([key, value]) => {
        if (value === undefined || value === null || (typeof value === 'string' && value === '')) {
          delete nextData[key];
        } else {
          nextData[key] = value;
        }
      });
      return { ...node, data: nextData };
    }));
  }, [getNodes, setNodes, id]);

  const commitSelector = useCallback(() => {
    const trimmed = selector.trim();
    setSelector(trimmed);
    updateNodeData({ selector: trimmed || undefined });
  }, [selector, updateNodeData]);

  const commitTimeout = useCallback(() => {
    const normalized = Math.max(500, Math.round(timeoutMs) || 500);
    setTimeoutMs(normalized);
    updateNodeData({ timeoutMs: normalized });
  }, [timeoutMs, updateNodeData]);

  const commitWait = useCallback(() => {
    const normalized = Math.max(0, Math.round(waitForMs) || 0);
    setWaitForMs(normalized);
    updateNodeData({ waitForMs: normalized });
  }, [waitForMs, updateNodeData]);

  const commitFilePaths = useCallback(() => {
    const parsed = parseFilePaths(fileText);
    const updates: Record<string, unknown> = {};
    if (parsed.length === 0) {
      updates.filePaths = undefined;
      updates.filePath = undefined;
    } else if (parsed.length === 1) {
      updates.filePaths = parsed;
      updates.filePath = parsed[0];
    } else {
      updates.filePaths = parsed;
      updates.filePath = undefined;
    }
    updateNodeData(updates);
  }, [fileText, updateNodeData]);

  const activePaths = useMemo(() => parseFilePaths(fileText), [fileText]);

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`} data-node-id={id}>
      <Handle type="target" position={Position.Top} className="node-handle" />
      <div className="flex items-start gap-2 mb-3">
        <UploadCloud size={16} className="text-pink-300" />
        <div>
          <div className="font-semibold text-sm">Upload File</div>
          <p className="text-[11px] text-gray-500">Attach local files to file inputs</p>
        </div>
      </div>

      <div className="space-y-3 text-xs">
        <div>
          <label className="text-[11px] font-semibold text-gray-400">Target selector</label>
          <input
            type="text"
            value={selector}
            onChange={(event) => setSelector(event.target.value)}
            onBlur={commitSelector}
            placeholder="#upload"
            className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
          <p className="text-[10px] text-gray-500 mt-1">Must point to an &lt;input type="file"&gt; element.</p>
        </div>

        <div>
          <label className="text-[11px] font-semibold text-gray-400">File paths</label>
          <textarea
            value={fileText}
            onChange={(event) => setFileText(event.target.value)}
            onBlur={commitFilePaths}
            placeholder="/home/me/Documents/example.pdf"
            rows={3}
            className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
          <div className="mt-1 text-[10px] text-gray-500 flex items-center gap-1">
            <FileWarning size={12} className="text-amber-300" />
            Paths must be absolute on the execution machine. One per line for multiple files.
          </div>
          {activePaths.length > 0 && (
            <p
              className="text-[10px] text-emerald-300 mt-1"
              data-testid={selectors.workflowBuilder.nodes.upload.pathCount({ id })}
            >
              {activePaths.length === 1 ? '1 file selected' : `${activePaths.length} files selected`}
            </p>
          )}
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-[11px] font-semibold text-gray-400">Timeout (ms)</label>
            <input
              type="number"
              min={100}
              value={timeoutMs}
              onChange={(event) => setTimeoutMs(Number(event.target.value))}
              onBlur={commitTimeout}
              className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-[11px] font-semibold text-gray-400">Wait after (ms)</label>
            <input
              type="number"
              min={0}
              value={waitForMs}
              onChange={(event) => setWaitForMs(Number(event.target.value))}
              onBlur={commitWait}
              className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        </div>
      </div>

      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(UploadFileNode);
