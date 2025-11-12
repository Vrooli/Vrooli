import { FC, memo, useCallback, useEffect, useMemo, useState } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { LayoutPanelTop } from 'lucide-react';

const MODES = [
  { value: 'selector', label: 'By selector' },
  { value: 'index', label: 'By index' },
  { value: 'name', label: 'By name attribute' },
  { value: 'url', label: 'By URL match' },
  { value: 'parent', label: 'Parent frame' },
  { value: 'main', label: 'Main document' },
];

const DEFAULT_TIMEOUT = 30000;

const modeDescriptionMap: Record<string, string> = {
  selector: 'Queries the current document for an iframe using a CSS selector.',
  index: 'Targets the nth iframe within the active document (0 = first).',
  name: 'Matches iframe elements by the value of their name attribute.',
  url: 'Matches frames whose current URL contains the pattern or matches /regex/.',
  parent: 'Returns to the parent frame of the current context.',
  main: 'Clears frame context and returns to the top-level document.',
};

const FrameSwitchNode: FC<NodeProps> = ({ id, data, selected }) => {
  const nodeData = (data ?? {}) as Record<string, unknown>;
  const { getNodes, setNodes } = useReactFlow();

  const [mode, setMode] = useState<string>(() => typeof nodeData.switchBy === 'string' ? nodeData.switchBy : 'selector');
  const [selector, setSelector] = useState<string>(() => typeof nodeData.selector === 'string' ? nodeData.selector : '');
  const [frameName, setFrameName] = useState<string>(() => typeof nodeData.name === 'string' ? nodeData.name : '');
  const [urlMatch, setUrlMatch] = useState<string>(() => typeof nodeData.urlMatch === 'string' ? nodeData.urlMatch : '');
  const [index, setIndex] = useState<number>(() => Number(nodeData.index ?? 0) || 0);
  const [timeoutMs, setTimeoutMs] = useState<number>(() => Number(nodeData.timeoutMs ?? DEFAULT_TIMEOUT) || DEFAULT_TIMEOUT);

  useEffect(() => {
    setMode(typeof nodeData.switchBy === 'string' ? nodeData.switchBy : 'selector');
  }, [nodeData.switchBy]);

  useEffect(() => {
    setSelector(typeof nodeData.selector === 'string' ? nodeData.selector : '');
  }, [nodeData.selector]);

  useEffect(() => {
    setFrameName(typeof nodeData.name === 'string' ? nodeData.name : '');
  }, [nodeData.name]);

  useEffect(() => {
    setUrlMatch(typeof nodeData.urlMatch === 'string' ? nodeData.urlMatch : '');
  }, [nodeData.urlMatch]);

  useEffect(() => {
    setIndex(Number(nodeData.index ?? 0) || 0);
  }, [nodeData.index]);

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
  }, [getNodes, id, setNodes]);

  const modeDescription = useMemo(() => modeDescriptionMap[mode] ?? modeDescriptionMap.selector, [mode]);

  const handleModeChange = useCallback((event: React.ChangeEvent<HTMLSelectElement>) => {
    const value = event.target.value || 'selector';
    setMode(value);
    updateNodeData({ switchBy: value });
  }, [updateNodeData]);

  const handleSelectorBlur = useCallback(() => {
    updateNodeData({ selector: selector.trim() });
  }, [selector, updateNodeData]);

  const handleNameBlur = useCallback(() => {
    updateNodeData({ name: frameName.trim() });
  }, [frameName, updateNodeData]);

  const handleUrlBlur = useCallback(() => {
    updateNodeData({ urlMatch: urlMatch.trim() });
  }, [updateNodeData, urlMatch]);

  const handleIndexBlur = useCallback(() => {
    const normalized = Math.max(0, Math.round(index) || 0);
    setIndex(normalized);
    updateNodeData({ index: normalized });
  }, [index, updateNodeData]);

  const handleTimeoutBlur = useCallback(() => {
    const normalized = Math.max(500, Math.round(timeoutMs) || DEFAULT_TIMEOUT);
    setTimeoutMs(normalized);
    updateNodeData({ timeoutMs: normalized });
  }, [timeoutMs, updateNodeData]);

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />
      <div className="flex items-center gap-2 mb-3">
        <LayoutPanelTop size={16} className="text-lime-300" />
        <span className="font-semibold text-sm">Frame Switch</span>
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

        {mode === 'selector' && (
          <div>
            <label className="text-gray-400 block mb-1">IFrame selector</label>
            <input
              type="text"
              value={selector}
              onChange={(event) => setSelector(event.target.value)}
              onBlur={handleSelectorBlur}
              placeholder="iframe[data-testid='editor']"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        {mode === 'index' && (
          <div>
            <label className="text-gray-400 block mb-1">Frame index</label>
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

        {mode === 'name' && (
          <div>
            <label className="text-gray-400 block mb-1">Frame name</label>
            <input
              type="text"
              value={frameName}
              onChange={(event) => setFrameName(event.target.value)}
              onBlur={handleNameBlur}
              placeholder="paymentFrame"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        {mode === 'url' && (
          <div>
            <label className="text-gray-400 block mb-1">URL pattern</label>
            <input
              type="text"
              value={urlMatch}
              onChange={(event) => setUrlMatch(event.target.value)}
              onBlur={handleUrlBlur}
              placeholder="/billing\\/portal/ or checkout"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        <div>
          <label className="text-gray-400 block mb-1">Timeout (ms)</label>
          <input
            type="number"
            min={500}
            value={timeoutMs}
            onChange={(event) => setTimeoutMs(Number(event.target.value))}
            onBlur={handleTimeoutBlur}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
          <p className="text-gray-500 mt-1">Applies only to selector/index/name/url modes.</p>
        </div>
      </div>

      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(FrameSwitchNode);
