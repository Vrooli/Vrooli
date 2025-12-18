import { memo, FC, useState, useEffect } from 'react';
import { Handle, Position, NodeProps, useReactFlow } from 'reactflow';
import { CheckCircle, Globe } from 'lucide-react';
import { useUpstreamUrl } from '@hooks/useUpstreamUrl';

const AssertNode: FC<NodeProps> = ({ data, selected, id }) => {
  const upstreamUrl = useUpstreamUrl(id);
  const { getNodes, setNodes } = useReactFlow();

  const assertModes = [
    { value: 'exists', label: 'Element Exists' },
    { value: 'not_exists', label: 'Element Does Not Exist' },
    { value: 'visible', label: 'Element Visible' },
    { value: 'text_equals', label: 'Text Equals' },
    { value: 'text_contains', label: 'Text Contains' },
    { value: 'attribute_equals', label: 'Attribute Equals' },
  ];

  const [label, setLabel] = useState(data.label || '');
  const [selector, setSelector] = useState(data.selector || '');
  const [expectedValue, setExpectedValue] = useState(data.expectedValue || '');
  const [attributeName, setAttributeName] = useState(data.attributeName || '');

  useEffect(() => {
    setLabel(data.label || '');
  }, [data.label]);

  useEffect(() => {
    setSelector(data.selector || '');
  }, [data.selector]);

  useEffect(() => {
    setExpectedValue(data.expectedValue || '');
  }, [data.expectedValue]);

  useEffect(() => {
    setAttributeName(data.attributeName || '');
  }, [data.attributeName]);

  const updateNodeData = (updates: Record<string, unknown>) => {
    const nodes = getNodes();
    const updatedNodes = nodes.map(node => {
      if (node.id === id) {
        return {
          ...node,
          data: {
            ...node.data,
            ...updates
          }
        };
      }
      return node;
    });
    setNodes(updatedNodes);
  };

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''} max-w-[280px]`}>
      <Handle type="target" position={Position.Top} className="node-handle" />

      <div className="flex items-center gap-2 mb-2">
        <CheckCircle size={16} className="text-orange-400" />
        <span className="font-semibold text-sm">Assert</span>
      </div>

      {upstreamUrl && (
        <div className="flex items-center gap-1 mb-2 p-1 bg-flow-bg/50 rounded text-xs border border-gray-700">
          <Globe size={12} className="text-blue-400 flex-shrink-0" />
          <span className="text-gray-400 truncate" title={upstreamUrl}>
            {upstreamUrl}
          </span>
        </div>
      )}

      <div className="space-y-2">
        <input
          type="text"
          placeholder="Label (optional)"
          className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
          value={label}
          onChange={(e) => setLabel(e.target.value)}
          onBlur={() => updateNodeData({ label })}
        />

        <select
          className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
          value={data.mode || 'exists'}
          onChange={(e) => updateNodeData({ mode: e.target.value })}
        >
          {assertModes.map(mode => (
            <option key={mode.value} value={mode.value}>
              {mode.label}
            </option>
          ))}
        </select>

        <input
          type="text"
          placeholder="CSS Selector..."
          className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
          value={selector}
          onChange={(e) => setSelector(e.target.value)}
          onBlur={() => updateNodeData({ selector })}
        />

        {(data.mode === 'text_equals' || data.mode === 'text_contains' || data.mode === 'attribute_equals') && (
          <input
            type="text"
            placeholder={data.mode === 'attribute_equals' ? 'Expected attribute value...' : 'Expected text...'}
            className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
            value={expectedValue}
            onChange={(e) => setExpectedValue(e.target.value)}
            onBlur={() => updateNodeData({ expectedValue })}
          />
        )}

        {data.mode === 'attribute_equals' && (
          <input
            type="text"
            placeholder="Attribute name..."
            className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
            value={attributeName}
            onChange={(e) => setAttributeName(e.target.value)}
            onBlur={() => updateNodeData({ attributeName })}
          />
        )}

        <div className="flex items-center gap-2 text-xs">
          <label className="flex items-center gap-1 cursor-pointer">
            <input
              type="checkbox"
              checked={data.continueOnFailure || false}
              onChange={(e) => updateNodeData({ continueOnFailure: e.target.checked })}
              className="rounded border-gray-700 bg-flow-bg text-flow-accent focus:ring-flow-accent focus:ring-offset-0"
            />
            <span className="text-gray-400">Continue on failure</span>
          </label>
        </div>

        {(data.mode === 'text_equals' || data.mode === 'text_contains') && (
          <div className="flex items-center gap-2 text-xs">
            <label className="flex items-center gap-1 cursor-pointer">
              <input
                type="checkbox"
                checked={data.caseSensitive || false}
                onChange={(e) => updateNodeData({ caseSensitive: e.target.checked })}
                className="rounded border-gray-700 bg-flow-bg text-flow-accent focus:ring-flow-accent focus:ring-offset-0"
              />
              <span className="text-gray-400">Case sensitive</span>
            </label>
          </div>
        )}
      </div>

      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(AssertNode);
