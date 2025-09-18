import { memo, FC } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';
import { MousePointer, Globe } from 'lucide-react';
import { useUpstreamUrl } from '../../hooks/useUpstreamUrl';

const ClickNode: FC<NodeProps> = ({ data, selected, id }) => {
  const upstreamUrl = useUpstreamUrl(id);

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />
      
      <div className="flex items-center gap-2 mb-2">
        <MousePointer size={16} className="text-green-400" />
        <span className="font-semibold text-sm">Click</span>
      </div>
      
      {upstreamUrl && (
        <div className="flex items-center gap-1 mb-2 p-1 bg-flow-bg/50 rounded text-xs border border-gray-700">
          <Globe size={12} className="text-blue-400 flex-shrink-0" />
          <span className="text-gray-400 truncate" title={upstreamUrl}>
            {upstreamUrl}
          </span>
        </div>
      )}
      
      <input
        type="text"
        placeholder="CSS Selector..."
        className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
        defaultValue={data.selector || ''}
        onChange={(e) => {
          data.selector = e.target.value;
        }}
      />
      
      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(ClickNode);