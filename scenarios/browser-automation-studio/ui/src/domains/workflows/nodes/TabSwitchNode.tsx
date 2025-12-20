import { FC, memo, useMemo } from 'react';
import type { NodeProps } from 'reactflow';
import { PanelsTopLeft } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import {
  useSyncedString,
  useSyncedNumber,
  useSyncedBoolean,
  useSyncedSelect,
} from '@hooks/useSyncedField';
import BaseNode from './BaseNode';
import { NodeTextField, NodeNumberField, NodeSelectField, NodeCheckbox, FieldRow } from './fields';

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

  // Select fields
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
          <NodeSelectField field={mode} label="Switch strategy" options={MODES} />
          <p className="text-gray-500 mt-1">{modeDescription}</p>
        </div>

        {requiresIndex && (
          <NodeNumberField field={index} label="Tab index" min={0} />
        )}

        {requiresTitle && (
          <NodeTextField
            field={titleMatch}
            label="Title pattern"
            placeholder="/Invoice/ or Dashboard"
          />
        )}

        {requiresUrl && (
          <NodeTextField
            field={urlMatch}
            label="URL pattern"
            placeholder="/auth\\/callback/ or /settings"
          />
        )}

        <FieldRow>
          <NodeCheckbox field={waitForNew} label="Wait for new tab" />
          <NodeCheckbox field={closeOld} label="Close previous tab" />
        </FieldRow>

        <NodeNumberField field={timeoutMs} label="Timeout (ms)" min={1000} />

        <p className="text-gray-500">
          Switch workflows between pop-ups, OAuth flows, or background tabs. Pattern fields accept plain text
          or full regular expressions.
        </p>
      </div>
    </BaseNode>
  );
};

export default memo(TabSwitchNode);
