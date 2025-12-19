import { FC, memo, useMemo } from 'react';
import type { NodeProps } from 'reactflow';
import { PanelsTopLeft } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import {
  useSyncedString,
  useSyncedNumber,
  useSyncedBoolean,
  useSyncedSelect,
  textInputHandler,
  numberInputHandler,
  checkboxInputHandler,
  selectInputHandler,
} from '@hooks/useSyncedField';
import BaseNode from './BaseNode';

// TabSwitchParams interface for V2 native action params
interface TabSwitchParams {
  switchBy?: string;
  index?: number;
  titleMatch?: string;
  urlMatch?: string;
  waitForNew?: boolean;
  closeOld?: boolean;
  timeoutMs?: number;
}

const MODES = [
  { value: 'newest', label: 'Newest tab' },
  { value: 'oldest', label: 'Oldest tab' },
  { value: 'index', label: 'By index' },
  { value: 'title', label: 'By title match' },
  { value: 'url', label: 'By URL match' },
];

const DEFAULT_TIMEOUT = 30000;

const TabSwitchNode: FC<NodeProps> = ({ id, selected }) => {
  const { params, updateParams } = useActionParams<TabSwitchParams>(id);

  // Select field
  const mode = useSyncedSelect(params?.switchBy ?? 'newest', {
    onCommit: (v) => updateParams({ switchBy: v }),
  });

  // Number fields
  const index = useSyncedNumber(params?.index ?? 0, {
    min: 0,
    onCommit: (v) => updateParams({ index: v }),
  });
  const timeoutMs = useSyncedNumber(params?.timeoutMs ?? DEFAULT_TIMEOUT, {
    min: 1000,
    fallback: DEFAULT_TIMEOUT,
    onCommit: (v) => updateParams({ timeoutMs: v }),
  });

  // String fields
  const titleMatch = useSyncedString(params?.titleMatch ?? '', {
    onCommit: (v) => updateParams({ titleMatch: v || undefined }),
  });
  const urlMatch = useSyncedString(params?.urlMatch ?? '', {
    onCommit: (v) => updateParams({ urlMatch: v || undefined }),
  });

  // Boolean fields
  const waitForNew = useSyncedBoolean(params?.waitForNew ?? false, {
    onCommit: (v) => updateParams({ waitForNew: v }),
  });
  const closeOld = useSyncedBoolean(params?.closeOld ?? false, {
    onCommit: (v) => updateParams({ closeOld: v }),
  });

  const requiresIndex = mode.value === 'index';
  const requiresTitle = mode.value === 'title';
  const requiresUrl = mode.value === 'url';

  const modeDescription = useMemo(() => {
    switch (mode.value) {
      case 'oldest':
        return 'Selects the first tab that was opened in this workflow.';
      case 'index':
        return 'Zero-based index of the tab (0 = first tab).';
      case 'title':
        return 'Matches tabs by window title using substring or regex.';
      case 'url':
        return 'Matches tabs by URL using substring or regex.';
      default:
        return 'Switches to the most recently opened tab.';
    }
  }, [mode.value]);

  return (
    <BaseNode selected={selected} icon={PanelsTopLeft} iconClassName="text-violet-300" title="Tab Switch">
      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 block mb-1">Switch strategy</label>
          <select
            value={mode.value}
            onChange={selectInputHandler(mode.setValue, mode.commit)}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          >
            {MODES.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
          <p className="text-gray-500 mt-1">{modeDescription}</p>
        </div>

        {requiresIndex && (
          <div>
            <label className="text-gray-400 block mb-1">Tab index</label>
            <input
              type="number"
              min={0}
              value={index.value}
              onChange={numberInputHandler(index.setValue)}
              onBlur={index.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        {requiresTitle && (
          <div>
            <label className="text-gray-400 block mb-1">Title pattern</label>
            <input
              type="text"
              value={titleMatch.value}
              placeholder="/Invoice/ or Dashboard"
              onChange={textInputHandler(titleMatch.setValue)}
              onBlur={titleMatch.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        {requiresUrl && (
          <div>
            <label className="text-gray-400 block mb-1">URL pattern</label>
            <input
              type="text"
              value={urlMatch.value}
              placeholder="/auth\\/callback/ or /settings"
              onChange={textInputHandler(urlMatch.setValue)}
              onBlur={urlMatch.commit}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        <div className="grid grid-cols-2 gap-2">
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              className="accent-flow-accent"
              checked={waitForNew.value}
              onChange={checkboxInputHandler(waitForNew.setValue, waitForNew.commit)}
            />
            <span className="text-gray-400">Wait for new tab</span>
          </label>
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              className="accent-flow-accent"
              checked={closeOld.value}
              onChange={checkboxInputHandler(closeOld.setValue, closeOld.commit)}
            />
            <span className="text-gray-400">Close previous tab</span>
          </label>
        </div>

        <div>
          <label className="text-gray-400 block mb-1">Timeout (ms)</label>
          <input
            type="number"
            min={1000}
            value={timeoutMs.value}
            onChange={numberInputHandler(timeoutMs.setValue)}
            onBlur={timeoutMs.commit}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>

        <p className="text-gray-500">
          Switch workflows between pop-ups, OAuth flows, or background tabs. Pattern fields accept plain text
          or full regular expressions.
        </p>
      </div>
    </BaseNode>
  );
};

export default memo(TabSwitchNode);
