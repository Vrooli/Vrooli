import { FC, memo, useCallback, useEffect, useMemo, useState } from 'react';
import type { KeyboardEvent } from 'react';
import { Handle, Position, NodeProps, useReactFlow } from 'reactflow';
import { Command, Globe, Target, Clock, X as XIcon } from 'lucide-react';
import clsx from 'clsx';
import { useUpstreamUrl } from '@hooks/useUpstreamUrl';
import { ElementPickerModal } from '../components';
import type { ElementInfo } from '@/types/elements';

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

const sanitizeDelay = (value: number | string): number => {
  const parsed = typeof value === 'number' ? value : Number.parseInt(String(value), 10);
  if (!Number.isFinite(parsed) || parsed < 0) {
    return 0;
  }
  return parsed;
};

const ShortcutNode: FC<NodeProps> = ({ data, selected, id }) => {
  const upstreamUrl = useUpstreamUrl(id);
  const { getNodes, setNodes } = useReactFlow();

  const nodeData = data as Record<string, unknown> | undefined;
  const storedUrl = typeof nodeData?.url === 'string' ? nodeData.url : '';
  const [urlDraft, setUrlDraft] = useState<string>(storedUrl);

  useEffect(() => {
    setUrlDraft(storedUrl);
  }, [storedUrl]);

  const trimmedStoredUrl = useMemo(() => storedUrl.trim(), [storedUrl]);
  const effectiveUrl = useMemo(() => {
    if (trimmedStoredUrl.length > 0) {
      return trimmedStoredUrl;
    }
    return upstreamUrl ?? null;
  }, [trimmedStoredUrl, upstreamUrl]);
  const hasCustomUrl = trimmedStoredUrl.length > 0;

  const [showPicker, setShowPicker] = useState(false);
  const [shortcutText, setShortcutText] = useState<string>(() => formatShortcutText(data.shortcuts ?? data.shortcut));
  const [focusSelector, setFocusSelector] = useState<string>(data.focusSelector || '');
  const [delayMs, setDelayMs] = useState<number>(() => sanitizeDelay(data.shortcutDelayMs ?? data.delayMs));

  useEffect(() => {
    setShortcutText(formatShortcutText(data.shortcuts ?? data.shortcut));
  }, [data.shortcuts, data.shortcut]);

  useEffect(() => {
    setFocusSelector(data.focusSelector || '');
  }, [data.focusSelector]);

  useEffect(() => {
    setDelayMs(sanitizeDelay(data.shortcutDelayMs ?? data.delayMs));
  }, [data.shortcutDelayMs, data.delayMs]);

  const updateNodeData = useCallback((updates: Record<string, unknown>) => {
    const nodes = getNodes();
    const updatedNodes = nodes.map((node) => {
      if (node.id !== id) {
        return node;
      }

      const nextData = { ...(node.data ?? {}) } as Record<string, unknown>;

      for (const [key, value] of Object.entries(updates)) {
        if (key === 'url') {
          const trimmed = typeof value === 'string' ? value.trim() : '';
          if (trimmed) {
            nextData.url = trimmed;
          } else {
            delete nextData.url;
          }
          continue;
        }

        if (value === undefined || value === null) {
          delete nextData[key];
        } else {
          nextData[key] = value;
        }
      }

      return {
        ...node,
        data: nextData,
      };
    });
    setNodes(updatedNodes);
  }, [getNodes, id, setNodes]);

  const shortcutsPreview = useMemo(() => parseShortcutText(shortcutText), [shortcutText]);

  const commitUrl = useCallback((raw: string) => {
    const trimmed = raw.trim();
    updateNodeData({ url: trimmed.length > 0 ? trimmed : null });
  }, [updateNodeData]);

  const handleUrlCommit = useCallback(() => {
    commitUrl(urlDraft);
  }, [commitUrl, urlDraft]);

  const handleUrlKeyDown = useCallback((event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter') {
      event.preventDefault();
      commitUrl(urlDraft);
    } else if (event.key === 'Escape') {
      event.preventDefault();
      setUrlDraft(storedUrl);
    }
  }, [commitUrl, storedUrl, urlDraft]);

  const resetUrl = useCallback(() => {
    setUrlDraft('');
    updateNodeData({ url: null });
  }, [updateNodeData]);

  const handleElementSelection = (selector: string, elementInfo: ElementInfo) => {
    const normalizedSelector = selector.trim();
    setFocusSelector(normalizedSelector);
    updateNodeData({ focusSelector: normalizedSelector, focusElement: elementInfo });
  };

  const handleShortcutsBlur = () => {
    const sanitized = parseShortcutText(shortcutText);
    updateNodeData({ shortcuts: sanitized });
    setShortcutText(sanitized.join('\n'));
  };

  const handleDelayBlur = () => {
    const sanitized = sanitizeDelay(delayMs);
    setDelayMs(sanitized);
    updateNodeData({ shortcutDelayMs: sanitized });
  };

  const clearFocus = () => {
    setFocusSelector('');
    updateNodeData({ focusSelector: '', focusElement: undefined });
  };

  return (
    <>
      <div className={clsx('workflow-node', selected && 'selected')}>
        <Handle type="target" position={Position.Top} className="node-handle" />

        <div className="flex items-center gap-2 mb-2">
          <Command size={16} className="text-indigo-400" />
          <span className="font-semibold text-sm">Keyboard Shortcut</span>
        </div>

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
              onBlur={handleUrlCommit}
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
            <span className="text-gray-600 normal-case font-normal">Separate combinations by new lines</span>
          </span>
          <textarea
            className="mt-1 w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none resize-none"
            rows={3}
            placeholder="Ctrl+Shift+P\nAlt+Tab"
            value={shortcutText}
            onChange={(event) => setShortcutText(event.target.value)}
            onBlur={handleShortcutsBlur}
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
                value={focusSelector}
                onChange={(event) => setFocusSelector(event.target.value)}
                onBlur={() => updateNodeData({ focusSelector: focusSelector.trim() })}
                placeholder="CSS selector to focus before shortcut"
                className="flex-1 px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
              />
              <button
                type="button"
                className={clsx('p-1.5 rounded border transition-colors', effectiveUrl ? 'border-gray-700 bg-flow-bg hover:bg-gray-700 text-gray-300' : 'border-gray-800 bg-flow-bg text-gray-600 cursor-not-allowed')}
                onClick={() => effectiveUrl && setShowPicker(true)}
                disabled={!effectiveUrl}
                title={effectiveUrl ? 'Pick focus element' : 'Set a page URL to pick elements'}
              >
                <Target size={14} />
              </button>
              {focusSelector && (
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
                value={delayMs}
                onChange={(event) => setDelayMs(sanitizeDelay(event.target.value))}
                onBlur={handleDelayBlur}
              />
            </div>
          </div>
        </div>

        {shortcutsPreview.length > 0 && (
          <div className="mt-3 p-2 bg-flow-bg/70 border border-gray-800 rounded text-[11px] text-gray-400">
            <span className="block mb-1 text-gray-500 uppercase tracking-wide">Will trigger</span>
            <ul className="space-y-1">
              {shortcutsPreview.map((combo) => (
                <li key={combo} className="font-mono text-gray-200">{combo}</li>
              ))}
            </ul>
          </div>
        )}

        <Handle type="source" position={Position.Bottom} className="node-handle" />
      </div>

      {showPicker && effectiveUrl && (
        <ElementPickerModal
          isOpen={showPicker}
          onClose={() => setShowPicker(false)}
          url={effectiveUrl!}
          onSelectElement={handleElementSelection}
          selectedSelector={focusSelector}
        />
      )}
    </>
  );
};

export default memo(ShortcutNode);
