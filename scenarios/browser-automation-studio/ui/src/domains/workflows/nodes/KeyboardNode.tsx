import { memo, FC, useMemo } from 'react';
import type { NodeProps } from 'reactflow';
import { KeySquare } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import {
  useSyncedString,
  useSyncedNumber,
  useSyncedSelect,
  useSyncedObject,
  textInputHandler,
} from '@hooks/useSyncedField';
import type { KeyboardParams } from '@utils/actionBuilder';
import BaseNode from './BaseNode';
import { NodeNumberField, NodeSelectField, FieldRow } from './fields';

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

const EVENT_TYPE_OPTIONS = [
  { value: 'keypress', label: 'Key press (down → up)' },
  { value: 'keydown', label: 'Key down only' },
  { value: 'keyup', label: 'Key up only' },
];

type ModifierState = {
  ctrl: boolean;
  shift: boolean;
  alt: boolean;
  meta: boolean;
};

const DEFAULT_MODIFIERS: ModifierState = {
  ctrl: false,
  shift: false,
  alt: false,
  meta: false,
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

  // Object field using useSyncedObject (replaces manual useState + useEffect + toggle)
  const modifiers = useSyncedObject<ModifierState>(
    normalizeModifiers(getValue<ModifierState>('modifiers')) ?? DEFAULT_MODIFIERS,
    { onCommit: (v) => updateData({ modifiers: v }) },
  );

  const datalistId = useMemo(() => `keyboard-node-keys-${id}`, [id]);

  return (
    <BaseNode selected={selected} icon={KeySquare} iconClassName="text-lime-300" title="Keyboard">
      <div className="space-y-3 text-xs">
        {/* Key field with datalist - custom since it has autocomplete */}
        <div>
          <label className="text-gray-400 block mb-1 text-xs">Key</label>
          <input
            type="text"
            list={datalistId}
            placeholder="Enter, ArrowDown, a..."
            value={keyValue.value}
            onChange={textInputHandler(keyValue.setValue)}
            onBlur={keyValue.commit}
            className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
          <datalist id={datalistId}>
            {KEY_SUGGESTIONS.map((option) => (
              <option value={option} key={option} />
            ))}
          </datalist>
        </div>

        <NodeSelectField field={eventType} label="Event Type" options={EVENT_TYPE_OPTIONS} />

        <div>
          <label className="text-gray-400 block mb-1 text-xs">Modifiers</label>
          <div className="grid grid-cols-2 gap-2">
            {(['ctrl', 'shift', 'alt', 'meta'] as (keyof ModifierState)[]).map((mod) => (
              <label
                key={mod}
                className="flex items-center gap-2 px-2 py-1 bg-flow-bg/60 rounded border border-gray-700 cursor-pointer select-none"
              >
                <input
                  type="checkbox"
                  checked={modifiers.value[mod]}
                  onChange={() => modifiers.toggle(mod)}
                  className="accent-flow-accent"
                />
                <span className="capitalize">{mod}</span>
              </label>
            ))}
          </div>
        </div>

        <FieldRow>
          <NodeNumberField field={delayMs} label="Hold Delay (ms)" min={0} />
          <NodeNumberField field={timeoutMs} label="Timeout (ms)" min={100} />
        </FieldRow>

        <p className="text-gray-500">
          Use press mode for sequences (down → optional hold → up). Toggle modifiers to simulate
          combos like Ctrl+Enter.
        </p>
      </div>
    </BaseNode>
  );
};

export default memo(KeyboardNode);
