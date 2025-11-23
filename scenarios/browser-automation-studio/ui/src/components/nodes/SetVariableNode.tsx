import { memo, FC, useState, useEffect, useMemo, useCallback, useRef } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { Variable, Settings2, Target } from 'lucide-react';
import { useUpstreamUrl } from '../../hooks/useUpstreamUrl';
import ElementPickerModal from '../ElementPickerModal';
import type { ElementInfo } from '../../types/elements';

const SOURCE_OPTIONS = [
  { value: 'static', label: 'Static Value' },
  { value: 'expression', label: 'JavaScript Expression' },
  { value: 'extract', label: 'Extract from Selector' },
];

const VALUE_TYPE_OPTIONS = [
  { value: 'text', label: 'Text' },
  { value: 'number', label: 'Number' },
  { value: 'boolean', label: 'Boolean' },
  { value: 'json', label: 'JSON' },
];

const EXTRACT_TYPE_OPTIONS = [
  { value: 'text', label: 'Text Content' },
  { value: 'attribute', label: 'Attribute' },
  { value: 'value', label: 'Input Value' },
  { value: 'html', label: 'Inner HTML' },
];

function coerceBoolean(value: unknown): boolean {
  if (typeof value === 'boolean') {
    return value;
  }
  if (typeof value === 'string') {
    return value.toLowerCase() === 'true';
  }
  return Boolean(value);
}

const SetVariableNode: FC<NodeProps> = ({ data, selected, id }) => {
  const nodeRef = useRef<HTMLDivElement>(null);
  const nodeData = (data ?? {}) as Record<string, unknown>;
  const upstreamUrl = useUpstreamUrl(id);
  const { getNodes, setNodes } = useReactFlow();

  const [name, setName] = useState<string>(String(nodeData.name ?? ''));
  const [sourceType, setSourceType] = useState<string>(String(nodeData.sourceType ?? 'static'));
  const [valueType, setValueType] = useState<string>(String(nodeData.valueType ?? 'text'));
  const [staticValue, setStaticValue] = useState<string>(String(nodeData.value ?? ''));
  const [expression, setExpression] = useState<string>(String(nodeData.expression ?? ''));
  const [selector, setSelector] = useState<string>(String(nodeData.selector ?? ''));
  const [extractType, setExtractType] = useState<string>(String(nodeData.extractType ?? 'text'));
  const [attribute, setAttribute] = useState<string>(String(nodeData.attribute ?? ''));
  const [storeAs, setStoreAs] = useState<string>(String(nodeData.storeAs ?? ''));
  const [timeoutMs, setTimeoutMs] = useState<number>(Number(nodeData.timeoutMs ?? 0));
  const [allMatches, setAllMatches] = useState<boolean>(coerceBoolean(nodeData.allMatches));
  const [showElementPicker, setShowElementPicker] = useState(false);
  const [urlDraft, setUrlDraft] = useState<string>(String(nodeData.url ?? ''));

  const storedUrl = typeof nodeData.url === 'string' ? nodeData.url : '';
  useEffect(() => {
    setUrlDraft(storedUrl);
  }, [storedUrl]);

  useEffect(() => {
    setName(String(nodeData.name ?? ''));
  }, [nodeData.name]);
  useEffect(() => {
    setSourceType(String(nodeData.sourceType ?? 'static'));
  }, [nodeData.sourceType]);
  useEffect(() => {
    setValueType(String(nodeData.valueType ?? 'text'));
  }, [nodeData.valueType]);
  useEffect(() => {
    setStaticValue(String(nodeData.value ?? ''));
  }, [nodeData.value]);
  useEffect(() => {
    setExpression(String(nodeData.expression ?? ''));
  }, [nodeData.expression]);
  useEffect(() => {
    setSelector(String(nodeData.selector ?? ''));
  }, [nodeData.selector]);
  useEffect(() => {
    setExtractType(String(nodeData.extractType ?? 'text'));
  }, [nodeData.extractType]);
  useEffect(() => {
    setAttribute(String(nodeData.attribute ?? ''));
  }, [nodeData.attribute]);
  useEffect(() => {
    setStoreAs(String(nodeData.storeAs ?? ''));
  }, [nodeData.storeAs]);
  useEffect(() => {
    setTimeoutMs(Number(nodeData.timeoutMs ?? 0));
  }, [nodeData.timeoutMs]);
  useEffect(() => {
    setAllMatches(coerceBoolean(nodeData.allMatches));
  }, [nodeData.allMatches]);

  const updateNodeData = useCallback((updates: Record<string, unknown>) => {
    const nodes = getNodes();
    setNodes(nodes.map((node) => {
      if (node.id !== id) {
        return node;
      }
      const nextData = { ...(node.data ?? {}) } as Record<string, unknown>;
      Object.entries(updates).forEach(([key, value]) => {
        if (value === undefined || value === null || value === '') {
          delete nextData[key];
        } else {
          nextData[key] = value;
        }
      });
      return { ...node, data: nextData };
    }));
  }, [getNodes, setNodes, id]);

  const effectiveUrl = useMemo(() => {
    const explicitUrl = typeof nodeData.url === 'string' ? nodeData.url.trim() : '';
    if (explicitUrl) {
      return explicitUrl;
    }
    return upstreamUrl ?? null;
  }, [nodeData.url, upstreamUrl]);

  const handleUrlCommit = useCallback(() => {
    const trimmed = urlDraft.trim();
    updateNodeData({ url: trimmed || undefined });
  }, [updateNodeData, urlDraft]);

  const handleElementSelection = (newSelector: string, elementInfo: ElementInfo) => {
    setSelector(newSelector);
    updateNodeData({ selector: newSelector, elementInfo });
  };

  const renderSourceFields = () => {
    if (sourceType === 'static') {
      return (
        <div className="space-y-2">
          <div>
            <label className="text-[11px] font-semibold text-gray-400">Value Type</label>
            <select
              className="w-full px-2 py-1 mt-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
              value={valueType}
              onChange={(event) => {
                const next = event.target.value;
                setValueType(next);
                updateNodeData({ valueType: next });
              }}
            >
              {VALUE_TYPE_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>{option.label}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-[11px] font-semibold text-gray-400">Value</label>
            <textarea
              className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
              rows={3}
              value={staticValue}
              onChange={(event) => setStaticValue(event.target.value)}
              onBlur={() => updateNodeData({ value: staticValue })}
              placeholder={valueType === 'json' ? '{"key": "value"}' : 'Value to store'}
            />
          </div>
        </div>
      );
    }

    if (sourceType === 'expression') {
      return (
        <div>
          <label className="text-[11px] font-semibold text-gray-400">Expression</label>
          <textarea
            className="w-full px-2 py-1 mt-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
            rows={4}
            value={expression}
            onChange={(event) => setExpression(event.target.value)}
            onBlur={() => updateNodeData({ expression })}
            placeholder="return document.querySelector('#title').textContent;"
          />
          <p className="text-[10px] text-gray-500 mt-1">Runs inside the page context. Return a value to store in the variable.</p>
        </div>
      );
    }

    return (
      <div className="space-y-2">
        <div>
          <label className="text-[11px] font-semibold text-gray-400">Page URL</label>
          <input
            type="text"
            placeholder={upstreamUrl ?? 'https://example.com'}
            className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
            value={urlDraft}
            onChange={(event) => setUrlDraft(event.target.value)}
            onBlur={handleUrlCommit}
          />
          {!storedUrl && upstreamUrl && (
            <p className="text-[10px] text-gray-500 mt-1" title={upstreamUrl}>Inherits {upstreamUrl}</p>
          )}
        </div>
        <div className="flex items-center gap-2">
          <input
            type="text"
            className="flex-1 px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
            placeholder="CSS selector..."
            value={selector}
            onChange={(event) => setSelector(event.target.value)}
            onBlur={() => updateNodeData({ selector })}
          />
          <button
            type="button"
            className={`p-1.5 rounded border border-gray-700 ${effectiveUrl ? 'hover:bg-gray-700' : 'opacity-50 cursor-not-allowed'}`}
            onClick={() => effectiveUrl && setShowElementPicker(true)}
            disabled={!effectiveUrl}
            title={effectiveUrl ? `Pick element from ${effectiveUrl}` : 'Set a page URL to enable element picker'}
          >
            <Target size={14} className="text-gray-400" />
          </button>
        </div>
        <div>
          <label className="text-[11px] font-semibold text-gray-400">Extract Type</label>
          <select
            className="w-full px-2 py-1 mt-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
            value={extractType}
            onChange={(event) => {
              const next = event.target.value;
              setExtractType(next);
              updateNodeData({ extractType: next });
            }}
          >
            {EXTRACT_TYPE_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>{option.label}</option>
            ))}
          </select>
        </div>
        {extractType === 'attribute' && (
          <div>
            <label className="text-[11px] font-semibold text-gray-400">Attribute Name</label>
            <input
              type="text"
              className="w-full px-2 py-1 mt-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
              value={attribute}
              onChange={(event) => setAttribute(event.target.value)}
              onBlur={() => updateNodeData({ attribute })}
              placeholder="data-testid"
            />
          </div>
        )}
        <div className="flex items-center justify-between text-xs text-gray-400">
          <label className="flex items-center gap-1">
            <input
              type="checkbox"
              checked={allMatches}
              onChange={(event) => {
                setAllMatches(event.target.checked);
                updateNodeData({ allMatches: event.target.checked });
              }}
            />
            Capture all matches
          </label>
          <div className="flex items-center gap-1">
            <span>Timeout</span>
            <input
              type="number"
              min={0}
              className="w-20 px-2 py-0.5 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              value={timeoutMs}
              onChange={(event) => {
                const next = Math.max(0, Number(event.target.value));
                setTimeoutMs(next);
                updateNodeData({ timeoutMs: next || undefined });
              }}
            />
          </div>
        </div>
      </div>
    );
  };

  // Add data-type attribute to React Flow wrapper div for test automation
  useEffect(() => {
    if (nodeRef.current) {
      const reactFlowNode = nodeRef.current.closest('.react-flow__node');
      if (reactFlowNode) {
        reactFlowNode.setAttribute('data-type', 'setVariable');
      }
    }
  }, []);

  return (
    <>
      <div ref={nodeRef} className={`workflow-node ${selected ? 'selected' : ''}`}>
        <Handle type="target" position={Position.Top} className="node-handle" />

        <div className="flex items-center gap-2 mb-2">
          <Variable size={16} className="text-emerald-300" />
          <span className="font-semibold text-sm">Set Variable</span>
        </div>

        <div className="space-y-2 text-xs">
          <div>
            <label className="text-[11px] font-semibold text-gray-400">Variable Name</label>
            <input
              type="text"
              className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              value={name}
              onChange={(event) => setName(event.target.value)}
              onBlur={() => updateNodeData({ name })}
              placeholder="variableName"
            />
          </div>
          <div>
            <label className="text-[11px] font-semibold text-gray-400">Store As (optional alias)</label>
            <input
              type="text"
              className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              value={storeAs}
              onChange={(event) => setStoreAs(event.target.value)}
              onBlur={() => updateNodeData({ storeAs })}
              placeholder="friendly alias"
            />
          </div>
          <div>
            <label className="text-[11px] font-semibold text-gray-400 flex items-center gap-1"><Settings2 size={12} /> Source</label>
            <select
              className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              value={sourceType}
              onChange={(event) => {
                const next = event.target.value;
                setSourceType(next);
                updateNodeData({ sourceType: next });
              }}
            >
              {SOURCE_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>{option.label}</option>
              ))}
            </select>
          </div>
          {renderSourceFields()}
        </div>

        <Handle type="source" position={Position.Bottom} className="node-handle" />
      </div>

      {showElementPicker && effectiveUrl && (
        <ElementPickerModal
          isOpen={showElementPicker}
          onClose={() => setShowElementPicker(false)}
          url={effectiveUrl}
          onSelectElement={handleElementSelection}
          selectedSelector={selector}
        />
      )}
    </>
  );
};

export default memo(SetVariableNode);
