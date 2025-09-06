import { memo, FC } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';
import { Globe } from 'lucide-react';

const NavigateNode: FC<NodeProps> = ({ data, selected }) => {
  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />
      
      <div className="flex items-center gap-2 mb-2">
        <Globe size={16} className="text-blue-400" />
        <span className="font-semibold text-sm">Navigate</span>
      </div>
      
      <input
        type="text"
        placeholder="Enter URL..."
        className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
        defaultValue={data.url || ''}
        onChange={(e) => {
          data.url = e.target.value;
        }}
      />
      
      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(NavigateNode);