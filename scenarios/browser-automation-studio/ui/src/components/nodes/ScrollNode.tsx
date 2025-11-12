import { FC, memo, useCallback, useEffect, useState } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { ScrollText } from 'lucide-react';

const MIN_AMOUNT = 10;
const MAX_AMOUNT = 5000;
const DEFAULT_AMOUNT = 400;
const MIN_SCROLLS = 1;
const MAX_SCROLLS = 200;
const DEFAULT_SCROLLS = 12;
const MIN_COORDINATE = -500000;
const MAX_COORDINATE = 500000;
const MIN_TIMEOUT = 100;

const clampNumber = (value: number, min: number, max: number, fallback: number): number => {
  if (!Number.isFinite(value)) {
    return fallback;
  }
  return Math.min(Math.max(Math.round(value), min), max);
};

const normalizeCoordinate = (value: number): number => {
  if (!Number.isFinite(value)) {
    return 0;
  }
  return Math.min(Math.max(Math.round(value), MIN_COORDINATE), MAX_COORDINATE);
};

const ScrollNode: FC<NodeProps> = ({ data, selected, id }) => {
  const nodeData = (data ?? {}) as Record<string, unknown>;
  const { getNodes, setNodes } = useReactFlow();

  const [scrollType, setScrollType] = useState<string>(() => typeof nodeData.scrollType === 'string' ? nodeData.scrollType : 'page');
  const [selector, setSelector] = useState<string>(() => typeof nodeData.selector === 'string' ? nodeData.selector : '');
  const [targetSelector, setTargetSelector] = useState<string>(() => typeof nodeData.targetSelector === 'string' ? nodeData.targetSelector : '');
  const [direction, setDirection] = useState<string>(() => typeof nodeData.direction === 'string' ? nodeData.direction : 'down');
  const [behavior, setBehavior] = useState<string>(() => typeof nodeData.behavior === 'string' ? nodeData.behavior : 'auto');
  const [amount, setAmount] = useState<number>(() => clampNumber(Number(nodeData.amount ?? DEFAULT_AMOUNT), MIN_AMOUNT, MAX_AMOUNT, DEFAULT_AMOUNT));
  const [maxScrolls, setMaxScrolls] = useState<number>(() => clampNumber(Number(nodeData.maxScrolls ?? DEFAULT_SCROLLS), MIN_SCROLLS, MAX_SCROLLS, DEFAULT_SCROLLS));
  const [x, setX] = useState<number>(() => normalizeCoordinate(Number(nodeData.x ?? 0)));
  const [y, setY] = useState<number>(() => normalizeCoordinate(Number(nodeData.y ?? 0)));
  const [timeoutMs, setTimeoutMs] = useState<number>(() => clampNumber(Number(nodeData.timeoutMs ?? 5000), MIN_TIMEOUT, 120000, 5000));
  const [waitForMs, setWaitForMs] = useState<number>(() => Math.max(0, Math.round(Number(nodeData.waitForMs ?? 0)) || 0));

  useEffect(() => {
    setScrollType(typeof nodeData.scrollType === 'string' ? nodeData.scrollType : 'page');
  }, [nodeData.scrollType]);

  useEffect(() => {
    setSelector(typeof nodeData.selector === 'string' ? nodeData.selector : '');
  }, [nodeData.selector]);

  useEffect(() => {
    setTargetSelector(typeof nodeData.targetSelector === 'string' ? nodeData.targetSelector : '');
  }, [nodeData.targetSelector]);

  useEffect(() => {
    setDirection(typeof nodeData.direction === 'string' ? nodeData.direction : 'down');
  }, [nodeData.direction]);

  useEffect(() => {
    setBehavior(typeof nodeData.behavior === 'string' ? nodeData.behavior : 'auto');
  }, [nodeData.behavior]);

  useEffect(() => {
    setAmount(clampNumber(Number(nodeData.amount ?? DEFAULT_AMOUNT), MIN_AMOUNT, MAX_AMOUNT, DEFAULT_AMOUNT));
  }, [nodeData.amount]);

  useEffect(() => {
    setMaxScrolls(clampNumber(Number(nodeData.maxScrolls ?? DEFAULT_SCROLLS), MIN_SCROLLS, MAX_SCROLLS, DEFAULT_SCROLLS));
  }, [nodeData.maxScrolls]);

  useEffect(() => {
    setX(normalizeCoordinate(Number(nodeData.x ?? 0)));
  }, [nodeData.x]);

  useEffect(() => {
    setY(normalizeCoordinate(Number(nodeData.y ?? 0)));
  }, [nodeData.y]);

  useEffect(() => {
    setTimeoutMs(clampNumber(Number(nodeData.timeoutMs ?? 5000), MIN_TIMEOUT, 120000, 5000));
  }, [nodeData.timeoutMs]);

  useEffect(() => {
    setWaitForMs(Math.max(0, Math.round(Number(nodeData.waitForMs ?? 0)) || 0));
  }, [nodeData.waitForMs]);

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

  const handleTypeChange = (value: string) => {
    setScrollType(value);
    updateNodeData({ scrollType: value });
  };

  const handleSelectorBlur = () => {
    updateNodeData({ selector: selector.trim() });
  };

  const handleTargetSelectorBlur = () => {
    updateNodeData({ targetSelector: targetSelector.trim() });
  };

  const handleDirectionChange = (value: string) => {
    setDirection(value);
    updateNodeData({ direction: value });
  };

  const handleBehaviorChange = (value: string) => {
    setBehavior(value);
    updateNodeData({ behavior: value });
  };

  const handleAmountBlur = () => {
    const clamped = clampNumber(amount, MIN_AMOUNT, MAX_AMOUNT, DEFAULT_AMOUNT);
    setAmount(clamped);
    updateNodeData({ amount: clamped });
  };

  const handleMaxScrollsBlur = () => {
    const clamped = clampNumber(maxScrolls, MIN_SCROLLS, MAX_SCROLLS, DEFAULT_SCROLLS);
    setMaxScrolls(clamped);
    updateNodeData({ maxScrolls: clamped });
  };

  const handleXBlur = () => {
    const normalized = normalizeCoordinate(x);
    setX(normalized);
    updateNodeData({ x: normalized });
  };

  const handleYBlur = () => {
    const normalized = normalizeCoordinate(y);
    setY(normalized);
    updateNodeData({ y: normalized });
  };

  const handleTimeoutBlur = () => {
    const normalized = clampNumber(timeoutMs, MIN_TIMEOUT, 120000, 5000);
    setTimeoutMs(normalized);
    updateNodeData({ timeoutMs: normalized });
  };

  const handleWaitBlur = () => {
    const normalized = Math.max(0, Math.round(waitForMs) || 0);
    setWaitForMs(normalized);
    updateNodeData({ waitForMs: normalized });
  };

  const showDirection = scrollType === 'page' || scrollType === 'untilVisible';
  const showAmount = scrollType === 'page' || scrollType === 'untilVisible';
  const showBehavior = scrollType !== 'untilVisible' && scrollType === 'page';
  const showSelector = scrollType === 'element';
  const showTargetSelector = scrollType === 'untilVisible';
  const showPosition = scrollType === 'position';

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />

      <div className="flex items-center gap-2 mb-3">
        <ScrollText size={16} className="text-amber-300" />
        <span className="font-semibold text-sm">Scroll</span>
      </div>

      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">Scroll type</label>
          <select
            value={scrollType}
            onChange={(event) => handleTypeChange(event.target.value)}
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
              value={behavior}
              onChange={(event) => handleBehaviorChange(event.target.value)}
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
              value={direction}
              onChange={(event) => handleDirectionChange(event.target.value)}
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
              value={amount}
              onChange={(event) => setAmount(Number(event.target.value))}
              onBlur={handleAmountBlur}
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
              value={selector}
              placeholder=".list-panel"
              onChange={(event) => setSelector(event.target.value)}
              onBlur={handleSelectorBlur}
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
              value={targetSelector}
              onChange={(event) => setTargetSelector(event.target.value)}
              onBlur={handleTargetSelectorBlur}
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
              value={maxScrolls}
              onChange={(event) => setMaxScrolls(Number(event.target.value))}
              onBlur={handleMaxScrollsBlur}
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
                value={x}
                onChange={(event) => setX(Number(event.target.value))}
                onBlur={handleXBlur}
                className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
            </div>
            <div>
              <label className="text-gray-400 block mb-1">Y (px)</label>
              <input
                type="number"
                value={y}
                onChange={(event) => setY(Number(event.target.value))}
                onBlur={handleYBlur}
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
              value={timeoutMs}
              onChange={(event) => setTimeoutMs(Number(event.target.value))}
              onBlur={handleTimeoutBlur}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 block mb-1">Post-scroll wait (ms)</label>
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

        <p className="text-gray-500">
          Scroll nodes unlock lazy-loaded content, infinite lists, and stable viewport positioning for assertions and screenshots.
        </p>
      </div>

      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(ScrollNode);
