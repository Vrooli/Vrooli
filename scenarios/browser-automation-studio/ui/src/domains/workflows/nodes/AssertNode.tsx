import { FC, memo } from 'react';
import type { NodeProps } from 'reactflow';
import { CheckCircle, Globe } from 'lucide-react';
import { useUpstreamUrl } from '@hooks/useUpstreamUrl';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import { useSyncedString, useSyncedBoolean, useSyncedSelect } from '@hooks/useSyncedField';
import type { AssertParams } from '@utils/actionBuilder';
import BaseNode from './BaseNode';
import { NodeTextField, NodeSelectField, NodeCheckbox } from './fields';

const assertModes = [
  { value: 'exists', label: 'Element Exists' },
  { value: 'not_exists', label: 'Element Does Not Exist' },
  { value: 'visible', label: 'Element Visible' },
  { value: 'text_equals', label: 'Text Equals' },
  { value: 'text_contains', label: 'Text Contains' },
  { value: 'attribute_equals', label: 'Attribute Equals' },
];

const AssertNode: FC<NodeProps> = ({ selected, id }) => {
  const upstreamUrl = useUpstreamUrl(id);
  const { getValue, updateData } = useNodeData(id);
  const { params, updateParams } = useActionParams<AssertParams>(id);

  // Action params fields
  const selector = useSyncedString(params?.selector ?? '', {
    onCommit: (v) => updateParams({ selector: v || undefined }),
  });
  const mode = useSyncedSelect(params?.mode ?? 'exists', {
    onCommit: (v) => updateParams({ mode: v }),
  });
  const expectedValue = useSyncedString(
    typeof params?.expected === 'string' ? params.expected : '',
    { onCommit: (v) => updateParams({ expected: v || undefined }) },
  );
  const attributeName = useSyncedString(params?.attributeName ?? '', {
    onCommit: (v) => updateParams({ attributeName: v || undefined }),
  });
  const caseSensitive = useSyncedBoolean(params?.caseSensitive ?? false, {
    onCommit: (v) => updateParams({ caseSensitive: v }),
  });

  // UI-specific fields (stored in node.data)
  const label = useSyncedString(getValue<string>('label') ?? '', {
    onCommit: (v) => updateData({ label: v || undefined }),
  });
  const continueOnFailure = useSyncedBoolean(getValue<boolean>('continueOnFailure') ?? false, {
    onCommit: (v) => updateData({ continueOnFailure: v }),
  });

  const showExpectedValue =
    mode.value === 'text_equals' || mode.value === 'text_contains' || mode.value === 'attribute_equals';
  const showAttributeName = mode.value === 'attribute_equals';
  const showCaseSensitive = mode.value === 'text_equals' || mode.value === 'text_contains';

  return (
    <BaseNode
      selected={selected}
      icon={CheckCircle}
      iconClassName="text-orange-400"
      title="Assert"
      className="max-w-[280px]"
    >
      {upstreamUrl && (
        <div className="flex items-center gap-1 mb-2 p-1 bg-flow-bg/50 rounded text-xs border border-gray-700">
          <Globe size={12} className="text-blue-400 flex-shrink-0" />
          <span className="text-gray-400 truncate" title={upstreamUrl}>
            {upstreamUrl}
          </span>
        </div>
      )}

      <div className="space-y-2">
        <NodeTextField field={label} label="" placeholder="Label (optional)" />

        <NodeSelectField field={mode} label="" options={assertModes} />

        <NodeTextField field={selector} label="" placeholder="CSS Selector..." />

        {showExpectedValue && (
          <NodeTextField
            field={expectedValue}
            label=""
            placeholder={mode.value === 'attribute_equals' ? 'Expected attribute value...' : 'Expected text...'}
          />
        )}

        {showAttributeName && (
          <NodeTextField field={attributeName} label="" placeholder="Attribute name..." />
        )}

        <NodeCheckbox field={continueOnFailure} label="Continue on failure" />

        {showCaseSensitive && (
          <NodeCheckbox field={caseSensitive} label="Case sensitive" />
        )}
      </div>
    </BaseNode>
  );
};

export default memo(AssertNode);
