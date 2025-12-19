import { memo, FC, useState } from 'react';
import type { NodeProps } from 'reactflow';
import { Database, Globe, Target } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useNodeData } from '@hooks/useNodeData';
import { useUrlInheritance } from '@hooks/useUrlInheritance';
import {
  useSyncedString,
  useSyncedSelect,
  textInputHandler,
  selectInputHandler,
} from '@hooks/useSyncedField';
import { ElementPickerModal } from '../components';
import type { ElementInfo } from '@/types/elements';
import BaseNode from './BaseNode';

// ExtractParams interface for V2 native action params
interface ExtractParams {
  selector?: string;
  extractType?: string;
  attribute?: string;
}

const ExtractNode: FC<NodeProps> = ({ selected, id }) => {
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

  // Node data hook for UI-specific fields
  const { updateData } = useNodeData(id);

  // V2 Native: Use action params as source of truth
  const { params, updateParams } = useActionParams<ExtractParams>(id);

  // Action params fields using useSyncedField
  const selector = useSyncedString(params?.selector ?? '', {
    onCommit: (v) => updateParams({ selector: v || undefined }),
  });
  const extractType = useSyncedSelect(params?.extractType ?? 'text', {
    onCommit: (v) => updateParams({ extractType: v }),
  });
  const attribute = useSyncedString(params?.attribute ?? '', {
    onCommit: (v) => updateParams({ attribute: v || undefined }),
  });

  // UI state
  const [showElementPicker, setShowElementPicker] = useState(false);

  const handleElementSelection = (newSelector: string, elementInfo: ElementInfo) => {
    selector.setValue(newSelector);
    updateParams({ selector: newSelector });
    updateData({ elementInfo });
  };

  return (
    <>
      <BaseNode selected={selected} icon={Database} iconClassName="text-pink-400" title="Extract Data">
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
            <p className="mt-1 text-[10px] text-red-400">Provide a URL to target this node.</p>
          )}
        </div>

        <div className="flex items-center gap-1 mb-2">
          <input
            type="text"
            placeholder="CSS Selector..."
            className="flex-1 px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
            value={selector.value}
            onChange={textInputHandler(selector.setValue)}
            onBlur={selector.commit}
          />
          <div
            className="inline-block"
            title={!effectiveUrl ? 'Set a page URL before picking elements' : ''}
          >
            <button
              onClick={() => effectiveUrl && setShowElementPicker(true)}
              className={`p-1.5 bg-flow-bg rounded border border-gray-700 transition-colors ${
                effectiveUrl ? 'hover:bg-gray-700 cursor-pointer' : 'opacity-50 cursor-not-allowed'
              }`}
              title={effectiveUrl ? `Pick element from ${effectiveUrl}` : undefined}
              disabled={!effectiveUrl}
            >
              <Target size={14} className={effectiveUrl ? 'text-gray-400' : 'text-gray-600'} />
            </button>
          </div>
        </div>

        <select
          className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none mb-2"
          value={extractType.value}
          onChange={selectInputHandler(extractType.setValue, extractType.commit)}
        >
          <option value="text">Text Content</option>
          <option value="attribute">Attribute</option>
          <option value="html">Inner HTML</option>
          <option value="value">Input Value</option>
        </select>

        {extractType.value === 'attribute' && (
          <input
            type="text"
            placeholder="Attribute name..."
            className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
            value={attribute.value}
            onChange={textInputHandler(attribute.setValue)}
            onBlur={attribute.commit}
          />
        )}
      </BaseNode>

      {showElementPicker && effectiveUrl && (
        <ElementPickerModal
          isOpen={showElementPicker}
          onClose={() => setShowElementPicker(false)}
          url={effectiveUrl}
          onSelectElement={handleElementSelection}
          selectedSelector={selector.value}
        />
      )}
    </>
  );
};

export default memo(ExtractNode);
