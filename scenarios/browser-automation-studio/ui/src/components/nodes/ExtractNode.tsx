import { memo, FC, useState, useEffect } from 'react';
import { Handle, Position, NodeProps, useReactFlow } from 'reactflow';
import { Database, Globe, Target } from 'lucide-react';
import { useUpstreamUrl } from '../../hooks/useUpstreamUrl';
import ElementPickerModal from '../ElementPickerModal';
import type { ElementInfo } from '../../types/elements';

const ExtractNode: FC<NodeProps> = ({ data, selected, id }) => {
  const upstreamUrl = useUpstreamUrl(id);
  const [showElementPicker, setShowElementPicker] = useState(false);
  const { getNodes, setNodes } = useReactFlow();

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

  const updateNodeData = (updates: Record<string, any>) => {
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
        
        {upstreamUrl && (
          <div className="flex items-center gap-1 mb-2 p-1 bg-flow-bg/50 rounded text-xs border border-gray-700">
            <Globe size={12} className="text-blue-400 flex-shrink-0" />
            <span className="text-gray-400 truncate" title={upstreamUrl}>
              {upstreamUrl}
            </span>
          </div>
        )}
        
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
            title={!upstreamUrl ? "Connect to a Navigate node first to pick elements from a webpage" : ""}
          >
            <button
              onClick={() => setShowElementPicker(true)}
              className={`p-1.5 bg-flow-bg rounded border border-gray-700 transition-colors ${
                upstreamUrl ? 'hover:bg-gray-700 cursor-pointer' : 'opacity-50 cursor-not-allowed'
              }`}
              title={upstreamUrl ? `Pick element from ${upstreamUrl}` : undefined}
              disabled={!upstreamUrl}
            >
              <Target size={14} className={upstreamUrl ? "text-gray-400" : "text-gray-600"} />
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
      
      {showElementPicker && upstreamUrl && (
        <ElementPickerModal
          isOpen={showElementPicker}
          onClose={() => setShowElementPicker(false)}
          url={upstreamUrl}
          onSelectElement={handleElementSelection}
          selectedSelector={selector}
        />
      )}
    </>
  );
};

export default memo(ExtractNode);