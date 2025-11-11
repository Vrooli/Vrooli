import { memo, FC, useCallback, useEffect, useMemo, useState } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { Crosshair, EyeOff, Target } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import ElementPickerModal from '../ElementPickerModal';
import { useUpstreamUrl } from '../../hooks/useUpstreamUrl';
import type { ElementInfo } from '../../types/elements';

type FocusBlurMode = 'focus' | 'blur';

interface FocusBlurNodeProps extends NodeProps {
  mode: FocusBlurMode;
  label: string;
  description: string;
  helperText: string;
  icon: LucideIcon;
}

const deriveTimeout = (value: unknown): number => {
  const numeric = Number(value);
  return Number.isFinite(numeric) && numeric > 0 ? Math.round(numeric) : 5000;
};

const deriveWait = (value: unknown, mode: FocusBlurMode): number => {
  const numeric = Number(value);
  if (Number.isFinite(numeric) && numeric >= 0) {
    return Math.round(numeric);
  }
  return mode === 'blur' ? 150 : 0;
};

const summarizeElement = (element: ElementInfo | null): string => {
  if (!element) {
    return '';
  }
  const tag = typeof element.tagName === 'string' ? element.tagName.toLowerCase() : '';
  const idAttr = element.attributes?.id ? `#${element.attributes.id}` : '';
  const text = typeof element.text === 'string' ? element.text.trim() : '';
  const snippet = text.length > 0 ? (text.length > 40 ? `${text.slice(0, 40)}…` : text) : '';
  const base = `${tag}${idAttr}`.trim();
  if (base && snippet) {
    return `${base} • ${snippet}`;
  }
  return base || snippet;
};

const FocusBlurNode: FC<FocusBlurNodeProps> = ({ data, selected, id, mode, label, description, helperText, icon: Icon }) => {
  const nodeData = (data ?? {}) as Record<string, unknown>;
  const { getNodes, setNodes } = useReactFlow();
  const upstreamUrl = useUpstreamUrl(id);

  const storedUrl = typeof nodeData.url === 'string' ? nodeData.url : '';
  const [urlDraft, setUrlDraft] = useState<string>(storedUrl);
  const [selector, setSelector] = useState<string>(() => typeof nodeData.selector === 'string' ? nodeData.selector : '');
  const [timeoutMs, setTimeoutMs] = useState<number>(() => deriveTimeout(nodeData.timeoutMs));
  const [waitForMs, setWaitForMs] = useState<number>(() => deriveWait(nodeData.waitForMs, mode));
  const [showPicker, setShowPicker] = useState(false);
  const [selectedElementInfo, setSelectedElementInfo] = useState<ElementInfo | null>(() =>
    nodeData.elementInfo ? (nodeData.elementInfo as ElementInfo) : null
  );

  useEffect(() => {
    setSelector(typeof nodeData.selector === 'string' ? nodeData.selector : '');
  }, [nodeData.selector]);

  useEffect(() => {
    setTimeoutMs(deriveTimeout(nodeData.timeoutMs));
  }, [nodeData.timeoutMs]);

  useEffect(() => {
    setWaitForMs(deriveWait(nodeData.waitForMs, mode));
  }, [nodeData.waitForMs, mode]);

  useEffect(() => {
    setUrlDraft(storedUrl);
  }, [storedUrl]);

  useEffect(() => {
    setSelectedElementInfo(nodeData.elementInfo ? (nodeData.elementInfo as ElementInfo) : null);
  }, [nodeData.elementInfo]);

  const effectiveUrl = useMemo(() => {
    const trimmed = storedUrl.trim();
    if (trimmed.length > 0) {
      return trimmed;
    }
    if (upstreamUrl && upstreamUrl.trim().length > 0) {
      return upstreamUrl.trim();
    }
    return '';
  }, [storedUrl, upstreamUrl]);

  const elementSummary = useMemo(() => summarizeElement(selectedElementInfo), [selectedElementInfo]);
  const accentClass = mode === 'focus' ? 'text-emerald-300' : 'text-amber-300';
  const canPickElement = effectiveUrl.length > 0;

  const updateNodeData = useCallback((updates: Record<string, unknown>) => {
    const nodes = getNodes();
    setNodes(nodes.map((node) => {
      if (node.id !== id) {
        return node;
      }
      const nextData = { ...(node.data ?? {}) } as Record<string, unknown>;
      Object.entries(updates).forEach(([key, value]) => {
        if (value === undefined) {
          delete nextData[key];
        } else {
          nextData[key] = value;
        }
      });
      return { ...node, data: nextData };
    }));
  }, [getNodes, setNodes, id]);

  const handleSelectorBlur = useCallback(() => {
    const trimmed = selector.trim();
    setSelector(trimmed);
    const updates: Record<string, unknown> = { selector: trimmed || undefined };
    if (!trimmed) {
      updates.elementInfo = undefined;
      setSelectedElementInfo(null);
    }
    updateNodeData(updates);
  }, [selector, updateNodeData]);

  const handleTimeoutBlur = useCallback(() => {
    const normalized = Math.max(100, Math.round(timeoutMs) || 100);
    setTimeoutMs(normalized);
    updateNodeData({ timeoutMs: normalized });
  }, [timeoutMs, updateNodeData]);

  const handleWaitBlur = useCallback(() => {
    const normalized = Math.max(0, Math.round(waitForMs) || 0);
    setWaitForMs(normalized);
    updateNodeData({ waitForMs: normalized });
  }, [waitForMs, updateNodeData]);

  const handleUrlCommit = useCallback(() => {
    const trimmed = urlDraft.trim();
    setUrlDraft(trimmed);
    updateNodeData({ url: trimmed || undefined });
  }, [updateNodeData, urlDraft]);

  const handleElementSelection = useCallback((newSelector: string, elementInfo: ElementInfo) => {
    setSelector(newSelector);
    setSelectedElementInfo(elementInfo);
    updateNodeData({ selector: newSelector, elementInfo });
    setShowPicker(false);
  }, [updateNodeData]);

  return (
    <>
      <div className={`workflow-node ${selected ? 'selected' : ''}`}>
        <Handle type="target" position={Position.Top} className="node-handle" />

        <div className="flex items-start gap-2 mb-3">
          <Icon size={16} className={accentClass} />
          <div>
            <div className="font-semibold text-sm">{label}</div>
            <p className="text-[11px] text-gray-500">{description}</p>
          </div>
        </div>

        <div className="space-y-3 text-xs">
          <div>
            <label className="text-[11px] font-semibold text-gray-400">Page URL (optional)</label>
            <input
              type="text"
              placeholder={upstreamUrl || 'https://example.com'}
              className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              value={urlDraft}
              onChange={(event) => setUrlDraft(event.target.value)}
              onBlur={handleUrlCommit}
            />
            {!storedUrl && upstreamUrl && (
              <p className="text-[10px] text-gray-500 mt-1" title={upstreamUrl}>Inherits {upstreamUrl}</p>
            )}
          </div>

          <div>
            <label className="text-[11px] font-semibold text-gray-400">Target selector</label>
            <div className="flex items-center gap-2 mt-1">
              <input
                type="text"
                className="flex-1 px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={selector}
                onChange={(event) => setSelector(event.target.value)}
                onBlur={handleSelectorBlur}
                placeholder="#email"
              />
              <button
                type="button"
                onClick={() => canPickElement && setShowPicker(true)}
                disabled={!canPickElement}
                className={`p-1.5 rounded border border-gray-700 transition-colors ${
                  canPickElement ? 'bg-flow-bg hover:bg-gray-700 cursor-pointer' : 'bg-flow-bg opacity-50 cursor-not-allowed'
                }`}
                title={canPickElement ? `Pick element from ${effectiveUrl}` : 'Navigate first or set a page URL'}
              >
                <Target size={14} className={canPickElement ? 'text-gray-300' : 'text-gray-600'} />
              </button>
            </div>
            {elementSummary && (
              <p className="text-[10px] text-gray-500 mt-1">Stored element: {elementSummary}</p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="text-[11px] font-semibold text-gray-400">Timeout (ms)</label>
              <input
                type="number"
                min={100}
                className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={timeoutMs}
                onChange={(event) => setTimeoutMs(Number(event.target.value))}
                onBlur={handleTimeoutBlur}
              />
            </div>
            <div>
              <label className="text-[11px] font-semibold text-gray-400">Post-action wait (ms)</label>
              <input
                type="number"
                min={0}
                className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={waitForMs}
                onChange={(event) => setWaitForMs(Number(event.target.value))}
                onBlur={handleWaitBlur}
              />
            </div>
          </div>

          <p className="text-[11px] text-gray-500">{helperText}</p>
        </div>

        <Handle type="source" position={Position.Bottom} className="node-handle" />
      </div>

      {showPicker && canPickElement && (
        <ElementPickerModal
          isOpen={showPicker}
          onClose={() => setShowPicker(false)}
          url={effectiveUrl}
          onSelectElement={handleElementSelection}
          selectedSelector={selector}
        />
      )}
    </>
  );
};

const FocusNodeComponent: FC<NodeProps> = (props) => (
  <FocusBlurNode
    {...props}
    mode="focus"
    label="Focus"
    description="Move keyboard focus to an element before typing or shortcuts."
    helperText="Ensures subsequent keyboard events hit the intended field and triggers onFocus handlers."
    icon={Crosshair}
  />
);

export const BlurNode = memo<FC<NodeProps>>((props) => (
  <FocusBlurNode
    {...props}
    mode="blur"
    label="Blur"
    description="Trigger blur/onChange validation hooks after editing a field."
    helperText="Great for forcing client-side validation, masking, or autosave logic after a Type node."
    icon={EyeOff}
  />
));

export default memo(FocusNodeComponent);
