import { memo, FC, useCallback, useEffect, useMemo, useState } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { KeySquare } from 'lucide-react';

const KEY_SUGGESTIONS = [
  'Enter',
  'Escape',
  'Tab',
  'Backspace',
  'Space',
  'ArrowUp',
  'ArrowDown',
  'ArrowLeft',
  'ArrowRight',
  'Delete',
  'Home',
  'End',
  'PageUp',
  'PageDown',
  'F1',
  'F2',
  'F3',
  'F4',
  'F5',
  'F6',
  'F7',
  'F8',
  'F9',
  'F10',
  'F11',
  'F12',
];

type ModifierState = {
  ctrl: boolean;
  shift: boolean;
  alt: boolean;
  meta: boolean;
};

const normalizeModifiers = (raw: unknown): ModifierState => {
  const map = typeof raw === 'object' && raw !== null ? raw as Record<string, unknown> : {};
  return {
    ctrl: Boolean(map.ctrl),
    shift: Boolean(map.shift),
    alt: Boolean(map.alt),
    meta: Boolean(map.meta),
  };
};

const KeyboardNode: FC<NodeProps> = ({ data, selected, id }) => {
  const { getNodes, setNodes } = useReactFlow();
  const nodeData = (data ?? {}) as Record<string, unknown>;

  const [keyValue, setKeyValue] = useState<string>(() => typeof nodeData.key === 'string' ? nodeData.key : '');
  const [eventType, setEventType] = useState<string>(() => typeof nodeData.eventType === 'string' ? nodeData.eventType : 'keypress');
  const [delayMs, setDelayMs] = useState<number>(() => Number(nodeData.delayMs ?? 0));
  const [timeoutMs, setTimeoutMs] = useState<number>(() => Number(nodeData.timeoutMs ?? 30000));
  const [modifiers, setModifiers] = useState<ModifierState>(() => normalizeModifiers(nodeData.modifiers));

  useEffect(() => {
    setKeyValue(typeof nodeData.key === 'string' ? nodeData.key : '');
  }, [nodeData.key]);

  useEffect(() => {
    setEventType(typeof nodeData.eventType === 'string' ? nodeData.eventType : 'keypress');
  }, [nodeData.eventType]);

  useEffect(() => {
    setDelayMs(Number(nodeData.delayMs ?? 0));
  }, [nodeData.delayMs]);

  useEffect(() => {
    setTimeoutMs(Number(nodeData.timeoutMs ?? 30000));
  }, [nodeData.timeoutMs]);

  useEffect(() => {
    setModifiers(normalizeModifiers(nodeData.modifiers));
  }, [nodeData.modifiers]);

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

  const handleModifierToggle = useCallback((field: keyof ModifierState) => {
    setModifiers((prev) => {
      const next = { ...prev, [field]: !prev[field] };
      updateNodeData({ modifiers: next });
      return next;
    });
  }, [updateNodeData]);

  const datalistId = useMemo(() => `keyboard-node-keys-${id}`, [id]);

  const handleKeyBlur = useCallback(() => {
    updateNodeData({ key: keyValue.trim() });
  }, [keyValue, updateNodeData]);

  const handleEventTypeChange = (value: string) => {
    setEventType(value);
    updateNodeData({ eventType: value });
  };

  const handleDelayBlur = useCallback(() => {
    const next = Math.max(0, Math.round(delayMs) || 0);
    setDelayMs(next);
    updateNodeData({ delayMs: next });
  }, [delayMs, updateNodeData]);

  const handleTimeoutBlur = useCallback(() => {
    const next = Math.max(100, Math.round(timeoutMs) || 100);
    setTimeoutMs(next);
    updateNodeData({ timeoutMs: next });
  }, [timeoutMs, updateNodeData]);

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />

      <div className="flex items-center gap-2 mb-2">
        <KeySquare size={16} className="text-lime-300" />
        <span className="font-semibold text-sm">Keyboard</span>
      </div>

      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">Key</label>
          <input
            type="text"
            list={datalistId}
            placeholder="Enter, ArrowDown, a..."
            value={keyValue}
            onChange={(event) => setKeyValue(event.target.value)}
            onBlur={handleKeyBlur}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
          <datalist id={datalistId}>
            {KEY_SUGGESTIONS.map((option) => (
              <option value={option} key={option} />
            ))}
          </datalist>
        </div>

        <div>
          <label className="text-gray-400 block mb-1">Event Type</label>
          <select
            value={eventType}
            onChange={(event) => handleEventTypeChange(event.target.value)}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          >
            <option value="keypress">Key press (down → up)</option>
            <option value="keydown">Key down only</option>
            <option value="keyup">Key up only</option>
          </select>
        </div>

        <div>
          <label className="text-gray-400 block mb-1">Modifiers</label>
          <div className="grid grid-cols-2 gap-2">
            {(['ctrl', 'shift', 'alt', 'meta'] as (keyof ModifierState)[]).map((mod) => (
              <label key={mod} className="flex items-center gap-2 px-2 py-1 bg-flow-bg/60 rounded border border-gray-700 cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={modifiers[mod]}
                  onChange={() => handleModifierToggle(mod)}
                  className="accent-flow-accent"
                />
                <span className="capitalize">{mod}</span>
              </label>
            ))}
          </div>
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 block mb-1">Hold Delay (ms)</label>
            <input
              type="number"
              min={0}
              value={delayMs}
              onChange={(event) => setDelayMs(Number(event.target.value))}
              onBlur={handleDelayBlur}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
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
        </div>

        <p className="text-gray-500">
          Use press mode for sequences (down → optional hold → up). Toggle modifiers to simulate combos like Ctrl+Enter.
        </p>
      </div>

      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(KeyboardNode);
