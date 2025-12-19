import { FC, memo } from 'react';
import type { NodeProps } from 'reactflow';
import { ScrollText } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import {
  useSyncedString,
  useSyncedNumber,
  useSyncedSelect,
  textInputHandler,
  numberInputHandler,
  selectInputHandler,
} from '@hooks/useSyncedField';
import type { ResilienceSettings } from '@/types/workflow';
import BaseNode from './BaseNode';
import ResiliencePanel from './ResiliencePanel';

// ScrollParams interface for V2 native action params
interface ScrollParams {
  selector?: string;
  x?: number;
  y?: number;
  behavior?: string;
  // UI-specific fields stored in action params for V2 native
  scrollType?: string;
  targetSelector?: string;
  direction?: string;
  amount?: number;
  maxScrolls?: number;
  timeoutMs?: number;
  waitForMs?: number;
}

const MIN_AMOUNT = 10;
const MAX_AMOUNT = 5000;
const DEFAULT_AMOUNT = 400;
const MIN_SCROLLS = 1;
const MAX_SCROLLS = 200;
const DEFAULT_SCROLLS = 12;
const MIN_COORDINATE = -500000;
const MAX_COORDINATE = 500000;
const MIN_TIMEOUT = 100;

const ScrollNode: FC<NodeProps> = ({ selected, id }) => {
  const { getValue, updateData } = useNodeData(id);
  const { params, updateParams } = useActionParams<ScrollParams>(id);

  // Select fields
  const scrollType = useSyncedSelect(params?.scrollType ?? 'page', {
    onCommit: (v) => updateParams({ scrollType: v }),
  });
  const direction = useSyncedSelect(params?.direction ?? 'down', {
    onCommit: (v) => updateParams({ direction: v }),
  });
  const behavior = useSyncedSelect(params?.behavior ?? 'auto', {
    onCommit: (v) => updateParams({ behavior: v }),
  });

  // String fields
  const selector = useSyncedString(params?.selector ?? '', {
    onCommit: (v) => updateParams({ selector: v || undefined }),
  });
  const targetSelector = useSyncedString(params?.targetSelector ?? '', {
    onCommit: (v) => updateParams({ targetSelector: v || undefined }),
  });

  // Number fields
  const x = useSyncedNumber(params?.x ?? 0, {
    min: MIN_COORDINATE,
    max: MAX_COORDINATE,
    onCommit: (v) => updateParams({ x: v }),
  });
  const y = useSyncedNumber(params?.y ?? 0, {
    min: MIN_COORDINATE,
    max: MAX_COORDINATE,
    onCommit: (v) => updateParams({ y: v }),
  });
  const amount = useSyncedNumber(params?.amount ?? DEFAULT_AMOUNT, {
    min: MIN_AMOUNT,
    max: MAX_AMOUNT,
    fallback: DEFAULT_AMOUNT,
    onCommit: (v) => updateParams({ amount: v }),
  });
  const maxScrolls = useSyncedNumber(params?.maxScrolls ?? DEFAULT_SCROLLS, {
    min: MIN_SCROLLS,
    max: MAX_SCROLLS,
    fallback: DEFAULT_SCROLLS,
    onCommit: (v) => updateParams({ maxScrolls: v }),
  });
  const timeoutMs = useSyncedNumber(params?.timeoutMs ?? 5000, {
    min: MIN_TIMEOUT,
    max: 120000,
    fallback: 5000,
    onCommit: (v) => updateParams({ timeoutMs: v }),
  });
  const waitForMs = useSyncedNumber(params?.waitForMs ?? 0, {
    min: 0,
    onCommit: (v) => updateParams({ waitForMs: v || undefined }),
  });

  const resilienceConfig = getValue<ResilienceSettings>('resilience');

  const showDirection = scrollType.value === 'page' || scrollType.value === 'untilVisible';
  const showAmount = scrollType.value === 'page' || scrollType.value === 'untilVisible';
  const showBehavior = scrollType.value !== 'untilVisible' && scrollType.value === 'page';
  const showSelector = scrollType.value === 'element';
  const showTargetSelector = scrollType.value === 'untilVisible';
  const showPosition = scrollType.value === 'position';

  return (
    <BaseNode selected={selected} icon={ScrollText} iconClassName="text-amber-300" title="Scroll">
      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">Scroll type</label>
          <select
            value={scrollType.value}
            onChange={selectInputHandler(scrollType.setValue, scrollType.commit)}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          >
            <option value="page">Page (window)</option>
            <option value="element">Element scrollIntoView</option>
            <option value="position">Scroll to coordinates</option>
            <option value="untilVisible">Scroll until visible</option>
          </select>
        </div>

        {showBehavior && (
          <div>
            <label className="text-gray-400 block mb-1">Behavior</label>
            <select
              value={behavior.value}
              onChange={selectInputHandler(behavior.setValue, behavior.commit)}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            >
              <option value="auto">Instant</option>
              <option value="smooth">Smooth</option>
            </select>
          </div>
        )}

        {showDirection && (
          <div>
            <label className="text-gray-400 block mb-1">Direction</label>
            <select
              value={direction.value}
              onChange={selectInputHandler(direction.setValue, direction.commit)}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            >
              <option value="down">Down</option>
              <option value="up">Up</option>
              <option value="left">Left</option>
              <option value="right">Right</option>
              <option value="top">Top</option>
              <option value="bottom">Bottom</option>
            </select>
          </div>
        )}

        {showAmount && (
          <div>
            <label className="text-gray-400 block mb-1">Amount (px)</label>
            <input
              type="number"
              min={MIN_AMOUNT}
              max={MAX_AMOUNT}
              value={amount.value}
              onChange={numberInputHandler(amount.setValue)}
              onBlur={amount.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
            <p className="text-gray-500 mt-1">Pixels per scroll step.</p>
          </div>
        )}

        {showSelector && (
          <div>
            <label className="text-gray-400 block mb-1">Element selector</label>
            <input
              type="text"
              value={selector.value}
              placeholder=".list-panel"
              onChange={textInputHandler(selector.setValue)}
              onBlur={selector.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        {showTargetSelector && (
          <div>
            <label className="text-gray-400 block mb-1">Target selector</label>
            <input
              type="text"
              placeholder="#lazy-card"
              value={targetSelector.value}
              onChange={textInputHandler(targetSelector.setValue)}
              onBlur={targetSelector.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
            <p className="text-gray-500 mt-1">Page scroll repeats until this element is visible.</p>
          </div>
        )}

        {showTargetSelector && (
          <div>
            <label className="text-gray-400 block mb-1">Max scroll attempts</label>
            <input
              type="number"
              min={MIN_SCROLLS}
              max={MAX_SCROLLS}
              value={maxScrolls.value}
              onChange={numberInputHandler(maxScrolls.setValue)}
              onBlur={maxScrolls.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        {showPosition && (
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="text-gray-400 block mb-1">X (px)</label>
              <input
                type="number"
                value={x.value}
                onChange={numberInputHandler(x.setValue)}
                onBlur={x.commit}
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
            </div>
            <div>
              <label className="text-gray-400 block mb-1">Y (px)</label>
              <input
                type="number"
                value={y.value}
                onChange={numberInputHandler(y.setValue)}
                onBlur={y.commit}
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
            </div>
          </div>
        )}

        <div className="grid grid-cols-2 gap-2">
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
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Post-scroll wait (ms)</label>
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

        <p className="text-gray-500">
          Scroll nodes unlock lazy-loaded content, infinite lists, and stable viewport positioning
          for assertions and screenshots.
        </p>
      </div>

      <ResiliencePanel
        value={resilienceConfig}
        onChange={(next) => updateData({ resilience: next ?? null })}
      />
    </BaseNode>
  );
};

export default memo(ScrollNode);
