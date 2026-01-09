import { memo, FC, useMemo } from 'react';
import type { NodeProps } from 'reactflow';
import { Crosshair, EyeOff } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useElementPicker } from '@hooks/useElementPicker';
import { useNodeData } from '@hooks/useNodeData';
import { useUrlInheritance } from '@hooks/useUrlInheritance';
import { useSyncedString, useSyncedNumber } from '@hooks/useSyncedField';
import type { FocusParams, BlurParams } from '@utils/actionBuilder';
import type { ElementInfo } from '@/types/elements';
import BaseNode from './BaseNode';
import { NodeUrlField, NodeSelectorField, TimeoutFields } from './fields';

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
  // URL inheritance for element picker
  const { effectiveUrl } = useUrlInheritance(id);

  // Node data hook for UI-specific fields
  const { getValue, updateData } = useNodeData(id);

  // V2 Native: Use action params as source of truth
  const { params, updateParams } = useActionParams<FocusParams | BlurParams>(id);

  // Element picker binding
  const elementPicker = useElementPicker(id);

  // Action params fields using useSyncedField
  const selector = useSyncedString(params?.selector ?? '', {
    onCommit: (v) => {
      updateParams({ selector: v || undefined });
      if (!v) {
        updateData({ elementInfo: undefined });
      }
    },
  });
  const timeoutMs = useSyncedNumber(params?.timeoutMs ?? 5000, {
    min: 100,
    fallback: 5000,
    onCommit: (v) => updateParams({ timeoutMs: v }),
  });

  // UI-specific fields
  const defaultWait = mode === 'blur' ? 150 : 0;
  const waitForMs = useSyncedNumber(getValue<number>('waitForMs') ?? defaultWait, {
    min: 0,
    onCommit: (v) => updateData({ waitForMs: v }),
  });

  const selectedElementInfo = getValue<ElementInfo>('elementInfo') ?? null;
  const elementSummary = useMemo(() => summarizeElement(selectedElementInfo), [selectedElementInfo]);
  const accentClass = mode === 'focus' ? 'text-emerald-300' : 'text-amber-300';

  return (
    <BaseNode selected={selected} icon={Icon} iconClassName={accentClass} title={label}>
      <p className="text-[11px] text-gray-500 mb-3">{description}</p>

      <div className="space-y-3 text-xs">
        <NodeUrlField nodeId={id} />

        <div>
          <NodeSelectorField
            field={selector}
            effectiveUrl={effectiveUrl}
            onElementSelect={elementPicker.onSelect}
            label="Target selector"
            placeholder="#email"
            className=""
          />
          {elementSummary && (
            <p className="text-[10px] text-gray-500 mt-1">Stored element: {elementSummary}</p>
          )}
        </div>

        <TimeoutFields timeoutMs={timeoutMs} waitForMs={waitForMs} />

        <p className="text-[11px] text-gray-500">{helperText}</p>
      </div>
    </BaseNode>
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
