import { FC, memo } from 'react';
import type { NodeProps } from 'reactflow';
import { Clock, Globe } from 'lucide-react';
import { useUpstreamUrl } from '@hooks/useUpstreamUrl';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import {
  useSyncedString,
  useSyncedNumber,
  useSyncedSelect,
} from '@hooks/useSyncedField';
import type { WaitParams } from '@utils/actionBuilder';
import BaseNode from './BaseNode';
import { NodeTextField, NodeNumberField, NodeSelectField } from './fields';

const WAIT_OPTIONS = [
  { label: 'Wait for time', value: 'time' },
  { label: 'Wait for element', value: 'element' },
  { label: 'Wait for navigation', value: 'navigation' },
];

const WaitNode: FC<NodeProps> = ({ selected, id }) => {
  const upstreamUrl = useUpstreamUrl(id);
  const { getValue, updateData } = useNodeData(id);
  const { params, updateParams } = useActionParams<WaitParams>(id);

  // waitType is UI-specific (controls which input to show), stored in node.data
  const waitType = useSyncedSelect(getValue<string>('waitType') ?? 'time', {
    onCommit: (v) => {
      updateData({ waitType: v });
      // Update the state in action params
      updateParams({ state: v === 'navigation' ? 'navigation' : undefined });
    },
  });

  // Action params fields
  const duration = useSyncedNumber(params?.durationMs ?? 1000, {
    min: 0,
    fallback: 1000,
    onCommit: (v) => updateParams({ durationMs: v }),
  });
  const selector = useSyncedString(params?.selector ?? '', {
    onCommit: (v) => updateParams({ selector: v || undefined }),
  });

  return (
    <BaseNode selected={selected} icon={Clock} iconClassName="text-gray-400" title="Wait">
      {upstreamUrl && (
        <div className="flex items-center gap-1 mb-2 p-1 bg-flow-bg/50 rounded text-xs border border-gray-700">
          <Globe size={12} className="text-blue-400 flex-shrink-0" />
          <span className="text-gray-400 truncate" title={upstreamUrl}>
            {upstreamUrl}
          </span>
        </div>
      )}

      <div className="space-y-2">
        <NodeSelectField field={waitType} label="" options={WAIT_OPTIONS} />

        {waitType.value === 'time' ? (
          <NodeNumberField field={duration} label="" placeholder="Milliseconds..." min={0} />
        ) : (
          <NodeTextField field={selector} label="" placeholder="CSS Selector..." />
        )}
      </div>
    </BaseNode>
  );
};

export default memo(WaitNode);
