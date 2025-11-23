import { memo, FC, useCallback, useEffect, useMemo, useState, useRef } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { Hand } from 'lucide-react';
import ResiliencePanel from './ResiliencePanel';
import type { ResilienceSettings } from '../../types/workflow';

const MIN_STEPS = 1;
const MAX_STEPS = 60;
const MIN_DURATION = 50;
const MAX_DURATION = 20000;
const MIN_OFFSET = -5000;
const MAX_OFFSET = 5000;
const MIN_TIMEOUT = 500;

const clampNumber = (value: number, min: number, max: number, fallback: number): number => {
  if (!Number.isFinite(value)) {
    return fallback;
  }
  return Math.min(Math.max(Math.round(value), min), max);
};

const normalizeOffset = (value: number): number => {
  if (!Number.isFinite(value)) {
    return 0;
  }
  return Math.min(Math.max(Math.round(value), MIN_OFFSET), MAX_OFFSET);
};

const DragDropNode: FC<NodeProps> = ({ data, selected, id }) => {
  const nodeRef = useRef<HTMLDivElement>(null);
  const nodeData = (data ?? {}) as Record<string, unknown>;
  const { getNodes, setNodes } = useReactFlow();

  const [sourceSelector, setSourceSelector] = useState<string>(String(nodeData.sourceSelector ?? ''));
  const [targetSelector, setTargetSelector] = useState<string>(String(nodeData.targetSelector ?? ''));
  const [holdMs, setHoldMs] = useState<number>(Number(nodeData.holdMs ?? 150));
  const [steps, setSteps] = useState<number>(() => clampNumber(Number(nodeData.steps ?? 18), MIN_STEPS, MAX_STEPS, 18));
  const [durationMs, setDurationMs] = useState<number>(() => clampNumber(Number(nodeData.durationMs ?? 600), MIN_DURATION, MAX_DURATION, 600));
  const [offsetX, setOffsetX] = useState<number>(() => normalizeOffset(Number(nodeData.offsetX ?? 0)));
  const [offsetY, setOffsetY] = useState<number>(() => normalizeOffset(Number(nodeData.offsetY ?? 0)));
  const [timeoutMs, setTimeoutMs] = useState<number>(() => Math.max(MIN_TIMEOUT, Number(nodeData.timeoutMs ?? 30000) || 30000));
  const [waitForMs, setWaitForMs] = useState<number>(() => Math.max(0, Number(nodeData.waitForMs ?? 0) || 0));

  useEffect(() => setSourceSelector(String(nodeData.sourceSelector ?? '')), [nodeData.sourceSelector]);
  useEffect(() => setTargetSelector(String(nodeData.targetSelector ?? '')), [nodeData.targetSelector]);
  useEffect(() => setHoldMs(Number(nodeData.holdMs ?? 150)), [nodeData.holdMs]);
  useEffect(() => setSteps(clampNumber(Number(nodeData.steps ?? 18), MIN_STEPS, MAX_STEPS, 18)), [nodeData.steps]);
  useEffect(() => setDurationMs(clampNumber(Number(nodeData.durationMs ?? 600), MIN_DURATION, MAX_DURATION, 600)), [nodeData.durationMs]);
  useEffect(() => setOffsetX(normalizeOffset(Number(nodeData.offsetX ?? 0))), [nodeData.offsetX]);
  useEffect(() => setOffsetY(normalizeOffset(Number(nodeData.offsetY ?? 0))), [nodeData.offsetY]);
  useEffect(() => setTimeoutMs(Math.max(MIN_TIMEOUT, Number(nodeData.timeoutMs ?? 30000) || 30000)), [nodeData.timeoutMs]);
  useEffect(() => setWaitForMs(Math.max(0, Number(nodeData.waitForMs ?? 0) || 0)), [nodeData.waitForMs]);

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

  const resilienceConfig = nodeData.resilience as ResilienceSettings | undefined;

  const movementSummary = useMemo(() => {
    if (steps <= 5) {
      return 'Quick snap';
    }
    if (steps <= 20) {
      return 'Smooth glide';
    }
    return 'Very smooth';
  }, [steps]);

  const commitSelector = useCallback((key: 'sourceSelector' | 'targetSelector', value: string) => {
    const trimmed = value.trim();
    updateNodeData({ [key]: trimmed || undefined });
  }, [updateNodeData]);

  const commitHold = useCallback(() => {
    const normalized = Math.max(0, Math.round(holdMs) || 0);
    setHoldMs(normalized);
    updateNodeData({ holdMs: normalized });
  }, [holdMs, updateNodeData]);

  const commitSteps = useCallback(() => {
    const normalized = clampNumber(steps, MIN_STEPS, MAX_STEPS, 18);
    setSteps(normalized);
    updateNodeData({ steps: normalized });
  }, [steps, updateNodeData]);

  const commitDuration = useCallback(() => {
    const normalized = clampNumber(durationMs, MIN_DURATION, MAX_DURATION, 600);
    setDurationMs(normalized);
    updateNodeData({ durationMs: normalized });
  }, [durationMs, updateNodeData]);

  const commitTimeout = useCallback(() => {
    const normalized = Math.max(MIN_TIMEOUT, Math.round(timeoutMs) || MIN_TIMEOUT);
    setTimeoutMs(normalized);
    updateNodeData({ timeoutMs: normalized });
  }, [timeoutMs, updateNodeData]);

  const commitWait = useCallback(() => {
    const normalized = Math.max(0, Math.round(waitForMs) || 0);
    setWaitForMs(normalized);
    updateNodeData({ waitForMs: normalized });
  }, [waitForMs, updateNodeData]);

  const commitOffset = useCallback((axis: 'offsetX' | 'offsetY', value: number) => {
    const normalized = normalizeOffset(value);
    if (axis === 'offsetX') {
      setOffsetX(normalized);
    } else {
      setOffsetY(normalized);
    }
    updateNodeData({ [axis]: normalized });
  }, [updateNodeData]);

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
    <div ref={nodeRef} className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />

      <div className="flex items-center gap-2 mb-3">
        <Hand size={16} className="text-pink-300" />
        <span className="font-semibold text-sm">Drag &amp; Drop</span>
      </div>

      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">Drag from selector</label>
          <input
            type="text"
            value={sourceSelector}
            placeholder=".kanban-card:first-child"
            onChange={(event) => setSourceSelector(event.target.value)}
            onBlur={() => commitSelector('sourceSelector', sourceSelector)}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>
        <div>
          <label className="text-gray-400 block mb-1">Drop target selector</label>
          <input
            type="text"
            value={targetSelector}
            placeholder=".drop-target"
            onChange={(event) => setTargetSelector(event.target.value)}
            onBlur={() => commitSelector('targetSelector', targetSelector)}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Hold before drag (ms)</label>
            <input
              type="number"
              min={0}
              value={holdMs}
              onChange={(event) => setHoldMs(Number(event.target.value))}
              onBlur={commitHold}
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
              value={steps}
              onChange={(event) => setSteps(Number(event.target.value))}
              onBlur={commitSteps}
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
              value={durationMs}
              onChange={(event) => setDurationMs(Number(event.target.value))}
              onBlur={commitDuration}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
            <p className="text-gray-500 mt-1">Controls how slow the card travels.</p>
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Timeout (ms)</label>
            <input
              type="number"
              min={MIN_TIMEOUT}
              value={timeoutMs}
              onChange={(event) => setTimeoutMs(Number(event.target.value))}
              onBlur={commitTimeout}
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
              value={offsetX}
              onChange={(event) => setOffsetX(Number(event.target.value))}
              onBlur={() => commitOffset('offsetX', offsetX)}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Offset Y (px)</label>
            <input
              type="number"
              min={MIN_OFFSET}
              max={MAX_OFFSET}
              value={offsetY}
              onChange={(event) => setOffsetY(Number(event.target.value))}
              onBlur={() => commitOffset('offsetY', offsetY)}
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
              value={waitForMs}
              onChange={(event) => setWaitForMs(Number(event.target.value))}
              onBlur={commitWait}
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
        onChange={(next) => updateNodeData({ resilience: next ?? null })}
      />

      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(DragDropNode);
