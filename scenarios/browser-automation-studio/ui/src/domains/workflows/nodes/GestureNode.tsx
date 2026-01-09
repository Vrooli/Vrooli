import { FC, memo, useMemo } from 'react';
import type { NodeProps } from 'reactflow';
import { Hand } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useSyncedString, useSyncedNumber, useSyncedSelect } from '@hooks/useSyncedField';
import BaseNode from './BaseNode';
import { NodeTextField, NodeNumberField, NodeSelectField, FieldRow } from './fields';

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
        <NodeSelectField
          field={gestureType}
          label="Gesture Type"
          options={GESTURE_TYPES}
          description={gestureHint}
        />

        <NodeTextField
          field={selector}
          label="Target Selector (optional)"
          placeholder="#mobile-card"
        />

        {showSwipeFields && (
          <div className="space-y-2">
            <FieldRow>
              <NodeSelectField field={direction} label="Direction" options={SWIPE_DIRECTIONS} />
              <NodeNumberField field={distance} label="Distance (px)" min={25} max={2000} />
            </FieldRow>
            <FieldRow>
              <NodeNumberField field={durationMs} label="Duration (ms)" min={50} max={10000} />
              <NodeNumberField field={steps} label="Movement Steps" min={2} max={60} />
            </FieldRow>
          </div>
        )}

        {showPinchFields && (
          <div className="space-y-2">
            <FieldRow>
              <NodeNumberField field={scale} label="Scale" min={0.2} max={4} step={0.1} />
              <NodeNumberField field={steps} label="Steps" min={2} max={60} />
            </FieldRow>
            <NodeNumberField field={durationMs} label="Duration (ms)" min={50} max={10000} />
          </div>
        )}

        {showHoldField && (
          <NodeNumberField field={holdMs} label="Hold Duration (ms)" min={200} max={10000} />
        )}

        <FieldRow>
          <NodeTextField field={startX} label="Start X (px)" placeholder="auto" />
          <NodeTextField field={startY} label="Start Y (px)" placeholder="auto" />
        </FieldRow>

        {showSwipeFields && (
          <FieldRow>
            <NodeTextField field={endX} label="End X (px)" placeholder="auto" />
            <NodeTextField field={endY} label="End Y (px)" placeholder="auto" />
          </FieldRow>
        )}

        <NodeNumberField field={timeoutMs} label="Timeout (ms)" min={0} placeholder="15000" />
      </div>
    </BaseNode>
  );
};

export default memo(GestureNode);
