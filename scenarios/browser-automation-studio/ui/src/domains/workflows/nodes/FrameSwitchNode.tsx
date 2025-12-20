import { FC, memo, useMemo } from 'react';
import type { NodeProps } from 'reactflow';
import { LayoutPanelTop } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import {
  useSyncedString,
  useSyncedNumber,
  useSyncedSelect,
} from '@hooks/useSyncedField';
import BaseNode from './BaseNode';
import { NodeTextField, NodeNumberField, NodeSelectField } from './fields';

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
          <NodeSelectField field={mode} label="Switch strategy" options={MODES} />
          <p className="text-gray-500 mt-1">{modeDescription}</p>
        </div>

        {mode.value === 'selector' && (
          <NodeTextField
            field={selector}
            label="IFrame selector"
            placeholder="iframe[data-testid='editor']"
          />
        )}

        {mode.value === 'index' && (
          <NodeNumberField field={index} label="Frame index" min={0} />
        )}

        {mode.value === 'name' && (
          <NodeTextField field={frameName} label="Frame name" placeholder="paymentFrame" />
        )}

        {mode.value === 'url' && (
          <NodeTextField
            field={urlMatch}
            label="URL pattern"
            placeholder="/billing\\/portal/ or checkout"
          />
        )}

        <div>
          <NodeNumberField field={timeoutMs} label="Timeout (ms)" min={500} />
          <p className="text-gray-500 mt-1">Applies only to selector/index/name/url modes.</p>
        </div>
      </div>
    </BaseNode>
  );
};

export default memo(FrameSwitchNode);
