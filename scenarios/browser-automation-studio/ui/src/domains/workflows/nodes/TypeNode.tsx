import { memo, FC, useCallback } from 'react';
import type { NodeProps } from 'reactflow';
import { Keyboard } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import { useUrlInheritance } from '@hooks/useUrlInheritance';
import { useSyncedString, textInputHandler } from '@hooks/useSyncedField';
import type { ElementInfo } from '@/types/elements';
import type { InputParams } from '@utils/actionBuilder';
import type { ResilienceSettings } from '@/types/workflow';
import BaseNode from './BaseNode';
import ResiliencePanel from './ResiliencePanel';
import { NodeUrlField, NodeSelectorField } from './fields';

const TypeNode: FC<NodeProps> = ({ selected, id }) => {
  // URL inheritance for element picker
  const { effectiveUrl } = useUrlInheritance(id);

  // Node data hook for non-action fields
  const { updateData, getValue } = useNodeData(id);

  // V2 Native: Use action params as source of truth
  const { params, updateParams } = useActionParams<InputParams>(id);

  // Action params fields using useSyncedField
  const selector = useSyncedString(params?.selector ?? '', {
    onCommit: (v) => updateParams({ selector: v || undefined }),
  });
  const text = useSyncedString(params?.value ?? '', {
    trim: false, // Don't trim text input
    onCommit: (v) => updateParams({ value: v }),
  });

  const handleElementSelection = useCallback(
    (newSelector: string, elementInfo: ElementInfo) => {
      // V2 Native: Update selector in action params
      updateParams({ selector: newSelector });
      // Keep elementInfo in data for now (not part of InputParams proto)
      updateData({ elementInfo });
    },
    [updateParams, updateData],
  );

  const resilienceConfig = getValue<ResilienceSettings>('resilience');

  return (
    <BaseNode
      selected={selected}
      icon={Keyboard}
      iconClassName="text-yellow-400"
      title="Type Text"
    >
      <NodeUrlField nodeId={id} />

      <NodeSelectorField
        field={selector}
        effectiveUrl={effectiveUrl}
        onElementSelect={handleElementSelection}
      />

      <textarea
        placeholder="Text to type..."
        className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none resize-none"
        rows={2}
        value={text.value}
        onChange={textInputHandler(text.setValue)}
        onBlur={text.commit}
      />

      <ResiliencePanel
        value={resilienceConfig}
        onChange={(next) => updateData({ resilience: next ?? null })}
      />
    </BaseNode>
  );
};

export default memo(TypeNode);
