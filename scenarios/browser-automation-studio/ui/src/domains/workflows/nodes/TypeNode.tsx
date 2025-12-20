import { memo, FC } from 'react';
import type { NodeProps } from 'reactflow';
import { Keyboard } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useElementPicker } from '@hooks/useElementPicker';
import { useResiliencePanelProps } from '@hooks/useResiliencePanel';
import { useUrlInheritance } from '@hooks/useUrlInheritance';
import { useSyncedString, textInputHandler } from '@hooks/useSyncedField';
import type { InputParams } from '@utils/actionBuilder';
import BaseNode from './BaseNode';
import ResiliencePanel from './ResiliencePanel';
import { NodeUrlField, NodeSelectorField } from './fields';

const TypeNode: FC<NodeProps> = ({ selected, id }) => {
  // URL inheritance for element picker
  const { effectiveUrl } = useUrlInheritance(id);

  // V2 Native: Use action params as source of truth
  const { params, updateParams } = useActionParams<InputParams>(id);

  // Resilience panel binding
  const resilience = useResiliencePanelProps(id);

  // Element picker binding
  const elementPicker = useElementPicker(id);

  // Action params fields using useSyncedField
  const selector = useSyncedString(params?.selector ?? '', {
    onCommit: (v) => updateParams({ selector: v || undefined }),
  });
  const text = useSyncedString(params?.value ?? '', {
    trim: false, // Don't trim text input
    onCommit: (v) => updateParams({ value: v }),
  });

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
        onElementSelect={elementPicker.onSelect}
      />

      <textarea
        placeholder="Text to type..."
        className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none resize-none"
        rows={2}
        value={text.value}
        onChange={textInputHandler(text.setValue)}
        onBlur={text.commit}
      />

      <ResiliencePanel {...resilience} />
    </BaseNode>
  );
};

export default memo(TypeNode);
