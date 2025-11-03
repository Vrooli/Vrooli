import { memo, FC, useState, useEffect, useMemo, useCallback } from 'react';
import type { KeyboardEvent } from 'react';
import { Handle, Position, NodeProps, useReactFlow } from 'reactflow';
import { Database, Globe, Target } from 'lucide-react';
import { useUpstreamUrl } from '../../hooks/useUpstreamUrl';
import ElementPickerModal from '../ElementPickerModal';
import type { ElementInfo } from '../../types/elements';

const ExtractNode: FC<NodeProps> = ({ data, selected, id }) => {
  const upstreamUrl = useUpstreamUrl(id);
  const [showElementPicker, setShowElementPicker] = useState(false);
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

  const [selector, setSelector] = useState(data.selector || '');
  const [extractType, setExtractType] = useState(data.extractType || 'text');
  const [attribute, setAttribute] = useState(data.attribute || '');

  useEffect(() => {
    setSelector(data.selector || '');
  }, [data.selector]);

  useEffect(() => {
    setExtractType(data.extractType || 'text');
  }, [data.extractType]);

  useEffect(() => {
    setAttribute(data.attribute || '');
  }, [data.attribute]);

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
    setSelector(selector);
    updateNodeData({ selector, elementInfo });
  };

  return (
    <>
      <div className={`workflow-node ${selected ? 'selected' : ''}`}>
        <Handle type="target" position={Position.Top} className="node-handle" />
        
        <div className="flex items-center gap-2 mb-2">
          <Database size={16} className="text-pink-400" />
          <span className="font-semibold text-sm">Extract Data</span>
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
            <p className="mt-1 text-[10px] text-red-400">Provide a URL to target this node.</p>
          )}
        </div>
        
        <div className="flex items-center gap-1 mb-2">
          <input
            type="text"
            placeholder="CSS Selector..."
            className="flex-1 px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
            value={selector}
            onChange={(e) => setSelector(e.target.value)}
            onBlur={() => updateNodeData({ selector })}
          />
          <div 
            className="inline-block"
            title={!effectiveUrl ? "Set a page URL before picking elements" : ""}
          >
            <button
              onClick={() => effectiveUrl && setShowElementPicker(true)}
              className={`p-1.5 bg-flow-bg rounded border border-gray-700 transition-colors ${
                effectiveUrl ? 'hover:bg-gray-700 cursor-pointer' : 'opacity-50 cursor-not-allowed'
              }`}
              title={effectiveUrl ? `Pick element from ${effectiveUrl}` : undefined}
              disabled={!effectiveUrl}
            >
              <Target size={14} className={effectiveUrl ? "text-gray-400" : "text-gray-600"} />
            </button>
          </div>
        </div>
        
        <select
          className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none mb-2"
          value={extractType}
          onChange={(e) => {
            const newType = e.target.value;
            setExtractType(newType);
            updateNodeData({ extractType: newType });
          }}
        >
          <option value="text">Text Content</option>
          <option value="attribute">Attribute</option>
          <option value="html">Inner HTML</option>
          <option value="value">Input Value</option>
        </select>
        
        {extractType === 'attribute' && (
          <input
            type="text"
            placeholder="Attribute name..."
            className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
            value={attribute}
            onChange={(e) => setAttribute(e.target.value)}
            onBlur={() => updateNodeData({ attribute })}
          />
        )}
        
        <Handle type="source" position={Position.Bottom} className="node-handle" />
      </div>
      
      {showElementPicker && effectiveUrl && (
        <ElementPickerModal
          isOpen={showElementPicker}
          onClose={() => setShowElementPicker(false)}
          url={effectiveUrl!}
          onSelectElement={handleElementSelection}
          selectedSelector={selector}
        />
      )}
    </>
  );
};

export default memo(ExtractNode);
