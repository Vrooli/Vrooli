import { memo, FC, useCallback, useEffect, useMemo, useState } from 'react';
import { Handle, Position, NodeProps, useReactFlow } from 'reactflow';
import {
  Camera,
  Globe,
  SlidersHorizontal,
  Focus,
  Wand2,
  Droplet,
  ZoomIn,
  Palette,
  Timer,
} from 'lucide-react';
import { useUpstreamUrl } from '../../hooks/useUpstreamUrl';

const ScreenshotNode: FC<NodeProps> = ({ data, selected, id }) => {
  const upstreamUrl = useUpstreamUrl(id);
  const { getNodes, setNodes } = useReactFlow();
  const [highlightInput, setHighlightInput] = useState<string>(() =>
    Array.isArray(data.highlightSelectors) ? data.highlightSelectors.join(', ') : ''
  );
  const [maskInput, setMaskInput] = useState<string>(() =>
    Array.isArray(data.maskSelectors) ? data.maskSelectors.join(', ') : ''
  );

  useEffect(() => {
    setHighlightInput(Array.isArray(data.highlightSelectors) ? data.highlightSelectors.join(', ') : '');
  }, [data.highlightSelectors]);

  useEffect(() => {
    setMaskInput(Array.isArray(data.maskSelectors) ? data.maskSelectors.join(', ') : '');
  }, [data.maskSelectors]);

  const [showAdvanced, setShowAdvanced] = useState<boolean>(false);

  const updateNodeData = useCallback(
    (updates: Record<string, any>) => {
      const nodes = getNodes();
      const updatedNodes = nodes.map((node) => {
        if (node.id !== id) {
          return node;
        }
        const nextData = { ...node.data } as Record<string, any>;
        Object.entries(updates).forEach(([key, value]) => {
          const shouldRemove =
            value === undefined ||
            (Array.isArray(value) && value.length === 0) ||
            (typeof value === 'string' && value.trim() === '');
          if (shouldRemove) {
            delete nextData[key];
          } else {
            nextData[key] = value;
          }
        });
        return {
          ...node,
          data: nextData,
        };
      });
      setNodes(updatedNodes);
    },
    [getNodes, setNodes, id]
  );

  const handleSelectorListChange = useCallback(
    (rawValue: string, key: 'highlightSelectors' | 'maskSelectors') => {
      const selectors = rawValue
        .split(/[\n,]+/)
        .map((entry) => entry.trim())
        .filter((entry) => entry.length > 0);
      updateNodeData({ [key]: selectors.length > 0 ? selectors : undefined });
    },
    [updateNodeData]
  );

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''} w-80`}>
      <Handle type="target" position={Position.Top} className="node-handle" />
      
      <div className="flex items-center gap-2 mb-2">
        <Camera size={16} className="text-purple-400" />
        <span className="font-semibold text-sm">Screenshot</span>
      </div>
      
      {upstreamUrl && (
        <div className="flex items-center gap-1 mb-2 p-1 bg-flow-bg/50 rounded text-xs border border-gray-700">
          <Globe size={12} className="text-blue-400 flex-shrink-0" />
          <span className="text-gray-400 truncate" title={upstreamUrl}>
            {upstreamUrl}
          </span>
        </div>
      )}
      
      <input
        type="text"
        placeholder="Screenshot name..."
        className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none mb-2"
        value={data.name ?? ''}
        onChange={(e) => {
          updateNodeData({ name: e.target.value });
        }}
      />
      
      <label className="flex items-center gap-2 text-xs">
        <input
          type="checkbox"
          className="rounded"
          defaultChecked={data.fullPage || false}
          onChange={(e) => {
            const isFullPage = e.target.checked;
            updateNodeData({ fullPage: isFullPage });
          }}
        />
        <span>Full page</span>
      </label>

      <button
        type="button"
        className="mt-3 mb-2 flex items-center gap-2 text-xs text-gray-400 hover:text-white transition-colors"
        onClick={() => setShowAdvanced((prev) => !prev)}
      >
        <SlidersHorizontal size={14} />
        {showAdvanced ? 'Hide advanced capture options' : 'Show advanced capture options'}
      </button>

      {showAdvanced && (
        <div className="space-y-3 border-t border-gray-800 pt-3">
          <div className="space-y-2">
            <label className="text-xs text-gray-400 flex items-center gap-2">
              <Focus size={12} className="text-blue-300" /> Focus selector
            </label>
            <input
              type="text"
              placeholder="CSS selector to focus before capture"
              className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
              value={data.focusSelector ?? ''}
              onChange={(e) => updateNodeData({ focusSelector: e.target.value })}
            />
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between gap-2">
              <label className="text-xs text-gray-400 flex items-center gap-2">
                <Wand2 size={12} className="text-purple-300" /> Highlight selectors
              </label>
              <span className="text-[10px] text-gray-500">comma or newline separated</span>
            </div>
            <textarea
              rows={2}
              className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
              placeholder="e.g. h1, .cta-button"
              value={highlightInput}
              onChange={(e) => {
                const value = e.target.value;
                setHighlightInput(value);
                handleSelectorListChange(value, 'highlightSelectors');
              }}
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <label className="text-xs text-gray-400 flex items-center gap-2">
                <Palette size={12} className="text-sky-300" /> Highlight color
              </label>
              <input
                type="text"
                placeholder="#00E5FF"
                className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={data.highlightColor ?? ''}
                onChange={(e) => updateNodeData({ highlightColor: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <label className="text-xs text-gray-400 flex items-center gap-2">
                <Wand2 size={12} className="text-purple-300" /> Padding (px)
              </label>
              <input
                type="number"
                min={0}
                className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={typeof data.highlightPadding === 'number' ? data.highlightPadding : ''}
                onChange={(e) => {
                  const raw = e.target.value;
                  const parsed = Number.parseInt(raw, 10);
                  updateNodeData({ highlightPadding: raw === '' || Number.isNaN(parsed) ? undefined : parsed });
                }}
              />
            </div>
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between gap-2">
              <label className="text-xs text-gray-400 flex items-center gap-2">
                <Droplet size={12} className="text-emerald-300" /> Mask selectors
              </label>
              <span className="text-[10px] text-gray-500">hide sensitive areas</span>
            </div>
            <textarea
              rows={2}
              className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
              placeholder="e.g. .ads, .cookie-banner"
              value={maskInput}
              onChange={(e) => {
                const value = e.target.value;
                setMaskInput(value);
                handleSelectorListChange(value, 'maskSelectors');
              }}
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <label className="text-xs text-gray-400 flex items-center gap-2">
                <Droplet size={12} className="text-emerald-300" /> Mask opacity
              </label>
              <input
                type="number"
                min={0}
                max={1}
                step={0.05}
                className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={typeof data.maskOpacity === 'number' ? data.maskOpacity : ''}
                onChange={(e) => {
                  const raw = e.target.value;
                  const parsed = Number.parseFloat(raw);
                  const clamped = Number.isNaN(parsed) ? undefined : Math.min(Math.max(parsed, 0), 1);
                  updateNodeData({ maskOpacity: raw === '' ? undefined : clamped });
                }}
              />
            </div>
            <div className="space-y-1">
              <label className="text-xs text-gray-400 flex items-center gap-2">
                <ZoomIn size={12} className="text-pink-300" /> Zoom factor
              </label>
              <input
                type="number"
                min={0}
                step={0.05}
                className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={typeof data.zoomFactor === 'number' ? data.zoomFactor : ''}
                onChange={(e) => {
                  const raw = e.target.value;
                  const parsed = Number.parseFloat(raw);
                  updateNodeData({ zoomFactor: raw === '' || Number.isNaN(parsed) ? undefined : parsed });
                }}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <label className="text-xs text-gray-400 flex items-center gap-2">
                <Globe size={12} className="text-blue-300" /> Viewport width
              </label>
              <input
                type="number"
                min={320}
                className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={typeof data.viewportWidth === 'number' ? data.viewportWidth : ''}
                onChange={(e) => {
                  const raw = e.target.value;
                  const parsed = Number.parseInt(raw, 10);
                  updateNodeData({ viewportWidth: raw === '' || Number.isNaN(parsed) ? undefined : parsed });
                }}
              />
            </div>
            <div className="space-y-1">
              <label className="text-xs text-gray-400 flex items-center gap-2">
                <Globe size={12} className="text-blue-300" /> Viewport height
              </label>
              <input
                type="number"
                min={320}
                className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={typeof data.viewportHeight === 'number' ? data.viewportHeight : ''}
                onChange={(e) => {
                  const raw = e.target.value;
                  const parsed = Number.parseInt(raw, 10);
                  updateNodeData({ viewportHeight: raw === '' || Number.isNaN(parsed) ? undefined : parsed });
                }}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <label className="text-xs text-gray-400 flex items-center gap-2">
                <Palette size={12} className="text-sky-300" /> Page background
              </label>
              <input
                type="text"
                placeholder="#0f172a or linear-gradient(...)"
                className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={data.background ?? ''}
                onChange={(e) => updateNodeData({ background: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <label className="text-xs text-gray-400 flex items-center gap-2">
                <Timer size={12} className="text-amber-300" /> Wait before capture (ms)
              </label>
              <input
                type="number"
                min={0}
                className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={typeof data.waitForMs === 'number' ? data.waitForMs : ''}
                onChange={(e) => {
                  const raw = e.target.value;
                  const parsed = Number.parseInt(raw, 10);
                  updateNodeData({ waitForMs: raw === '' || Number.isNaN(parsed) ? undefined : parsed });
                }}
              />
            </div>
          </div>
        </div>
      )}
      
      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(ScreenshotNode);
