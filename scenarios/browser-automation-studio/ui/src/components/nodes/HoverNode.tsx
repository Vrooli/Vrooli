import { memo, FC, useCallback, useEffect, useMemo, useState } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { MousePointer } from 'lucide-react';

const MIN_STEPS = 1;
const MAX_STEPS = 50;
const DEFAULT_STEPS = 10;
const MIN_DURATION = 50;
const MAX_DURATION = 10000;
const DEFAULT_DURATION = 350;

const clampNumber = (value: number, min: number, max: number, fallback: number): number => {
  if (!Number.isFinite(value)) {
    return fallback;
  }
  return Math.min(Math.max(Math.round(value), min), max);
};

const HoverNode: FC<NodeProps> = ({ data, selected, id }) => {
  const nodeData = (data ?? {}) as Record<string, unknown>;
  const { getNodes, setNodes } = useReactFlow();

  const [selector, setSelector] = useState<string>(() => typeof nodeData.selector === 'string' ? nodeData.selector : '');
  const [timeoutMs, setTimeoutMs] = useState<number>(() => Number(nodeData.timeoutMs ?? 5000) || 5000);
  const [waitForMs, setWaitForMs] = useState<number>(() => Number(nodeData.waitForMs ?? 0) || 0);
  const [steps, setSteps] = useState<number>(() => clampNumber(Number(nodeData.steps ?? DEFAULT_STEPS), MIN_STEPS, MAX_STEPS, DEFAULT_STEPS));
  const [durationMs, setDurationMs] = useState<number>(() => clampNumber(Number(nodeData.durationMs ?? DEFAULT_DURATION), MIN_DURATION, MAX_DURATION, DEFAULT_DURATION));

  useEffect(() => {
    setSelector(typeof nodeData.selector === 'string' ? nodeData.selector : '');
  }, [nodeData.selector]);

  useEffect(() => {
    setTimeoutMs(Number(nodeData.timeoutMs ?? 5000) || 5000);
  }, [nodeData.timeoutMs]);

  useEffect(() => {
    setWaitForMs(Number(nodeData.waitForMs ?? 0) || 0);
  }, [nodeData.waitForMs]);

  useEffect(() => {
    setSteps(clampNumber(Number(nodeData.steps ?? DEFAULT_STEPS), MIN_STEPS, MAX_STEPS, DEFAULT_STEPS));
  }, [nodeData.steps]);

  useEffect(() => {
    setDurationMs(clampNumber(Number(nodeData.durationMs ?? DEFAULT_DURATION), MIN_DURATION, MAX_DURATION, DEFAULT_DURATION));
  }, [nodeData.durationMs]);

  const updateNodeData = useCallback((updates: Record<string, unknown>) => {
    const nodes = getNodes();
    setNodes(nodes.map((node) => {
      if (node.id !== id) {
        return node;
      }
      return {
        ...node,
        data: {
          ...(node.data ?? {}),
          ...updates,
        },
      };
    }));
  }, [getNodes, setNodes, id]);

  const stepsHint = useMemo(() => {
    if (steps <= 3) {
      return 'Instant';
    }
    if (steps <= 10) {
      return 'Smooth';
    }
    return 'Very smooth';
  }, [steps]);

  const handleSelectorBlur = useCallback(() => {
    updateNodeData({ selector: selector.trim() });
  }, [selector, updateNodeData]);

  const handleTimeoutBlur = useCallback(() => {
    const clamped = Math.max(100, Math.round(timeoutMs) || 100);
    setTimeoutMs(clamped);
    updateNodeData({ timeoutMs: clamped });
  }, [timeoutMs, updateNodeData]);

  const handleWaitBlur = useCallback(() => {
    const normalized = Math.max(0, Math.round(waitForMs) || 0);
    setWaitForMs(normalized);
    updateNodeData({ waitForMs: normalized });
  }, [waitForMs, updateNodeData]);

  const handleStepsBlur = useCallback(() => {
    const normalized = clampNumber(steps, MIN_STEPS, MAX_STEPS, DEFAULT_STEPS);
    setSteps(normalized);
    updateNodeData({ steps: normalized });
  }, [steps, updateNodeData]);

  const handleDurationBlur = useCallback(() => {
    const normalized = clampNumber(durationMs, MIN_DURATION, MAX_DURATION, DEFAULT_DURATION);
    setDurationMs(normalized);
    updateNodeData({ durationMs: normalized });
  }, [durationMs, updateNodeData]);

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />

      <div className="flex items-center gap-2 mb-3">
        <MousePointer size={16} className="text-sky-300" />
        <span className="font-semibold text-sm">Hover</span>
      </div>

      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">Target selector</label>
          <input
            type="text"
            value={selector}
            placeholder=".menu-item > button"
            onChange={(event) => setSelector(event.target.value)}
            onBlur={handleSelectorBlur}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Timeout (ms)</label>
            <input
              type="number"
              min={100}
              value={timeoutMs}
              onChange={(event) => setTimeoutMs(Number(event.target.value))}
              onBlur={handleTimeoutBlur}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Post-hover wait (ms)</label>
            <input
              type="number"
              min={0}
              value={waitForMs}
              onChange={(event) => setWaitForMs(Number(event.target.value))}
              onBlur={handleWaitBlur}
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
              value={steps}
              onChange={(event) => setSteps(Number(event.target.value))}
              onBlur={handleStepsBlur}
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
              value={durationMs}
              onChange={(event) => setDurationMs(Number(event.target.value))}
              onBlur={handleDurationBlur}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
            <p className="text-gray-500 mt-1">Controls how long the pointer glides.</p>
          </div>
        </div>

        <p className="text-gray-500">
          Hover nodes move the cursor without clicking so menus, tooltips, and other hover-only UI remain open for following nodes.
        </p>
      </div>

      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(HoverNode);
