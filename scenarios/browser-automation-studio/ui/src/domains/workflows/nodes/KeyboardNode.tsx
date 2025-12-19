import { memo, FC, useCallback, useMemo, useState, useEffect } from 'react';
import type { NodeProps } from 'reactflow';
import { KeySquare } from 'lucide-react';
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
import type { KeyboardParams } from '@utils/actionBuilder';
import BaseNode from './BaseNode';

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
  const map = typeof raw === 'object' && raw !== null ? (raw as Record<string, unknown>) : {};
  return {
    ctrl: Boolean(map.ctrl),
    shift: Boolean(map.shift),
    alt: Boolean(map.alt),
    meta: Boolean(map.meta),
  };
};

const KeyboardNode: FC<NodeProps> = ({ selected, id }) => {
  // Node data hook for UI-specific fields
  const { getValue, updateData } = useNodeData(id);

  // V2 Native: Use action params as source of truth
  const { params, updateParams } = useActionParams<KeyboardParams>(id);

  // Action params fields
  const keyValue = useSyncedString(params?.key ?? '', {
    onCommit: (v) => updateParams({ key: v || undefined }),
  });

  // UI-specific fields (stored in node.data)
  const eventType = useSyncedSelect(getValue<string>('eventType') ?? 'keypress', {
    onCommit: (v) => updateData({ eventType: v }),
  });
  const delayMs = useSyncedNumber(getValue<number>('delayMs') ?? 0, {
    min: 0,
    onCommit: (v) => updateData({ delayMs: v }),
  });
  const timeoutMs = useSyncedNumber(getValue<number>('timeoutMs') ?? 30000, {
    min: 100,
    fallback: 30000,
    onCommit: (v) => updateData({ timeoutMs: v }),
  });

  // Modifiers need special handling since it's an object
  const [modifiers, setModifiers] = useState<ModifierState>(() =>
    normalizeModifiers(getValue<ModifierState>('modifiers')),
  );

  useEffect(() => {
    setModifiers(normalizeModifiers(getValue<ModifierState>('modifiers')));
  }, [getValue]);

  const handleModifierToggle = useCallback(
    (field: keyof ModifierState) => {
      setModifiers((prev) => {
        const next = { ...prev, [field]: !prev[field] };
        updateData({ modifiers: next });
        return next;
      });
    },
    [updateData],
  );

  const datalistId = useMemo(() => `keyboard-node-keys-${id}`, [id]);

  return (
    <BaseNode selected={selected} icon={KeySquare} iconClassName="text-lime-300" title="Keyboard">
      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">Key</label>
          <input
            type="text"
            list={datalistId}
            placeholder="Enter, ArrowDown, a..."
            value={keyValue.value}
            onChange={textInputHandler(keyValue.setValue)}
            onBlur={keyValue.commit}
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
            value={eventType.value}
            onChange={selectInputHandler(eventType.setValue, eventType.commit)}
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
              <label
                key={mod}
                className="flex items-center gap-2 px-2 py-1 bg-flow-bg/60 rounded border border-gray-700 cursor-pointer select-none"
              >
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
              value={delayMs.value}
              onChange={numberInputHandler(delayMs.setValue)}
              onBlur={delayMs.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
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
        </div>

        <p className="text-gray-500">
          Use press mode for sequences (down → optional hold → up). Toggle modifiers to simulate
          combos like Ctrl+Enter.
        </p>
      </div>
    </BaseNode>
  );
};

export default memo(KeyboardNode);
