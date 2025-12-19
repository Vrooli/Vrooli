import { FC, memo, useMemo } from 'react';
import type { NodeProps } from 'reactflow';
import { MousePointer } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import { useSyncedString, useSyncedNumber } from '@hooks/useSyncedField';
import type { HoverParams } from '@utils/actionBuilder';
import type { ResilienceSettings } from '@/types/workflow';
import BaseNode from './BaseNode';
import ResiliencePanel from './ResiliencePanel';
import { NodeTextField, NodeNumberField, FieldRow } from './fields';

const MIN_STEPS = 1;
const MAX_STEPS = 50;
const DEFAULT_STEPS = 10;
const MIN_DURATION = 50;
const MAX_DURATION = 10000;
const DEFAULT_DURATION = 350;

const HoverNode: FC<NodeProps> = ({ selected, id }) => {
  const { getValue, updateData } = useNodeData(id);
  const { params, updateParams } = useActionParams<HoverParams>(id);

  // Action params fields
  const selector = useSyncedString(params?.selector ?? '', {
    onCommit: (v) => updateParams({ selector: v || undefined }),
  });
  const timeoutMs = useSyncedNumber(params?.timeoutMs ?? 5000, {
    min: 100,
    fallback: 5000,
    onCommit: (v) => updateParams({ timeoutMs: v }),
  });

  // UI-specific fields (stored in node.data for now)
  const waitForMs = useSyncedNumber(getValue<number>('waitForMs') ?? 0, {
    min: 0,
    onCommit: (v) => updateData({ waitForMs: v }),
  });
  const steps = useSyncedNumber(getValue<number>('steps') ?? DEFAULT_STEPS, {
    min: MIN_STEPS,
    max: MAX_STEPS,
    fallback: DEFAULT_STEPS,
    onCommit: (v) => updateData({ steps: v }),
  });
  const durationMs = useSyncedNumber(getValue<number>('durationMs') ?? DEFAULT_DURATION, {
    min: MIN_DURATION,
    max: MAX_DURATION,
    fallback: DEFAULT_DURATION,
    onCommit: (v) => updateData({ durationMs: v }),
  });

  const resilienceConfig = getValue<ResilienceSettings>('resilience');

  const stepsHint = useMemo(() => {
    if (steps.value <= 3) return 'Instant';
    if (steps.value <= 10) return 'Smooth';
    return 'Very smooth';
  }, [steps.value]);

  return (
    <BaseNode selected={selected} icon={MousePointer} iconClassName="text-sky-300" title="Hover">
      <div className="space-y-3 text-xs">
        <NodeTextField
          field={selector}
          label="Target selector"
          placeholder=".menu-item > button"
        />

        <FieldRow>
          <NodeNumberField field={timeoutMs} label="Timeout (ms)" min={100} />
          <NodeNumberField field={waitForMs} label="Post-hover wait (ms)" min={0} />
        </FieldRow>

        <FieldRow>
          <NodeNumberField
            field={steps}
            label="Movement steps"
            min={MIN_STEPS}
            max={MAX_STEPS}
            description={`${stepsHint} cursor glide`}
          />
          <NodeNumberField
            field={durationMs}
            label="Duration (ms)"
            min={MIN_DURATION}
            max={MAX_DURATION}
            description="Controls how long the pointer glides."
          />
        </FieldRow>

        <p className="text-gray-500">
          Hover nodes move the cursor without clicking so menus, tooltips, and other hover-only UI
          remain open for following nodes.
        </p>
      </div>

      <ResiliencePanel
        value={resilienceConfig}
        onChange={(next) => updateData({ resilience: next ?? null })}
      />
    </BaseNode>
  );
};

export default memo(HoverNode);
