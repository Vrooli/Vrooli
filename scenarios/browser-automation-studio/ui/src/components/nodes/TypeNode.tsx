import { memo, FC } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';
import { Keyboard } from 'lucide-react';

const TypeNode: FC<NodeProps> = ({ data, selected }) => {
  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />
      
      <div className="flex items-center gap-2 mb-2">
        <Keyboard size={16} className="text-yellow-400" />
        <span className="font-semibold text-sm">Type Text</span>
      </div>
      
      <input
        type="text"
        placeholder="CSS Selector..."
        className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none mb-2"
        defaultValue={data.selector || ''}
        onChange={(e) => {
          data.selector = e.target.value;
        }}
      />
      
      <textarea
        placeholder="Text to type..."
        className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none resize-none"
        rows={2}
        defaultValue={data.text || ''}
        onChange={(e) => {
          data.text = e.target.value;
        }}
      />
      
      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(TypeNode);