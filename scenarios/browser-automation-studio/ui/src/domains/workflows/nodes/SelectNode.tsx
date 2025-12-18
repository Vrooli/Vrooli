import { memo, FC, useState, useEffect, useCallback, ChangeEvent } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { ListFilter } from 'lucide-react';
import ResiliencePanel from './ResiliencePanel';
import type { ResilienceSettings } from '@/types/workflow';

const SELECT_BY_OPTIONS = [
  { value: 'value', label: 'Option value' },
  { value: 'text', label: 'Displayed text' },
  { value: 'index', label: 'Option index' },
];

const parseValuesInput = (raw: string): string[] => {
  return raw
    .split(/\r?\n|,/)
    .map((entry) => entry.trim())
    .filter((entry) => entry.length > 0);
};

const formatValues = (values?: unknown): string => {
  if (!Array.isArray(values)) {
    return '';
  }
  return values.join('\n');
};

const SelectNode: FC<NodeProps> = ({ data, selected, id }) => {
  const nodeData = (data ?? {}) as Record<string, unknown>;
  const { getNodes, setNodes } = useReactFlow();

  const [selector, setSelector] = useState<string>(() => typeof nodeData.selector === 'string' ? nodeData.selector : '');
  const [selectBy, setSelectBy] = useState<string>(() => typeof nodeData.selectBy === 'string' ? nodeData.selectBy : 'value');
  const [value, setValue] = useState<string>(() => typeof nodeData.value === 'string' ? nodeData.value : '');
  const [text, setText] = useState<string>(() => typeof nodeData.text === 'string' ? nodeData.text : '');
  const [index, setIndex] = useState<number>(() => {
    const raw = Number((nodeData.index ?? 0));
    return Number.isFinite(raw) && raw >= 0 ? raw : 0;
  });
  const [multiple, setMultiple] = useState<boolean>(() => Boolean(nodeData.multiple));
  const [valuesDraft, setValuesDraft] = useState<string>(() => formatValues(nodeData.values));
  const [timeoutMs, setTimeoutMs] = useState<number>(() => Number(nodeData.timeoutMs ?? 5000) || 5000);
  const [waitForMs, setWaitForMs] = useState<number>(() => Number(nodeData.waitForMs ?? 0) || 0);

  useEffect(() => {
    setSelector(typeof nodeData.selector === 'string' ? nodeData.selector : '');
  }, [nodeData.selector]);

  useEffect(() => {
    setSelectBy(typeof nodeData.selectBy === 'string' ? nodeData.selectBy : 'value');
  }, [nodeData.selectBy]);

  useEffect(() => {
    setValue(typeof nodeData.value === 'string' ? nodeData.value : '');
  }, [nodeData.value]);

  useEffect(() => {
    setText(typeof nodeData.text === 'string' ? nodeData.text : '');
  }, [nodeData.text]);

  useEffect(() => {
    const raw = Number(nodeData.index ?? 0);
    setIndex(Number.isFinite(raw) && raw >= 0 ? raw : 0);
  }, [nodeData.index]);

  useEffect(() => {
    setMultiple(Boolean(nodeData.multiple));
  }, [nodeData.multiple]);

  useEffect(() => {
    setValuesDraft(formatValues(nodeData.values));
  }, [nodeData.values]);

  useEffect(() => {
    setTimeoutMs(Number(nodeData.timeoutMs ?? 5000) || 5000);
  }, [nodeData.timeoutMs]);

  useEffect(() => {
    setWaitForMs(Number(nodeData.waitForMs ?? 0) || 0);
  }, [nodeData.waitForMs]);

  const resilienceConfig = nodeData.resilience as ResilienceSettings | undefined;

  const updateNodeData = useCallback((updates: Record<string, unknown>) => {
    const nodes = getNodes();
    setNodes(nodes.map((node) => {
      if (node.id !== id) {
        return node;
      }
      const nextData = { ...(node.data ?? {}) } as Record<string, unknown>;
      for (const [key, val] of Object.entries(updates)) {
        if (val === undefined || val === null || (typeof val === 'string' && val.trim() === '')) {
          delete nextData[key];
        } else {
          nextData[key] = val;
        }
      }
      return { ...node, data: nextData };
    }));
  }, [getNodes, id, setNodes]);

  useEffect(() => {
    if (multiple && selectBy === 'index') {
      setSelectBy('value');
      updateNodeData({ selectBy: 'value' });
    }
  }, [multiple, selectBy, updateNodeData]);

  const handleSelectorBlur = useCallback(() => {
    updateNodeData({ selector: selector.trim() });
  }, [selector, updateNodeData]);

  const handleValueBlur = useCallback(() => {
    updateNodeData({ value: value.trim() });
  }, [updateNodeData, value]);

  const handleTextBlur = useCallback(() => {
    updateNodeData({ text: text.trim() });
  }, [text, updateNodeData]);

  const handleIndexBlur = useCallback(() => {
    const normalized = Math.max(0, Math.round(index) || 0);
    setIndex(normalized);
    updateNodeData({ index: normalized });
  }, [index, updateNodeData]);

  const handleValuesBlur = useCallback(() => {
    const parsed = parseValuesInput(valuesDraft);
    setValuesDraft(parsed.join('\n'));
    updateNodeData({ values: parsed });
  }, [updateNodeData, valuesDraft]);

  const handleTimeoutBlur = useCallback(() => {
    const normalized = Math.max(100, Math.round(timeoutMs) || 100);
    setTimeoutMs(normalized);
    updateNodeData({ timeoutMs: normalized });
  }, [timeoutMs, updateNodeData]);

  const handleWaitBlur = useCallback(() => {
    const normalized = Math.max(0, Math.round(waitForMs) || 0);
    setWaitForMs(normalized);
    updateNodeData({ waitForMs: normalized });
  }, [updateNodeData, waitForMs]);

  const handleSelectByChange = useCallback((event: ChangeEvent<HTMLSelectElement>) => {
    const value = event.target.value;
    setSelectBy(value);
    updateNodeData({ selectBy: value });
  }, [updateNodeData]);

  const toggleMultiple = useCallback(() => {
    const next = !multiple;
    setMultiple(next);
    updateNodeData({ multiple: next });
  }, [multiple, updateNodeData]);

  const renderSelectionField = () => {
    if (multiple) {
      return (
        <div>
          <label className="text-gray-400 text-[11px] uppercase tracking-wide block mb-1">Values (one per line)</label>
          <textarea
            rows={3}
            value={valuesDraft}
            onChange={(event) => setValuesDraft(event.target.value)}
            onBlur={handleValuesBlur}
            placeholder={selectBy === 'text' ? 'Primary\nSecondary' : 'value-a\nvalue-b'}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 text-xs focus:border-flow-accent focus:outline-none resize-none"
          />
          <p className="text-[10px] text-gray-500 mt-1">All matching options will be selected.</p>
        </div>
      );
    }

    if (selectBy === 'index') {
      return (
        <div>
          <label className="text-gray-400 text-[11px] uppercase tracking-wide block mb-1">Option index</label>
          <input
            type="number"
            min={0}
            value={index}
            onChange={(event) => setIndex(Number(event.target.value))}
            onBlur={handleIndexBlur}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 text-xs focus:border-flow-accent focus:outline-none"
          />
          <p className="text-[10px] text-gray-500 mt-1">Uses zero-based indexing (0 = first option).</p>
        </div>
      );
    }

    if (selectBy === 'text') {
      return (
        <div>
          <label className="text-gray-400 text-[11px] uppercase tracking-wide block mb-1">Text to match</label>
          <input
            type="text"
            value={text}
            onChange={(event) => setText(event.target.value)}
            onBlur={handleTextBlur}
            placeholder="Contains..."
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 text-xs focus:border-flow-accent focus:outline-none"
          />
          <p className="text-[10px] text-gray-500 mt-1">Matches options whose text includes this value.</p>
        </div>
      );
    }

    return (
      <div>
        <label className="text-gray-400 text-[11px] uppercase tracking-wide block mb-1">Option value</label>
        <input
          type="text"
          value={value}
          onChange={(event) => setValue(event.target.value)}
          onBlur={handleValueBlur}
          placeholder="Option value attribute"
          className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 text-xs focus:border-flow-accent focus:outline-none"
        />
      </div>
    );
  };

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />

      <div className="flex items-center gap-2 mb-3">
        <ListFilter size={16} className="text-teal-300" />
        <span className="font-semibold text-sm">Select Option</span>
      </div>

      <div className="space-y-3 text-xs">
        <div>
          <label className="text-gray-400 text-[11px] uppercase tracking-wide block mb-1">Target selector</label>
          <input
            type="text"
            value={selector}
            onChange={(event) => setSelector(event.target.value)}
            onBlur={handleSelectorBlur}
          placeholder={'select[name="country"]'}
            className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 text-[11px] uppercase tracking-wide block mb-1">Select by</label>
            <select
              value={selectBy}
              onChange={handleSelectByChange}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            >
              {SELECT_BY_OPTIONS.map((option) => (
                <option key={option.value} value={option.value} disabled={option.value === 'index' && multiple}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-gray-400 text-[11px] uppercase tracking-wide block mb-1">Allow multiple</label>
            <button
              type="button"
              onClick={toggleMultiple}
              className={`w-full px-2 py-1 rounded border text-xs transition-colors ${multiple ? 'border-teal-400 text-teal-200 bg-teal-400/10' : 'border-gray-700 text-gray-300 hover:border-gray-500'}`}
            >
              {multiple ? 'Enabled' : 'Disabled'}
            </button>
          </div>
        </div>

        {renderSelectionField()}

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="text-gray-400 text-[11px] uppercase tracking-wide block mb-1">Timeout (ms)</label>
            <input
              type="number"
              min={100}
              value={timeoutMs}
              onChange={(event) => setTimeoutMs(Number(event.target.value))}
              onBlur={handleTimeoutBlur}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
          <div>
            <label className="text-gray-400 text-[11px] uppercase tracking-wide block mb-1">Post-change wait (ms)</label>
            <input
              type="number"
              min={0}
              value={waitForMs}
              onChange={(event) => setWaitForMs(Number(event.target.value))}
              onBlur={handleWaitBlur}
              className="w-full px-2 py-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            />
          </div>
        </div>

        <p className="text-[11px] text-gray-500">
          Dispatches change/input events after updating the element so downstream nodes can rely on the new value.
        </p>
      </div>

      <ResiliencePanel
        value={resilienceConfig}
        onChange={(next) => updateNodeData({ resilience: next ?? null })}
      />

      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(SelectNode);
