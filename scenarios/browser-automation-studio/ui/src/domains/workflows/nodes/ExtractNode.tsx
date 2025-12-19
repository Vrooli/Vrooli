import { memo, FC, useCallback } from 'react';
import type { NodeProps } from 'reactflow';
import { Database } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import { useUrlInheritance } from '@hooks/useUrlInheritance';
import { useSyncedString, useSyncedSelect } from '@hooks/useSyncedField';
import type { ElementInfo } from '@/types/elements';
import BaseNode from './BaseNode';
import { NodeTextField, NodeSelectField, NodeUrlField, NodeSelectorField } from './fields';

// ExtractParams interface for V2 native action params
interface ExtractParams {
  selector?: string;
  extractType?: string;
  attribute?: string;
}

const EXTRACT_TYPE_OPTIONS = [
  { value: 'text', label: 'Text Content' },
  { value: 'attribute', label: 'Attribute' },
  { value: 'html', label: 'Inner HTML' },
  { value: 'value', label: 'Input Value' },
];

const ExtractNode: FC<NodeProps> = ({ selected, id }) => {
  // URL inheritance for element picker
  const { effectiveUrl } = useUrlInheritance(id);

  // Node data hook for UI-specific fields
  const { updateData } = useNodeData(id);

  // V2 Native: Use action params as source of truth
  const { params, updateParams } = useActionParams<ExtractParams>(id);

  // Action params fields using useSyncedField
  const selector = useSyncedString(params?.selector ?? '', {
    onCommit: (v) => updateParams({ selector: v || undefined }),
  });
  const extractType = useSyncedSelect(params?.extractType ?? 'text', {
    onCommit: (v) => updateParams({ extractType: v }),
  });
  const attribute = useSyncedString(params?.attribute ?? '', {
    onCommit: (v) => updateParams({ attribute: v || undefined }),
  });

  const handleElementSelection = useCallback(
    (newSelector: string, elementInfo: ElementInfo) => {
      updateParams({ selector: newSelector });
      updateData({ elementInfo });
    },
    [updateParams, updateData],
  );

  return (
    <BaseNode selected={selected} icon={Database} iconClassName="text-pink-400" title="Extract Data">
      <NodeUrlField nodeId={id} />

      <NodeSelectorField
        field={selector}
        effectiveUrl={effectiveUrl}
        onElementSelect={handleElementSelection}
      />

      <NodeSelectField field={extractType} label="" options={EXTRACT_TYPE_OPTIONS} className="mb-2" />

      {extractType.value === 'attribute' && (
        <NodeTextField field={attribute} label="" placeholder="Attribute name..." />
      )}
    </BaseNode>
  );
};

export default memo(ExtractNode);
