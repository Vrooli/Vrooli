import { FC, memo, useCallback, useEffect, useMemo, useState } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { Hand } from 'lucide-react';

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

const clampNumber = (value: number, min: number, max: number): number => {
  if (!Number.isFinite(value)) {
    return min;
  }
  return Math.min(Math.max(Math.round(value), min), max);
};

const clampScale = (raw: number): number => {
  if (!Number.isFinite(raw) || raw <= 0) {
    return DEFAULT_SCALE;
  }
  const normalized = Math.max(0.2, Math.min(4, raw));
  return Number(normalized.toFixed(2));
};

const formatCoordinate = (value: unknown): string => {
  if (value === undefined || value === null || value === '') {
    return '';
  }
  const numeric = Number(value);
  if (!Number.isFinite(numeric)) {
    return '';
  }
  return String(Math.round(numeric));
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

const GestureNode: FC<NodeProps> = ({ id, data, selected }) => {
  const { getNodes, setNodes } = useReactFlow();
  const nodeData = (data ?? {}) as Record<string, unknown>;

  const [gestureType, setGestureType] = useState(() => String(nodeData.gestureType ?? 'tap'));
  const [selector, setSelector] = useState(() => String(nodeData.selector ?? ''));
  const [direction, setDirection] = useState(() => String(nodeData.direction ?? 'down'));
  const [distance, setDistance] = useState(() => Number(nodeData.distance ?? DEFAULT_DISTANCE));
  const [durationMs, setDurationMs] = useState(() => Number(nodeData.durationMs ?? DEFAULT_DURATION));
  const [steps, setSteps] = useState(() => Number(nodeData.steps ?? DEFAULT_STEPS));
  const [holdMs, setHoldMs] = useState(() => Number(nodeData.holdMs ?? DEFAULT_HOLD));
  const [scale, setScale] = useState(() => Number(nodeData.scale ?? DEFAULT_SCALE));
  const [startX, setStartX] = useState(() => formatCoordinate(nodeData.startX));
  const [startY, setStartY] = useState(() => formatCoordinate(nodeData.startY));
  const [endX, setEndX] = useState(() => formatCoordinate(nodeData.endX));
  const [endY, setEndY] = useState(() => formatCoordinate(nodeData.endY));
  const [timeoutMs, setTimeoutMs] = useState(() => Number(nodeData.timeoutMs ?? 0));

  useEffect(() => setGestureType(String(nodeData.gestureType ?? 'tap')), [nodeData.gestureType]);
  useEffect(() => setSelector(String(nodeData.selector ?? '')), [nodeData.selector]);
  useEffect(() => setDirection(String(nodeData.direction ?? 'down')), [nodeData.direction]);
  useEffect(() => setDistance(Number(nodeData.distance ?? DEFAULT_DISTANCE)), [nodeData.distance]);
  useEffect(() => setDurationMs(Number(nodeData.durationMs ?? DEFAULT_DURATION)), [nodeData.durationMs]);
  useEffect(() => setSteps(Number(nodeData.steps ?? DEFAULT_STEPS)), [nodeData.steps]);
  useEffect(() => setHoldMs(Number(nodeData.holdMs ?? DEFAULT_HOLD)), [nodeData.holdMs]);
  useEffect(() => setScale(Number(nodeData.scale ?? DEFAULT_SCALE)), [nodeData.scale]);
  useEffect(() => setStartX(formatCoordinate(nodeData.startX)), [nodeData.startX]);
  useEffect(() => setStartY(formatCoordinate(nodeData.startY)), [nodeData.startY]);
  useEffect(() => setEndX(formatCoordinate(nodeData.endX)), [nodeData.endX]);
  useEffect(() => setEndY(formatCoordinate(nodeData.endY)), [nodeData.endY]);
  useEffect(() => setTimeoutMs(Number(nodeData.timeoutMs ?? 0)), [nodeData.timeoutMs]);

  const updateNodeData = useCallback((updates: Record<string, unknown>) => {
    const nodes = getNodes();
    setNodes(nodes.map((node) => {
      if (node.id !== id) {
        return node;
      }
      const nextData = { ...(node.data ?? {}) } as Record<string, unknown>;
      Object.entries(updates).forEach(([key, value]) => {
        if (value === undefined || value === null || value === '') {
          delete nextData[key];
        } else {
          nextData[key] = value;
        }
      });
      return { ...node, data: nextData };
    }));
  }, [getNodes, setNodes, id]);

  const handleGestureChange = useCallback((value: string) => {
    setGestureType(value);
    updateNodeData({ gestureType: value });
  }, [updateNodeData]);

  const commitDistance = useCallback(() => {
    const normalized = clampNumber(distance || DEFAULT_DISTANCE, 25, 2000);
    setDistance(normalized);
    updateNodeData({ distance: normalized });
  }, [distance, updateNodeData]);

  const commitDuration = useCallback(() => {
    const normalized = clampNumber(durationMs || DEFAULT_DURATION, 50, 10000);
    setDurationMs(normalized);
    updateNodeData({ durationMs: normalized });
  }, [durationMs, updateNodeData]);

  const commitSteps = useCallback(() => {
    const normalized = clampNumber(steps || DEFAULT_STEPS, 2, 60);
    setSteps(normalized);
    updateNodeData({ steps: normalized });
  }, [steps, updateNodeData]);

  const commitHold = useCallback(() => {
    const normalized = clampNumber(holdMs || DEFAULT_HOLD, 200, 10000);
    setHoldMs(normalized);
    updateNodeData({ holdMs: normalized });
  }, [holdMs, updateNodeData]);

  const commitScale = useCallback(() => {
    const normalized = clampScale(scale || DEFAULT_SCALE);
    setScale(normalized);
    updateNodeData({ scale: normalized });
  }, [scale, updateNodeData]);

  const commitTimeout = useCallback(() => {
    const normalized = Math.max(0, Math.round(timeoutMs) || 0);
    setTimeoutMs(normalized);
    updateNodeData({ timeoutMs: normalized || undefined });
  }, [timeoutMs, updateNodeData]);

  const commitCoordinate = useCallback((key: 'startX' | 'startY' | 'endX' | 'endY', raw: string) => {
    const parsed = parseCoordinate(raw);
    updateNodeData({ [key]: parsed });
  }, [updateNodeData]);

  const showSwipeFields = gestureType === 'swipe';
  const showPinchFields = gestureType === 'pinch';
  const showHoldField = gestureType === 'longPress';

  const gestureHint = useMemo(() => {
    switch (gestureType) {
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
  }, [gestureType]);

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />
      <Handle type="source" position={Position.Bottom} className="node-handle" />

      <div className="flex items-center gap-2 mb-3">
        <Hand size={16} className="text-rose-300" />
        <div>
          <div className="font-semibold text-sm">Gesture</div>
          <p className="text-[11px] text-gray-500">Recreate mobile touch interactions mid-run.</p>
        </div>
      </div>

      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">Gesture Type</label>
          <select
            value={gestureType}
            onChange={(event) => handleGestureChange(event.target.value)}
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
            value={selector}
            onChange={(event) => setSelector(event.target.value)}
            onBlur={() => updateNodeData({ selector: selector.trim() })}
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
                  value={direction}
                  onChange={(event) => {
                    setDirection(event.target.value);
                    updateNodeData({ direction: event.target.value });
                  }}
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
                  value={distance}
                  onChange={(event) => setDistance(Number(event.target.value))}
                  onBlur={commitDistance}
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
                  value={durationMs}
                  onChange={(event) => setDurationMs(Number(event.target.value))}
                  onBlur={commitDuration}
                  className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                />
              </div>
              <div>
                <label className="text-gray-400 block mb-1">Movement Steps</label>
                <input
                  type="number"
                  min={2}
                  max={60}
                  value={steps}
                  onChange={(event) => setSteps(Number(event.target.value))}
                  onBlur={commitSteps}
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
                  value={scale}
                  onChange={(event) => setScale(Number(event.target.value))}
                  onBlur={commitScale}
                  className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
                />
              </div>
              <div>
                <label className="text-gray-400 block mb-1">Steps</label>
                <input
                  type="number"
                  min={2}
                  max={60}
                  value={steps}
                  onChange={(event) => setSteps(Number(event.target.value))}
                  onBlur={commitSteps}
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
                value={durationMs}
                onChange={(event) => setDurationMs(Number(event.target.value))}
                onBlur={commitDuration}
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
              value={holdMs}
              onChange={(event) => setHoldMs(Number(event.target.value))}
              onBlur={commitHold}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Start X (px)</label>
            <input
              type="number"
              value={startX}
              onChange={(event) => setStartX(event.target.value)}
              onBlur={() => commitCoordinate('startX', startX)}
              placeholder="auto"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Start Y (px)</label>
            <input
              type="number"
              value={startY}
              onChange={(event) => setStartY(event.target.value)}
              onBlur={() => commitCoordinate('startY', startY)}
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
                value={endX}
                onChange={(event) => setEndX(event.target.value)}
                onBlur={() => commitCoordinate('endX', endX)}
                placeholder="auto"
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
            </div>
            <div>
              <label className="text-gray-400 block mb-1">End Y (px)</label>
              <input
                type="number"
                value={endY}
                onChange={(event) => setEndY(event.target.value)}
                onBlur={() => commitCoordinate('endY', endY)}
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
            value={timeoutMs}
            onChange={(event) => setTimeoutMs(Number(event.target.value))}
            onBlur={commitTimeout}
            placeholder="15000"
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>
      </div>
    </div>
  );
};

export default memo(GestureNode);
