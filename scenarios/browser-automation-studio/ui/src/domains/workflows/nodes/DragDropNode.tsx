import { memo, FC, useEffect, useMemo, useRef } from 'react';
import type { NodeProps } from 'reactflow';
import { Hand } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import {
  useSyncedString,
  useSyncedNumber,
  textInputHandler,
  numberInputHandler,
} from '@hooks/useSyncedField';
import ResiliencePanel from './ResiliencePanel';
import BaseNode from './BaseNode';
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
    if (steps.value <= 5) {
      return 'Quick snap';
    }
    if (steps.value <= 20) {
      return 'Smooth glide';
    }
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
          <div>
            <label className="text-gray-400 block mb-1">Drag from selector</label>
            <input
              type="text"
              value={sourceSelector.value}
              placeholder=".kanban-card:first-child"
              onChange={textInputHandler(sourceSelector.setValue)}
              onBlur={sourceSelector.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Drop target selector</label>
            <input
              type="text"
              value={targetSelector.value}
              placeholder=".drop-target"
              onChange={textInputHandler(targetSelector.setValue)}
              onBlur={targetSelector.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="text-gray-400 block mb-1">Hold before drag (ms)</label>
              <input
                type="number"
                min={0}
                value={holdMs.value}
                onChange={numberInputHandler(holdMs.setValue)}
                onBlur={holdMs.commit}
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
              <p className="text-gray-500 mt-1">Ensures press/hold listeners fire.</p>
            </div>
            <div>
              <label className="text-gray-400 block mb-1">Pointer steps</label>
              <input
                type="number"
                min={MIN_STEPS}
                max={MAX_STEPS}
                value={steps.value}
                onChange={numberInputHandler(steps.setValue)}
                onBlur={steps.commit}
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
              <p className="text-gray-500 mt-1">{movementSummary}</p>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="text-gray-400 block mb-1">Drag duration (ms)</label>
              <input
                type="number"
                min={MIN_DURATION}
                max={MAX_DURATION}
                value={durationMs.value}
                onChange={numberInputHandler(durationMs.setValue)}
                onBlur={durationMs.commit}
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
              <p className="text-gray-500 mt-1">Controls how slow the card travels.</p>
            </div>
            <div>
              <label className="text-gray-400 block mb-1">Timeout (ms)</label>
              <input
                type="number"
                min={MIN_TIMEOUT}
                value={timeoutMs.value}
                onChange={numberInputHandler(timeoutMs.setValue)}
                onBlur={timeoutMs.commit}
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
              <p className="text-gray-500 mt-1">Waits for elements to appear.</p>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="text-gray-400 block mb-1">Offset X (px)</label>
              <input
                type="number"
                min={MIN_OFFSET}
                max={MAX_OFFSET}
                value={offsetX.value}
                onChange={numberInputHandler(offsetX.setValue)}
                onBlur={offsetX.commit}
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
            </div>
            <div>
              <label className="text-gray-400 block mb-1">Offset Y (px)</label>
              <input
                type="number"
                min={MIN_OFFSET}
                max={MAX_OFFSET}
                value={offsetY.value}
                onChange={numberInputHandler(offsetY.setValue)}
                onBlur={offsetY.commit}
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="text-gray-400 block mb-1">Post-drop wait (ms)</label>
              <input
                type="number"
                min={0}
                value={waitForMs.value}
                onChange={numberInputHandler(waitForMs.setValue)}
                onBlur={waitForMs.commit}
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
              <p className="text-gray-500 mt-1">Gives UI time to reorder lists.</p>
            </div>
            <div className="text-gray-500 flex items-center">
              <p>
                Use offsets when the drop target expects the cursor inside a sub-region (e.g. header vs body).
              </p>
            </div>
          </div>
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
