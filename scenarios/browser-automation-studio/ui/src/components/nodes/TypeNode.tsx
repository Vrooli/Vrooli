import { memo, FC, useState } from 'react';
import { Handle, Position, NodeProps, useReactFlow } from 'reactflow';
import { Keyboard, Globe, Target } from 'lucide-react';
import { useUpstreamUrl } from '../../hooks/useUpstreamUrl';
import ElementPickerModal from '../ElementPickerModal';

const TypeNode: FC<NodeProps> = ({ data, selected, id }) => {
  const upstreamUrl = useUpstreamUrl(id);
  const [showElementPicker, setShowElementPicker] = useState(false);
  const { getNodes, setNodes } = useReactFlow();

  const handleElementSelection = (selector: string, elementInfo: any) => {
    const nodes = getNodes();
    const updatedNodes = nodes.map(node => {
      if (node.id === id) {
        return {
          ...node,
          data: {
            ...node.data,
            selector: selector,
            elementInfo: elementInfo
          }
        };
      }
      return node;
    });
    setNodes(updatedNodes);
  };

  return (
    <>
      <div className={`workflow-node ${selected ? 'selected' : ''}`}>
        <Handle type="target" position={Position.Top} className="node-handle" />
        
        <div className="flex items-center gap-2 mb-2">
          <Keyboard size={16} className="text-yellow-400" />
          <span className="font-semibold text-sm">Type Text</span>
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
            defaultValue={data.selector || ''}
            onChange={(e) => {
              const newSelector = e.target.value;
              const nodes = getNodes();
              const updatedNodes = nodes.map(node => {
                if (node.id === id) {
                  return {
                    ...node,
                    data: {
                      ...node.data,
                      selector: newSelector
                    }
                  };
                }
                return node;
              });
              setNodes(updatedNodes);
            }}
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
        
        <textarea
          placeholder="Text to type..."
          className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none resize-none"
          rows={2}
          defaultValue={data.text || ''}
          onChange={(e) => {
            const newText = e.target.value;
            const nodes = getNodes();
            const updatedNodes = nodes.map(node => {
              if (node.id === id) {
                return {
                  ...node,
                  data: {
                    ...node.data,
                    text: newText
                  }
                };
              }
              return node;
            });
            setNodes(updatedNodes);
          }}
        />
        
        <Handle type="source" position={Position.Bottom} className="node-handle" />
      </div>
      
      {showElementPicker && upstreamUrl && (
        <ElementPickerModal
          isOpen={showElementPicker}
          onClose={() => setShowElementPicker(false)}
          url={upstreamUrl}
          onSelectElement={handleElementSelection}
          selectedSelector={data.selector}
        />
      )}
    </>
  );
};

export default memo(TypeNode);