import { FC, memo, useMemo } from 'react';
import type { NodeProps } from 'reactflow';
import { Hand } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import {
  useSyncedString,
  useSyncedNumber,
  useSyncedSelect,
  textInputHandler,
  numberInputHandler,
  selectInputHandler,
} from '@hooks/useSyncedField';
import BaseNode from './BaseNode';

// GestureParams interface for V2 native action params
interface GestureParams {
  gestureType?: string;
  selector?: string;
  direction?: string;
  distance?: number;
  durationMs?: number;
  steps?: number;
  holdMs?: number;
  scale?: number;
  startX?: number;
  startY?: number;
  endX?: number;
  endY?: number;
  timeoutMs?: number;
}

const GESTURE_TYPES = [
  { value: 'tap', label: 'Tap (single touch)' },
  { value: 'doubleTap', label: 'Double Tap' },
  { value: 'longPress', label: 'Long Press' },
  { value: 'swipe', label: 'Swipe / Flick' },
  { value: 'pinch', label: 'Pinch / Zoom' },
];

const SWIPE_DIRECTIONS = [
  { value: 'down', label: 'Down' },
  { value: 'up', label: 'Up' },
  { value: 'left', label: 'Left' },
  { value: 'right', label: 'Right' },
];

const DEFAULT_DISTANCE = 150;
const DEFAULT_DURATION = 350;
const DEFAULT_STEPS = 15;
const DEFAULT_HOLD = 800;
const DEFAULT_SCALE = 0.6;

const formatCoordinate = (value?: number): string => {
  if (value === undefined || value === null) {
    return '';
  }
  if (!Number.isFinite(value)) {
    return '';
  }
  return String(Math.round(value));
};

const parseCoordinate = (raw: string): number | undefined => {
  const trimmed = raw.trim();
  if (trimmed === '') {
    return undefined;
  }
  const numeric = Number(trimmed);
  if (!Number.isFinite(numeric)) {
    return undefined;
  }
  return Math.round(numeric);
};

const GestureNode: FC<NodeProps> = ({ id, selected }) => {
  const { params, updateParams } = useActionParams<GestureParams>(id);

  // Select fields
  const gestureType = useSyncedSelect(params?.gestureType ?? 'tap', {
    onCommit: (v) => updateParams({ gestureType: v }),
  });
  const direction = useSyncedSelect(params?.direction ?? 'down', {
    onCommit: (v) => updateParams({ direction: v }),
  });

  // String fields
  const selector = useSyncedString(params?.selector ?? '', {
    onCommit: (v) => updateParams({ selector: v || undefined }),
  });

  // Number fields
  const distance = useSyncedNumber(params?.distance ?? DEFAULT_DISTANCE, {
    min: 25,
    max: 2000,
    fallback: DEFAULT_DISTANCE,
    onCommit: (v) => updateParams({ distance: v }),
  });
  const durationMs = useSyncedNumber(params?.durationMs ?? DEFAULT_DURATION, {
    min: 50,
    max: 10000,
    fallback: DEFAULT_DURATION,
    onCommit: (v) => updateParams({ durationMs: v }),
  });
  const steps = useSyncedNumber(params?.steps ?? DEFAULT_STEPS, {
    min: 2,
    max: 60,
    fallback: DEFAULT_STEPS,
    onCommit: (v) => updateParams({ steps: v }),
  });
  const holdMs = useSyncedNumber(params?.holdMs ?? DEFAULT_HOLD, {
    min: 200,
    max: 10000,
    fallback: DEFAULT_HOLD,
    onCommit: (v) => updateParams({ holdMs: v }),
  });
  const scale = useSyncedNumber(params?.scale ?? DEFAULT_SCALE, {
    min: 0.2,
    max: 4,
    fallback: DEFAULT_SCALE,
    onCommit: (v) => updateParams({ scale: v }),
  });
  const timeoutMs = useSyncedNumber(params?.timeoutMs ?? 0, {
    min: 0,
    onCommit: (v) => updateParams({ timeoutMs: v || undefined }),
  });

  // Coordinate fields (as strings for optional handling)
  const startX = useSyncedString(formatCoordinate(params?.startX), {
    onCommit: (v) => updateParams({ startX: parseCoordinate(v) }),
  });
  const startY = useSyncedString(formatCoordinate(params?.startY), {
    onCommit: (v) => updateParams({ startY: parseCoordinate(v) }),
  });
  const endX = useSyncedString(formatCoordinate(params?.endX), {
    onCommit: (v) => updateParams({ endX: parseCoordinate(v) }),
  });
  const endY = useSyncedString(formatCoordinate(params?.endY), {
    onCommit: (v) => updateParams({ endY: parseCoordinate(v) }),
  });

  const showSwipeFields = gestureType.value === 'swipe';
  const showPinchFields = gestureType.value === 'pinch';
  const showHoldField = gestureType.value === 'longPress';

  const gestureHint = useMemo(() => {
    switch (gestureType.value) {
      case 'swipe':
        return 'Simulate a directional swipe to reveal new content or dismiss UI elements.';
      case 'pinch':
        return 'Pinch in/out to test zooming and map interactions.';
      case 'doubleTap':
        return 'Rapidly tap twice, useful for zoom shortcuts or liking content.';
      case 'longPress':
        return 'Hold your finger in place to open context menus or drag handles.';
      default:
        return 'Single tap gestures cover the majority of mobile interactions.';
    }
  }, [gestureType.value]);

  return (
    <BaseNode selected={selected} icon={Hand} iconClassName="text-rose-300" title="Gesture">
      <p className="text-[11px] text-gray-500 mb-3">Recreate mobile touch interactions mid-run.</p>

      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">Gesture Type</label>
          <select
            value={gestureType.value}
            onChange={selectInputHandler(gestureType.setValue, gestureType.commit)}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          >
            {GESTURE_TYPES.map((option) => (
              <option key={option.value} value={option.value}>{option.label}</option>
            ))}
          </select>
          <p className="text-gray-500 mt-1">{gestureHint}</p>
        </div>

        <div>
          <label className="text-gray-400 block mb-1">Target Selector (optional)</label>
          <input
            type="text"
            value={selector.value}
            onChange={textInputHandler(selector.setValue)}
            onBlur={selector.commit}
            placeholder="#mobile-card"
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>

        {showSwipeFields && (
          <div className="space-y-2">
            <div className="grid grid-cols-2 gap-2">
              <div>
                <label className="text-gray-400 block mb-1">Direction</label>
                <select
                  value={direction.value}
                  onChange={selectInputHandler(direction.setValue, direction.commit)}
                  className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                >
                  {SWIPE_DIRECTIONS.map((option) => (
                    <option key={option.value} value={option.value}>{option.label}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-gray-400 block mb-1">Distance (px)</label>
                <input
                  type="number"
                  min={25}
                  max={2000}
                  value={distance.value}
                  onChange={numberInputHandler(distance.setValue)}
                  onBlur={distance.commit}
                  className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div>
                <label className="text-gray-400 block mb-1">Duration (ms)</label>
                <input
                  type="number"
                  min={50}
                  max={10000}
                  value={durationMs.value}
                  onChange={numberInputHandler(durationMs.setValue)}
                  onBlur={durationMs.commit}
                  className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                />
              </div>
              <div>
                <label className="text-gray-400 block mb-1">Movement Steps</label>
                <input
                  type="number"
                  min={2}
                  max={60}
                  value={steps.value}
                  onChange={numberInputHandler(steps.setValue)}
                  onBlur={steps.commit}
                  className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                />
              </div>
            </div>
          </div>
        )}

        {showPinchFields && (
          <div className="space-y-2">
            <div className="grid grid-cols-2 gap-2">
              <div>
                <label className="text-gray-400 block mb-1">Scale</label>
                <input
                  type="number"
                  step={0.1}
                  min={0.2}
                  max={4}
                  value={scale.value}
                  onChange={numberInputHandler(scale.setValue)}
                  onBlur={scale.commit}
                  className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                />
              </div>
              <div>
                <label className="text-gray-400 block mb-1">Steps</label>
                <input
                  type="number"
                  min={2}
                  max={60}
                  value={steps.value}
                  onChange={numberInputHandler(steps.setValue)}
                  onBlur={steps.commit}
                  className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                />
              </div>
            </div>
            <div>
              <label className="text-gray-400 block mb-1">Duration (ms)</label>
              <input
                type="number"
                min={50}
                max={10000}
                value={durationMs.value}
                onChange={numberInputHandler(durationMs.setValue)}
                onBlur={durationMs.commit}
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
            </div>
          </div>
        )}

        {showHoldField && (
          <div>
            <label className="text-gray-400 block mb-1">Hold Duration (ms)</label>
            <input
              type="number"
              min={200}
              max={10000}
              value={holdMs.value}
              onChange={numberInputHandler(holdMs.setValue)}
              onBlur={holdMs.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Start X (px)</label>
            <input
              type="number"
              value={startX.value}
              onChange={textInputHandler(startX.setValue)}
              onBlur={startX.commit}
              placeholder="auto"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Start Y (px)</label>
            <input
              type="number"
              value={startY.value}
              onChange={textInputHandler(startY.setValue)}
              onBlur={startY.commit}
              placeholder="auto"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        </div>

        {showSwipeFields && (
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label className="text-gray-400 block mb-1">End X (px)</label>
              <input
                type="number"
                value={endX.value}
                onChange={textInputHandler(endX.setValue)}
                onBlur={endX.commit}
                placeholder="auto"
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
            </div>
            <div>
              <label className="text-gray-400 block mb-1">End Y (px)</label>
              <input
                type="number"
                value={endY.value}
                onChange={textInputHandler(endY.setValue)}
                onBlur={endY.commit}
                placeholder="auto"
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
            </div>
          </div>
        )}

        <div>
          <label className="text-gray-400 block mb-1">Timeout (ms)</label>
          <input
            type="number"
            min={0}
            value={timeoutMs.value}
            onChange={numberInputHandler(timeoutMs.setValue)}
            onBlur={timeoutMs.commit}
            placeholder="15000"
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>
      </div>
    </BaseNode>
  );
};

export default memo(GestureNode);
