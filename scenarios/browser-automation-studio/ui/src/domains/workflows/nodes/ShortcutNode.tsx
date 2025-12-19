import { FC, memo, useMemo, useState } from 'react';
import type { NodeProps } from 'reactflow';
import { Command, Globe, Target, Clock, X as XIcon } from 'lucide-react';
import clsx from 'clsx';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import { useUrlInheritance } from '@hooks/useUrlInheritance';
import {
  useSyncedString,
  useSyncedNumber,
  textInputHandler,
  numberInputHandler,
} from '@hooks/useSyncedField';
import { ElementPickerModal } from '../components';
import type { ElementInfo } from '@/types/elements';
import BaseNode from './BaseNode';

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
  // URL inheritance hook handles URL state and handlers
  const {
    urlDraft,
    setUrlDraft,
    effectiveUrl,
    hasCustomUrl,
    upstreamUrl,
    commitUrl,
    resetUrl,
    handleUrlKeyDown,
  } = useUrlInheritance(id);

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

  const [showPicker, setShowPicker] = useState(false);

  const shortcutsPreview = useMemo(() => parseShortcutText(shortcutText.value), [shortcutText.value]);

  const handleElementSelection = (selector: string, elementInfo: ElementInfo) => {
    const normalizedSelector = selector.trim();
    focusSelector.setValue(normalizedSelector);
    updateParams({ focusSelector: normalizedSelector });
    updateData({ focusElement: elementInfo });
  };

  const clearFocus = () => {
    focusSelector.setValue('');
    updateParams({ focusSelector: undefined });
    updateData({ focusElement: undefined });
  };

  return (
    <>
      <BaseNode
        selected={selected}
        icon={Command}
        iconClassName="text-indigo-400"
        title="Keyboard Shortcut"
      >
        <div className="mb-2">
          <label className="flex items-center gap-1 text-[11px] font-semibold uppercase tracking-wide text-gray-400">
            <Globe size={12} className="text-blue-400" />
            Page URL
          </label>
          <div className="mt-1 flex items-center gap-2">
            <input
              type="text"
              placeholder={upstreamUrl ?? 'https://example.com'}
              className="flex-1 px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
              value={urlDraft}
              onChange={(event) => setUrlDraft(event.target.value)}
              onBlur={commitUrl}
              onKeyDown={handleUrlKeyDown}
            />
            {hasCustomUrl && (
              <button
                type="button"
                className="px-2 py-1 text-[11px] rounded border border-gray-700 text-gray-300 hover:bg-gray-700 transition-colors"
                onClick={resetUrl}
              >
                Reset
              </button>
            )}
          </div>
          {!hasCustomUrl && upstreamUrl && (
            <p className="mt-1 text-[10px] text-gray-500 truncate" title={upstreamUrl}>
              Inherits {upstreamUrl}
            </p>
          )}
          {!effectiveUrl && !upstreamUrl && (
            <p className="mt-1 text-[10px] text-red-400">Provide a URL to execute this shortcut.</p>
          )}
        </div>

        <label className="block mb-2">
          <span className="text-[11px] uppercase tracking-wide text-gray-500 flex items-center justify-between">
            Shortcuts
            <span className="text-gray-600 normal-case font-normal">
              Separate combinations by new lines
            </span>
          </span>
          <textarea
            className="mt-1 w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none resize-none"
            rows={3}
            placeholder="Ctrl+Shift+P\nAlt+Tab"
            value={shortcutText.value}
            onChange={textInputHandler(shortcutText.setValue)}
            onBlur={shortcutText.commit}
          />
        </label>

        <div className="grid grid-cols-1 gap-3">
          <div>
            <label className="text-[11px] uppercase tracking-wide text-gray-500 block mb-1">
              Focus Selector (optional)
            </label>
            <div className="flex items-center gap-1">
              <input
                type="text"
                value={focusSelector.value}
                onChange={textInputHandler(focusSelector.setValue)}
                onBlur={focusSelector.commit}
                placeholder="CSS selector to focus before shortcut"
                className="flex-1 px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
              <button
                type="button"
                className={clsx(
                  'p-1.5 rounded border transition-colors',
                  effectiveUrl
                    ? 'border-gray-700 bg-flow-bg hover:bg-gray-700 text-gray-300'
                    : 'border-gray-800 bg-flow-bg text-gray-600 cursor-not-allowed'
                )}
                onClick={() => effectiveUrl && setShowPicker(true)}
                disabled={!effectiveUrl}
                title={effectiveUrl ? 'Pick focus element' : 'Set a page URL to pick elements'}
              >
                <Target size={14} />
              </button>
              {focusSelector.value && (
                <button
                  type="button"
                  className="p-1.5 rounded border border-gray-700 bg-flow-bg text-gray-300 hover:bg-gray-700"
                  onClick={clearFocus}
                  title="Clear focus selector"
                >
                  <XIcon size={12} />
                </button>
              )}
            </div>
          </div>

          <div>
            <label className="text-[11px] uppercase tracking-wide text-gray-500 block mb-1">
              Delay Between Combos (ms)
            </label>
            <div className="flex items-center gap-2">
              <div className="px-2 py-1 bg-flow-bg rounded border border-gray-700 text-gray-400">
                <Clock size={12} />
              </div>
              <input
                type="number"
                min={0}
                step={50}
                className="flex-1 px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
                value={delayMs.value}
                onChange={numberInputHandler(delayMs.setValue)}
                onBlur={delayMs.commit}
              />
            </div>
          </div>
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

      {showPicker && effectiveUrl && (
        <ElementPickerModal
          isOpen={showPicker}
          onClose={() => setShowPicker(false)}
          url={effectiveUrl}
          onSelectElement={handleElementSelection}
          selectedSelector={focusSelector.value}
        />
      )}
    </>
  );
};

export default memo(ShortcutNode);
