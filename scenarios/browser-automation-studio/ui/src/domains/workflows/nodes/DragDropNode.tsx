import { memo, FC, useEffect, useMemo, useRef } from 'react';
import type { NodeProps } from 'reactflow';
import { Hand } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import { useSyncedString, useSyncedNumber } from '@hooks/useSyncedField';
import ResiliencePanel from './ResiliencePanel';
import BaseNode from './BaseNode';
import { NodeTextField, NodeNumberField, FieldRow } from './fields';
import type { ResilienceSettings } from '@/types/workflow';

// DragDropParams interface for V2 native action params
interface DragDropParams {
  sourceSelector?: string;
  targetSelector?: string;
  holdMs?: number;
  steps?: number;
  durationMs?: number;
  offsetX?: number;
  offsetY?: number;
  timeoutMs?: number;
  waitForMs?: number;
}

const MIN_STEPS = 1;
const MAX_STEPS = 60;
const MIN_DURATION = 50;
const MAX_DURATION = 20000;
const MIN_OFFSET = -5000;
const MAX_OFFSET = 5000;
const MIN_TIMEOUT = 500;

const DragDropNode: FC<NodeProps> = ({ selected, id }) => {
  const nodeRef = useRef<HTMLDivElement>(null);
  const { params, updateParams } = useActionParams<DragDropParams>(id);
  const { getValue, updateData } = useNodeData(id);

  // String fields
  const sourceSelector = useSyncedString(params?.sourceSelector ?? '', {
    onCommit: (v) => updateParams({ sourceSelector: v || undefined }),
  });
  const targetSelector = useSyncedString(params?.targetSelector ?? '', {
    onCommit: (v) => updateParams({ targetSelector: v || undefined }),
  });

  // Number fields
  const holdMs = useSyncedNumber(params?.holdMs ?? 150, {
    min: 0,
    onCommit: (v) => updateParams({ holdMs: v }),
  });
  const steps = useSyncedNumber(params?.steps ?? 18, {
    min: MIN_STEPS,
    max: MAX_STEPS,
    fallback: 18,
    onCommit: (v) => updateParams({ steps: v }),
  });
  const durationMs = useSyncedNumber(params?.durationMs ?? 600, {
    min: MIN_DURATION,
    max: MAX_DURATION,
    fallback: 600,
    onCommit: (v) => updateParams({ durationMs: v }),
  });
  const offsetX = useSyncedNumber(params?.offsetX ?? 0, {
    min: MIN_OFFSET,
    max: MAX_OFFSET,
    onCommit: (v) => updateParams({ offsetX: v }),
  });
  const offsetY = useSyncedNumber(params?.offsetY ?? 0, {
    min: MIN_OFFSET,
    max: MAX_OFFSET,
    onCommit: (v) => updateParams({ offsetY: v }),
  });
  const timeoutMs = useSyncedNumber(params?.timeoutMs ?? 30000, {
    min: MIN_TIMEOUT,
    fallback: 30000,
    onCommit: (v) => updateParams({ timeoutMs: v }),
  });
  const waitForMs = useSyncedNumber(params?.waitForMs ?? 0, {
    min: 0,
    onCommit: (v) => updateParams({ waitForMs: v || undefined }),
  });

  const resilienceConfig = getValue<ResilienceSettings>('resilience');

  const movementSummary = useMemo(() => {
    if (steps.value <= 5) return 'Quick snap';
    if (steps.value <= 20) return 'Smooth glide';
    return 'Very smooth';
  }, [steps.value]);

  // Add data-type attribute to React Flow wrapper div for test automation
  useEffect(() => {
    if (nodeRef.current) {
      const reactFlowNode = nodeRef.current.closest('.react-flow__node');
      if (reactFlowNode) {
        reactFlowNode.setAttribute('data-type', 'dragDrop');
      }
    }
  }, []);

  return (
    <div ref={nodeRef}>
      <BaseNode selected={selected} icon={Hand} iconClassName="text-pink-300" title="Drag & Drop">
        <div className="space-y-3 text-xs">
          <NodeTextField
            field={sourceSelector}
            label="Drag from selector"
            placeholder=".kanban-card:first-child"
          />

          <NodeTextField
            field={targetSelector}
            label="Drop target selector"
            placeholder=".drop-target"
          />

          <FieldRow>
            <NodeNumberField
              field={holdMs}
              label="Hold before drag (ms)"
              min={0}
              description="Ensures press/hold listeners fire."
            />
            <NodeNumberField
              field={steps}
              label="Pointer steps"
              min={MIN_STEPS}
              max={MAX_STEPS}
              description={movementSummary}
            />
          </FieldRow>

          <FieldRow>
            <NodeNumberField
              field={durationMs}
              label="Drag duration (ms)"
              min={MIN_DURATION}
              max={MAX_DURATION}
              description="Controls how slow the card travels."
            />
            <NodeNumberField
              field={timeoutMs}
              label="Timeout (ms)"
              min={MIN_TIMEOUT}
              description="Waits for elements to appear."
            />
          </FieldRow>

          <FieldRow>
            <NodeNumberField
              field={offsetX}
              label="Offset X (px)"
              min={MIN_OFFSET}
              max={MAX_OFFSET}
            />
            <NodeNumberField
              field={offsetY}
              label="Offset Y (px)"
              min={MIN_OFFSET}
              max={MAX_OFFSET}
            />
          </FieldRow>

          <FieldRow>
            <NodeNumberField
              field={waitForMs}
              label="Post-drop wait (ms)"
              min={0}
              description="Gives UI time to reorder lists."
            />
            <div className="text-gray-500 flex items-center text-xs">
              <p>Use offsets when the drop target expects the cursor inside a sub-region.</p>
            </div>
          </FieldRow>
        </div>

        <ResiliencePanel
          value={resilienceConfig}
          onChange={(next) => updateData({ resilience: next ?? null })}
        />
      </BaseNode>
    </div>
  );
};

export default memo(DragDropNode);
