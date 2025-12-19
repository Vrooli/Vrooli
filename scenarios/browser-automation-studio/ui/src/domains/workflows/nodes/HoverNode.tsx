import { FC, memo, useMemo } from 'react';
import type { NodeProps } from 'reactflow';
import { MousePointer } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import {
  useSyncedString,
  useSyncedNumber,
  textInputHandler,
  numberInputHandler,
} from '@hooks/useSyncedField';
import type { HoverParams } from '@utils/actionBuilder';
import type { ResilienceSettings } from '@/types/workflow';
import BaseNode from './BaseNode';
import ResiliencePanel from './ResiliencePanel';

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
    if (steps.value <= 3) {
      return 'Instant';
    }
    if (steps.value <= 10) {
      return 'Smooth';
    }
    return 'Very smooth';
  }, [steps.value]);

  return (
    <BaseNode selected={selected} icon={MousePointer} iconClassName="text-sky-300" title="Hover">
      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">Target selector</label>
          <input
            type="text"
            value={selector.value}
            placeholder=".menu-item > button"
            onChange={textInputHandler(selector.setValue)}
            onBlur={selector.commit}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Timeout (ms)</label>
            <input
              type="number"
              min={100}
              value={timeoutMs.value}
              onChange={numberInputHandler(timeoutMs.setValue)}
              onBlur={timeoutMs.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Post-hover wait (ms)</label>
            <input
              type="number"
              min={0}
              value={waitForMs.value}
              onChange={numberInputHandler(waitForMs.setValue)}
              onBlur={waitForMs.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Movement steps</label>
            <input
              type="number"
              min={MIN_STEPS}
              max={MAX_STEPS}
              value={steps.value}
              onChange={numberInputHandler(steps.setValue)}
              onBlur={steps.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
            <p className="text-gray-500 mt-1">{stepsHint} cursor glide</p>
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Duration (ms)</label>
            <input
              type="number"
              min={MIN_DURATION}
              max={MAX_DURATION}
              value={durationMs.value}
              onChange={numberInputHandler(durationMs.setValue)}
              onBlur={durationMs.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
            <p className="text-gray-500 mt-1">Controls how long the pointer glides.</p>
          </div>
        </div>

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
