import { FC, memo } from 'react';
import type { NodeProps } from 'reactflow';
import { ScrollText } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import { useSyncedString, useSyncedNumber, useSyncedSelect } from '@hooks/useSyncedField';
import type { ResilienceSettings } from '@/types/workflow';
import BaseNode from './BaseNode';
import ResiliencePanel from './ResiliencePanel';
import { NodeTextField, NodeNumberField, NodeSelectField, FieldRow } from './fields';

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

const SCROLL_TYPE_OPTIONS = [
  { value: 'page', label: 'Page (window)' },
  { value: 'element', label: 'Element scrollIntoView' },
  { value: 'position', label: 'Scroll to coordinates' },
  { value: 'untilVisible', label: 'Scroll until visible' },
];

const BEHAVIOR_OPTIONS = [
  { value: 'auto', label: 'Instant' },
  { value: 'smooth', label: 'Smooth' },
];

const DIRECTION_OPTIONS = [
  { value: 'down', label: 'Down' },
  { value: 'up', label: 'Up' },
  { value: 'left', label: 'Left' },
  { value: 'right', label: 'Right' },
  { value: 'top', label: 'Top' },
  { value: 'bottom', label: 'Bottom' },
];

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
        <NodeSelectField field={scrollType} label="Scroll type" options={SCROLL_TYPE_OPTIONS} />

        {showBehavior && (
          <NodeSelectField field={behavior} label="Behavior" options={BEHAVIOR_OPTIONS} />
        )}

        {showDirection && (
          <NodeSelectField field={direction} label="Direction" options={DIRECTION_OPTIONS} />
        )}

        {showAmount && (
          <NodeNumberField
            field={amount}
            label="Amount (px)"
            min={MIN_AMOUNT}
            max={MAX_AMOUNT}
            description="Pixels per scroll step."
          />
        )}

        {showSelector && (
          <NodeTextField field={selector} label="Element selector" placeholder=".list-panel" />
        )}

        {showTargetSelector && (
          <>
            <NodeTextField
              field={targetSelector}
              label="Target selector"
              placeholder="#lazy-card"
              description="Page scroll repeats until this element is visible."
            />
            <NodeNumberField
              field={maxScrolls}
              label="Max scroll attempts"
              min={MIN_SCROLLS}
              max={MAX_SCROLLS}
            />
          </>
        )}

        {showPosition && (
          <FieldRow>
            <NodeNumberField field={x} label="X (px)" />
            <NodeNumberField field={y} label="Y (px)" />
          </FieldRow>
        )}

        <FieldRow>
          <NodeNumberField field={timeoutMs} label="Timeout (ms)" min={MIN_TIMEOUT} />
          <NodeNumberField field={waitForMs} label="Post-scroll wait (ms)" min={0} />
        </FieldRow>

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
