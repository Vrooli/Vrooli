import { memo, FC, useState, useEffect } from 'react';
import type { NodeProps } from 'reactflow';
import { Variable, Settings2, Target } from 'lucide-react';
import { useNodeData } from '@hooks/useNodeData';
import { useUrlInheritance } from '@hooks/useUrlInheritance';
import { ElementPickerModal } from '../components';
import type { ElementInfo } from '@/types/elements';
import BaseNode from './BaseNode';

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

const SetVariableNode: FC<NodeProps> = ({ selected, id }) => {
  // URL inheritance hook handles URL state and handlers (for extract mode)
  const {
    urlDraft,
    setUrlDraft,
    effectiveUrl,
    hasCustomUrl,
    upstreamUrl,
    commitUrl,
  } = useUrlInheritance(id);

  // Node data hook for fields (set variable doesn't have a proto type, so all stored in node.data)
  const { getValue, updateData } = useNodeData(id);

  // Local state - all fields are UI-specific for this node
  const [name, setName] = useState<string>(getValue<string>('name') ?? '');
  const [sourceType, setSourceType] = useState<string>(getValue<string>('sourceType') ?? 'static');
  const [valueType, setValueType] = useState<string>(getValue<string>('valueType') ?? 'text');
  const [staticValue, setStaticValue] = useState<string>(getValue<string>('value') ?? '');
  const [expression, setExpression] = useState<string>(getValue<string>('expression') ?? '');
  const [selector, setSelector] = useState<string>(getValue<string>('selector') ?? '');
  const [extractType, setExtractType] = useState<string>(getValue<string>('extractType') ?? 'text');
  const [attribute, setAttribute] = useState<string>(getValue<string>('attribute') ?? '');
  const [storeAs, setStoreAs] = useState<string>(getValue<string>('storeAs') ?? '');
  const [timeoutMs, setTimeoutMs] = useState<number>(getValue<number>('timeoutMs') ?? 0);
  const [allMatches, setAllMatches] = useState<boolean>(coerceBoolean(getValue<boolean>('allMatches')));
  const [showElementPicker, setShowElementPicker] = useState(false);

  // Sync node.data fields
  useEffect(() => {
    setName(getValue<string>('name') ?? '');
  }, [getValue]);

  useEffect(() => {
    setSourceType(getValue<string>('sourceType') ?? 'static');
  }, [getValue]);

  useEffect(() => {
    setValueType(getValue<string>('valueType') ?? 'text');
  }, [getValue]);

  useEffect(() => {
    setStaticValue(getValue<string>('value') ?? '');
  }, [getValue]);

  useEffect(() => {
    setExpression(getValue<string>('expression') ?? '');
  }, [getValue]);

  useEffect(() => {
    setSelector(getValue<string>('selector') ?? '');
  }, [getValue]);

  useEffect(() => {
    setExtractType(getValue<string>('extractType') ?? 'text');
  }, [getValue]);

  useEffect(() => {
    setAttribute(getValue<string>('attribute') ?? '');
  }, [getValue]);

  useEffect(() => {
    setStoreAs(getValue<string>('storeAs') ?? '');
  }, [getValue]);

  useEffect(() => {
    setTimeoutMs(getValue<number>('timeoutMs') ?? 0);
  }, [getValue]);

  useEffect(() => {
    setAllMatches(coerceBoolean(getValue<boolean>('allMatches')));
  }, [getValue]);

  const handleElementSelection = (newSelector: string, elementInfo: ElementInfo) => {
    setSelector(newSelector);
    updateData({ selector: newSelector, elementInfo });
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
                updateData({ valueType: next });
              }}
            >
              {VALUE_TYPE_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
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
              onBlur={() => updateData({ value: staticValue })}
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
            onBlur={() => updateData({ expression })}
            placeholder="return document.querySelector('#title').textContent;"
          />
          <p className="text-[10px] text-gray-500 mt-1">
            Runs inside the page context. Return a value to store in the variable.
          </p>
        </div>
      );
    }

    // Extract mode
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
            onBlur={commitUrl}
          />
          {!hasCustomUrl && upstreamUrl && (
            <p className="text-[10px] text-gray-500 mt-1" title={upstreamUrl}>
              Inherits {upstreamUrl}
            </p>
          )}
        </div>
        <div className="flex items-center gap-2">
          <input
            type="text"
            className="flex-1 px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
            placeholder="CSS selector..."
            value={selector}
            onChange={(event) => setSelector(event.target.value)}
            onBlur={() => updateData({ selector })}
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
              updateData({ extractType: next });
            }}
          >
            {EXTRACT_TYPE_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
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
              onBlur={() => updateData({ attribute })}
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
                updateData({ allMatches: event.target.checked });
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
                updateData({ timeoutMs: next || undefined });
              }}
            />
          </div>
        </div>
      </div>
    );
  };

  return (
    <>
      <BaseNode selected={selected} icon={Variable} iconClassName="text-emerald-300" title="Set Variable">
        <div className="space-y-2 text-xs">
          <div>
            <label className="text-[11px] font-semibold text-gray-400">Variable Name</label>
            <input
              type="text"
              className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              value={name}
              onChange={(event) => setName(event.target.value)}
              onBlur={() => updateData({ name })}
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
              onBlur={() => updateData({ storeAs })}
              placeholder="friendly alias"
            />
          </div>
          <div>
            <label className="text-[11px] font-semibold text-gray-400 flex items-center gap-1">
              <Settings2 size={12} /> Source
            </label>
            <select
              className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
              value={sourceType}
              onChange={(event) => {
                const next = event.target.value;
                setSourceType(next);
                updateData({ sourceType: next });
              }}
            >
              {SOURCE_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>
          {renderSourceFields()}
        </div>
      </BaseNode>

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
