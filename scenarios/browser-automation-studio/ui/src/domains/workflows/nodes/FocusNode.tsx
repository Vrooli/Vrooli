import { memo, FC, useEffect, useMemo, useState } from 'react';
import type { NodeProps } from 'reactflow';
import { Crosshair, EyeOff, Target } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { ElementPickerModal } from '../components';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import { useUrlInheritance } from '@hooks/useUrlInheritance';
import type { FocusParams, BlurParams } from '@utils/actionBuilder';
import type { ElementInfo } from '@/types/elements';
import BaseNode from './BaseNode';

type FocusBlurMode = 'focus' | 'blur';

interface FocusBlurNodeProps extends NodeProps {
  mode: FocusBlurMode;
  label: string;
  description: string;
  helperText: string;
  icon: LucideIcon;
}

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

const FocusBlurNode: FC<FocusBlurNodeProps> = ({
  selected,
  id,
  mode,
  label,
  description,
  helperText,
  icon: Icon,
}) => {
  // URL inheritance hook handles URL state and handlers
  const {
    urlDraft,
    setUrlDraft,
    effectiveUrl,
    upstreamUrl,
    commitUrl,
  } = useUrlInheritance(id);

  // Node data hook for UI-specific fields
  const { getValue, updateData } = useNodeData(id);

  // V2 Native: Use action params as source of truth
  // FocusParams and BlurParams have same shape (selector, timeoutMs)
  const { params, updateParams } = useActionParams<FocusParams | BlurParams>(id);

  // Local state - action params fields
  const [selector, setSelector] = useState<string>(params?.selector ?? '');
  const [timeoutMs, setTimeoutMs] = useState<number>(params?.timeoutMs ?? 5000);

  // Local state - UI-specific fields
  const [waitForMs, setWaitForMs] = useState<number>(() => {
    const value = getValue<number>('waitForMs');
    if (value !== undefined && value >= 0) return value;
    return mode === 'blur' ? 150 : 0;
  });
  const [showPicker, setShowPicker] = useState(false);
  const [selectedElementInfo, setSelectedElementInfo] = useState<ElementInfo | null>(
    getValue<ElementInfo>('elementInfo') ?? null
  );

  // Sync action params fields
  useEffect(() => {
    setSelector(params?.selector ?? '');
  }, [params?.selector]);

  useEffect(() => {
    setTimeoutMs(params?.timeoutMs ?? 5000);
  }, [params?.timeoutMs]);

  // Sync node.data fields
  useEffect(() => {
    const value = getValue<number>('waitForMs');
    if (value !== undefined && value >= 0) {
      setWaitForMs(value);
    }
  }, [getValue]);

  useEffect(() => {
    setSelectedElementInfo(getValue<ElementInfo>('elementInfo') ?? null);
  }, [getValue]);

  const elementSummary = useMemo(() => summarizeElement(selectedElementInfo), [selectedElementInfo]);
  const accentClass = mode === 'focus' ? 'text-emerald-300' : 'text-amber-300';
  const canPickElement = (effectiveUrl?.length ?? 0) > 0;

  const handleSelectorBlur = () => {
    const trimmed = selector.trim();
    setSelector(trimmed);
    updateParams({ selector: trimmed || undefined });
    if (!trimmed) {
      updateData({ elementInfo: undefined });
      setSelectedElementInfo(null);
    }
  };

  const handleTimeoutBlur = () => {
    const normalized = Math.max(100, Math.round(timeoutMs) || 100);
    setTimeoutMs(normalized);
    updateParams({ timeoutMs: normalized });
  };

  const handleWaitBlur = () => {
    const normalized = Math.max(0, Math.round(waitForMs) || 0);
    setWaitForMs(normalized);
    updateData({ waitForMs: normalized });
  };

  const handleElementSelection = (newSelector: string, elementInfo: ElementInfo) => {
    setSelector(newSelector);
    setSelectedElementInfo(elementInfo);
    updateParams({ selector: newSelector });
    updateData({ elementInfo });
    setShowPicker(false);
  };

  return (
    <>
      <BaseNode selected={selected} icon={Icon} iconClassName={accentClass} title={label}>
        <p className="text-[11px] text-gray-500 mb-3">{description}</p>

        <div className="space-y-3 text-xs">
          <div>
            <label className="text-[11px] font-semibold text-gray-400">Page URL (optional)</label>
            <input
              type="text"
              placeholder={upstreamUrl || 'https://example.com'}
              className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              value={urlDraft}
              onChange={(event) => setUrlDraft(event.target.value)}
              onBlur={commitUrl}
            />
            {!urlDraft && upstreamUrl && (
              <p className="text-[10px] text-gray-500 mt-1" title={upstreamUrl}>
                Inherits {upstreamUrl}
              </p>
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
                  canPickElement
                    ? 'bg-flow-bg hover:bg-gray-700 cursor-pointer'
                    : 'bg-flow-bg opacity-50 cursor-not-allowed'
                }`}
                title={
                  canPickElement
                    ? `Pick element from ${effectiveUrl}`
                    : 'Navigate first or set a page URL'
                }
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
      </BaseNode>

      {showPicker && canPickElement && effectiveUrl && (
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

export const BlurNode: FC<NodeProps> = memo((props) => (
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
