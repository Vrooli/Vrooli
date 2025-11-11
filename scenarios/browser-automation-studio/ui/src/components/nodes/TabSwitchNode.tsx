import { FC, memo, useCallback, useEffect, useMemo, useState } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { PanelsTopLeft } from 'lucide-react';

const MODES = [
  { value: 'newest', label: 'Newest tab' },
  { value: 'oldest', label: 'Oldest tab' },
  { value: 'index', label: 'By index' },
  { value: 'title', label: 'By title match' },
  { value: 'url', label: 'By URL match' },
];

const DEFAULT_TIMEOUT = 30000;

const TabSwitchNode: FC<NodeProps> = ({ id, data, selected }) => {
  const nodeData = (data ?? {}) as Record<string, unknown>;
  const { getNodes, setNodes } = useReactFlow();

  const [mode, setMode] = useState<string>(() => typeof nodeData.switchBy === 'string' ? nodeData.switchBy : 'newest');
  const [index, setIndex] = useState<number>(() => Number(nodeData.index ?? 0) || 0);
  const [titleMatch, setTitleMatch] = useState<string>(() => typeof nodeData.titleMatch === 'string' ? nodeData.titleMatch : '');
  const [urlMatch, setUrlMatch] = useState<string>(() => typeof nodeData.urlMatch === 'string' ? nodeData.urlMatch : '');
  const [waitForNew, setWaitForNew] = useState<boolean>(() => Boolean(nodeData.waitForNew));
  const [closeOld, setCloseOld] = useState<boolean>(() => Boolean(nodeData.closeOld));
  const [timeoutMs, setTimeoutMs] = useState<number>(() => Number(nodeData.timeoutMs ?? DEFAULT_TIMEOUT) || DEFAULT_TIMEOUT);

  useEffect(() => {
    setMode(typeof nodeData.switchBy === 'string' ? nodeData.switchBy : 'newest');
  }, [nodeData.switchBy]);

  useEffect(() => {
    setIndex(Number(nodeData.index ?? 0) || 0);
  }, [nodeData.index]);

  useEffect(() => {
    setTitleMatch(typeof nodeData.titleMatch === 'string' ? nodeData.titleMatch : '');
  }, [nodeData.titleMatch]);

  useEffect(() => {
    setUrlMatch(typeof nodeData.urlMatch === 'string' ? nodeData.urlMatch : '');
  }, [nodeData.urlMatch]);

  useEffect(() => {
    setWaitForNew(Boolean(nodeData.waitForNew));
  }, [nodeData.waitForNew]);

  useEffect(() => {
    setCloseOld(Boolean(nodeData.closeOld));
  }, [nodeData.closeOld]);

  useEffect(() => {
    setTimeoutMs(Number(nodeData.timeoutMs ?? DEFAULT_TIMEOUT) || DEFAULT_TIMEOUT);
  }, [nodeData.timeoutMs]);

  const updateNodeData = useCallback((updates: Record<string, unknown>) => {
    const nodes = getNodes();
    setNodes(nodes.map((node) => {
      if (node.id !== id) {
        return node;
      }
      return {
        ...node,
        data: {
          ...(node.data ?? {}),
          ...updates,
        },
      };
    }));
  }, [getNodes, setNodes, id]);

  const requiresIndex = mode === 'index';
  const requiresTitle = mode === 'title';
  const requiresUrl = mode === 'url';

  const modeDescription = useMemo(() => {
    switch (mode) {
      case 'oldest':
        return 'Selects the first tab that was opened in this workflow.';
      case 'index':
        return 'Zero-based index of the tab (0 = first tab).';
      case 'title':
        return 'Matches tabs by window title using substring or regex.';
      case 'url':
        return 'Matches tabs by URL using substring or regex.';
      default:
        return 'Switches to the most recently opened tab.';
    }
  }, [mode]);

  const handleModeChange = useCallback((event: React.ChangeEvent<HTMLSelectElement>) => {
    const value = event.target.value || 'newest';
    setMode(value);
    updateNodeData({ switchBy: value });
  }, [updateNodeData]);

  const handleIndexBlur = useCallback(() => {
    const normalized = Math.max(0, Math.round(index) || 0);
    setIndex(normalized);
    updateNodeData({ index: normalized });
  }, [index, updateNodeData]);

  const handleTimeoutBlur = useCallback(() => {
    const normalized = Math.max(1000, Math.round(timeoutMs) || DEFAULT_TIMEOUT);
    setTimeoutMs(normalized);
    updateNodeData({ timeoutMs: normalized });
  }, [timeoutMs, updateNodeData]);

  const handleTitleBlur = useCallback(() => {
    updateNodeData({ titleMatch: titleMatch.trim() });
  }, [titleMatch, updateNodeData]);

  const handleUrlBlur = useCallback(() => {
    updateNodeData({ urlMatch: urlMatch.trim() });
  }, [urlMatch, updateNodeData]);

  const handleWaitToggle = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const next = event.target.checked;
    setWaitForNew(next);
    updateNodeData({ waitForNew: next });
  }, [updateNodeData]);

  const handleCloseToggle = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const next = event.target.checked;
    setCloseOld(next);
    updateNodeData({ closeOld: next });
  }, [updateNodeData]);

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />
      <div className="flex items-center gap-2 mb-3">
        <PanelsTopLeft size={16} className="text-violet-300" />
        <span className="font-semibold text-sm">Tab Switch</span>
      </div>

      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">Switch strategy</label>
          <select
            value={mode}
            onChange={handleModeChange}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          >
            {MODES.map((option) => (
              <option key={option.value} value={option.value}>{option.label}</option>
            ))}
          </select>
          <p className="text-gray-500 mt-1">{modeDescription}</p>
        </div>

        {requiresIndex && (
          <div>
            <label className="text-gray-400 block mb-1">Tab index</label>
            <input
              type="number"
              min={0}
              value={index}
              onChange={(event) => setIndex(Number(event.target.value))}
              onBlur={handleIndexBlur}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        {requiresTitle && (
          <div>
            <label className="text-gray-400 block mb-1">Title pattern</label>
            <input
              type="text"
              value={titleMatch}
              placeholder="/Invoice/ or Dashboard"
              onChange={(event) => setTitleMatch(event.target.value)}
              onBlur={handleTitleBlur}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        {requiresUrl && (
          <div>
            <label className="text-gray-400 block mb-1">URL pattern</label>
            <input
              type="text"
              value={urlMatch}
              placeholder="/auth\\/callback/ or /settings"
              onChange={(event) => setUrlMatch(event.target.value)}
              onBlur={handleUrlBlur}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        <div className="grid grid-cols-2 gap-2">
          <label className="flex items-center gap-2">
            <input type="checkbox" className="accent-flow-accent" checked={waitForNew} onChange={handleWaitToggle} />
            <span className="text-gray-400">Wait for new tab</span>
          </label>
          <label className="flex items-center gap-2">
            <input type="checkbox" className="accent-flow-accent" checked={closeOld} onChange={handleCloseToggle} />
            <span className="text-gray-400">Close previous tab</span>
          </label>
        </div>

        <div>
          <label className="text-gray-400 block mb-1">Timeout (ms)</label>
          <input
            type="number"
            min={1000}
            value={timeoutMs}
            onChange={(event) => setTimeoutMs(Number(event.target.value))}
            onBlur={handleTimeoutBlur}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>

        <p className="text-gray-500">
          Switch workflows between pop-ups, OAuth flows, or background tabs. Pattern fields accept plain text or full regular expressions.
        </p>
      </div>

      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(TabSwitchNode);
