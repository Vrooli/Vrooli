import { FC, memo, useMemo, useCallback } from 'react';
import type { NodeProps } from 'reactflow';
import { Command } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import { useUrlInheritance } from '@hooks/useUrlInheritance';
import { useSyncedString, useSyncedNumber } from '@hooks/useSyncedField';
import type { ElementInfo } from '@/types/elements';
import BaseNode from './BaseNode';
import { NodeTextArea, NodeNumberField, NodeUrlField, NodeSelectorField } from './fields';

// ShortcutParams interface for V2 native action params
interface ShortcutParams {
  shortcuts?: string[];
  shortcut?: string;
  focusSelector?: string;
  focusElement?: ElementInfo;
  shortcutDelayMs?: number;
  delayMs?: number;
}

const KEY_ALIASES: Record<string, string> = {
  ctrl: 'Control',
  control: 'Control',
  cmd: 'Meta',
  command: 'Meta',
  meta: 'Meta',
  super: 'Meta',
  windows: 'Meta',
  win: 'Meta',
  alt: 'Alt',
  option: 'Alt',
  shift: 'Shift',
  enter: 'Enter',
  return: 'Enter',
  esc: 'Escape',
  escape: 'Escape',
  space: 'Space',
  spacebar: 'Space',
  tab: 'Tab',
  backspace: 'Backspace',
  delete: 'Delete',
  del: 'Delete',
  home: 'Home',
  end: 'End',
  pageup: 'PageUp',
  pagedown: 'PageDown',
  arrowup: 'ArrowUp',
  arrowdown: 'ArrowDown',
  arrowleft: 'ArrowLeft',
  arrowright: 'ArrowRight',
  up: 'ArrowUp',
  down: 'ArrowDown',
  left: 'ArrowLeft',
  right: 'ArrowRight',
  plus: '+',
};

const normalizeShortcutSegment = (segment: string): string => {
  const trimmed = segment.trim();
  if (!trimmed) {
    return '';
  }
  const lower = trimmed.toLowerCase().replace(/-/g, '');
  if (KEY_ALIASES[lower]) {
    return KEY_ALIASES[lower];
  }
  if (lower.startsWith('f') && lower.length > 1 && /^[f][0-9]+$/.test(lower)) {
    return lower.toUpperCase();
  }
  if (trimmed.length === 1) {
    return trimmed.toUpperCase();
  }
  return trimmed.slice(0, 1).toUpperCase() + trimmed.slice(1);
};

const normalizeShortcutCombo = (combo: string): string => {
  if (!combo || typeof combo !== 'string') {
    return '';
  }
  const parts = combo
    .split('+')
    .map((part) => normalizeShortcutSegment(part))
    .filter(Boolean);
  if (parts.length === 0) {
    return '';
  }
  return Array.from(new Set(parts)).join('+');
};

const parseShortcutText = (value: string): string[] => {
  if (!value) {
    return [];
  }
  const entries = value
    .split(/\r?\n|,/)
    .map((entry) => normalizeShortcutCombo(entry))
    .filter(Boolean);

  const seen = new Set<string>();
  const result: string[] = [];
  for (const entry of entries) {
    if (!seen.has(entry)) {
      seen.add(entry);
      result.push(entry);
    }
  }
  return result;
};

const formatShortcutText = (shortcuts: unknown): string => {
  if (!Array.isArray(shortcuts)) {
    return '';
  }
  const normalized = shortcuts
    .map((item) => (typeof item === 'string' ? normalizeShortcutCombo(item) : ''))
    .filter(Boolean);
  return normalized.join('\n');
};

const ShortcutNode: FC<NodeProps> = ({ selected, id }) => {
  // URL inheritance for element picker
  const { effectiveUrl } = useUrlInheritance(id);

  // Action params and node data hooks
  const { params, updateParams } = useActionParams<ShortcutParams>(id);
  const { updateData } = useNodeData(id);

  // Shortcuts as text (special handling for array)
  const shortcutText = useSyncedString(formatShortcutText(params?.shortcuts ?? params?.shortcut), {
    trim: false,
    onCommit: (v) => {
      const sanitized = parseShortcutText(v);
      updateParams({ shortcuts: sanitized });
    },
  });

  // Focus selector
  const focusSelector = useSyncedString(params?.focusSelector ?? '', {
    onCommit: (v) => updateParams({ focusSelector: v || undefined }),
  });

  // Delay
  const delayMs = useSyncedNumber(params?.shortcutDelayMs ?? params?.delayMs ?? 0, {
    min: 0,
    onCommit: (v) => updateParams({ shortcutDelayMs: v || undefined }),
  });

  const shortcutsPreview = useMemo(() => parseShortcutText(shortcutText.value), [shortcutText.value]);

  const handleElementSelection = useCallback(
    (selector: string, elementInfo: ElementInfo) => {
      updateParams({ focusSelector: selector });
      updateData({ focusElement: elementInfo });
    },
    [updateParams, updateData],
  );

  const handleClearFocus = useCallback(() => {
    updateParams({ focusSelector: undefined });
    updateData({ focusElement: undefined });
  }, [updateParams, updateData]);

  return (
    <BaseNode
      selected={selected}
      icon={Command}
      iconClassName="text-indigo-400"
      title="Keyboard Shortcut"
    >
      <NodeUrlField nodeId={id} errorMessage="Provide a URL to execute this shortcut." />

      <NodeTextArea
        field={shortcutText}
        label="Shortcuts"
        rows={3}
        placeholder="Ctrl+Shift+P\nAlt+Tab"
        description="Separate combinations by new lines"
      />

      <div className="space-y-3">
        <NodeSelectorField
          field={focusSelector}
          effectiveUrl={effectiveUrl}
          onElementSelect={handleElementSelection}
          onClear={handleClearFocus}
          label="Focus Selector (optional)"
          placeholder="CSS selector to focus before shortcut"
          showClear
        />

        <NodeNumberField
          field={delayMs}
          label="Delay Between Combos (ms)"
          min={0}
          step={50}
        />
      </div>

      {shortcutsPreview.length > 0 && (
        <div className="mt-3 p-2 bg-flow-bg/70 border border-gray-800 rounded text-[11px] text-gray-400">
          <span className="block mb-1 text-gray-500 uppercase tracking-wide">Will trigger</span>
          <ul className="space-y-1">
            {shortcutsPreview.map((combo) => (
              <li key={combo} className="font-mono text-gray-200">
                {combo}
              </li>
            ))}
          </ul>
        </div>
      )}
    </BaseNode>
  );
};

export default memo(ShortcutNode);
