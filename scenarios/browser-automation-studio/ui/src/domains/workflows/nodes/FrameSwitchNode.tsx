import { FC, memo, useMemo } from 'react';
import type { NodeProps } from 'reactflow';
import { LayoutPanelTop } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import {
  useSyncedString,
  useSyncedNumber,
  useSyncedSelect,
  textInputHandler,
  numberInputHandler,
  selectInputHandler,
} from '@hooks/useSyncedField';
import BaseNode from './BaseNode';

// FrameSwitchParams interface for V2 native action params
interface FrameSwitchParams {
  switchBy?: string;
  selector?: string;
  name?: string;
  urlMatch?: string;
  index?: number;
  timeoutMs?: number;
}

const MODES = [
  { value: 'selector', label: 'By selector' },
  { value: 'index', label: 'By index' },
  { value: 'name', label: 'By name attribute' },
  { value: 'url', label: 'By URL match' },
  { value: 'parent', label: 'Parent frame' },
  { value: 'main', label: 'Main document' },
];

const DEFAULT_TIMEOUT = 30000;

const modeDescriptionMap: Record<string, string> = {
  selector: 'Queries the current document for an iframe using a CSS selector.',
  index: 'Targets the nth iframe within the active document (0 = first).',
  name: 'Matches iframe elements by the value of their name attribute.',
  url: 'Matches frames whose current URL contains the pattern or matches /regex/.',
  parent: 'Returns to the parent frame of the current context.',
  main: 'Clears frame context and returns to the top-level document.',
};

const FrameSwitchNode: FC<NodeProps> = ({ id, selected }) => {
  const { params, updateParams } = useActionParams<FrameSwitchParams>(id);

  // Select field
  const mode = useSyncedSelect(params?.switchBy ?? 'selector', {
    onCommit: (v) => updateParams({ switchBy: v }),
  });

  // String fields
  const selector = useSyncedString(params?.selector ?? '', {
    onCommit: (v) => updateParams({ selector: v || undefined }),
  });
  const frameName = useSyncedString(params?.name ?? '', {
    onCommit: (v) => updateParams({ name: v || undefined }),
  });
  const urlMatch = useSyncedString(params?.urlMatch ?? '', {
    onCommit: (v) => updateParams({ urlMatch: v || undefined }),
  });

  // Number fields
  const index = useSyncedNumber(params?.index ?? 0, {
    min: 0,
    onCommit: (v) => updateParams({ index: v }),
  });
  const timeoutMs = useSyncedNumber(params?.timeoutMs ?? DEFAULT_TIMEOUT, {
    min: 500,
    fallback: DEFAULT_TIMEOUT,
    onCommit: (v) => updateParams({ timeoutMs: v }),
  });

  const modeDescription = useMemo(
    () => modeDescriptionMap[mode.value] ?? modeDescriptionMap.selector,
    [mode.value],
  );

  return (
    <BaseNode selected={selected} icon={LayoutPanelTop} iconClassName="text-lime-300" title="Frame Switch">
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

        {mode.value === 'selector' && (
          <div>
            <label className="text-gray-400 block mb-1">IFrame selector</label>
            <input
              type="text"
              value={selector.value}
              onChange={textInputHandler(selector.setValue)}
              onBlur={selector.commit}
              placeholder="iframe[data-testid='editor']"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        {mode.value === 'index' && (
          <div>
            <label className="text-gray-400 block mb-1">Frame index</label>
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

        {mode.value === 'name' && (
          <div>
            <label className="text-gray-400 block mb-1">Frame name</label>
            <input
              type="text"
              value={frameName.value}
              onChange={textInputHandler(frameName.setValue)}
              onBlur={frameName.commit}
              placeholder="paymentFrame"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        {mode.value === 'url' && (
          <div>
            <label className="text-gray-400 block mb-1">URL pattern</label>
            <input
              type="text"
              value={urlMatch.value}
              onChange={textInputHandler(urlMatch.setValue)}
              onBlur={urlMatch.commit}
              placeholder="/billing\\/portal/ or checkout"
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        )}

        <div>
          <label className="text-gray-400 block mb-1">Timeout (ms)</label>
          <input
            type="number"
            min={500}
            value={timeoutMs.value}
            onChange={numberInputHandler(timeoutMs.setValue)}
            onBlur={timeoutMs.commit}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
          <p className="text-gray-500 mt-1">Applies only to selector/index/name/url modes.</p>
        </div>
      </div>
    </BaseNode>
  );
};

export default memo(FrameSwitchNode);
