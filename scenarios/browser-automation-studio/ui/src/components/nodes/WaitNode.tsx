import { memo, FC } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';
import { Clock, Globe } from 'lucide-react';
import { useUpstreamUrl } from '../../hooks/useUpstreamUrl';

const WaitNode: FC<NodeProps> = ({ data, selected, id }) => {
  const upstreamUrl = useUpstreamUrl(id);

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />
      
      <div className="flex items-center gap-2 mb-2">
        <Clock size={16} className="text-gray-400" />
        <span className="font-semibold text-sm">Wait</span>
      </div>
      
      {upstreamUrl && (
        <div className="flex items-center gap-1 mb-2 p-1 bg-flow-bg/50 rounded text-xs border border-gray-700">
          <Globe size={12} className="text-blue-400 flex-shrink-0" />
          <span className="text-gray-400 truncate" title={upstreamUrl}>
            {upstreamUrl}
          </span>
        </div>
      )}
      
      <select
        className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none mb-2"
        defaultValue={data.waitType || 'time'}
        onChange={(e) => {
          data.waitType = e.target.value;
        }}
      >
        <option value="time">Wait for time</option>
        <option value="element">Wait for element</option>
        <option value="navigation">Wait for navigation</option>
      </select>
      
      {data.waitType === 'time' ? (
        <input
          type="number"
          placeholder="Milliseconds..."
          className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
          defaultValue={data.duration || 1000}
          onChange={(e) => {
            data.duration = parseInt(e.target.value);
          }}
        />
      ) : (
        <input
          type="text"
          placeholder="CSS Selector..."
          className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
          defaultValue={data.selector || ''}
          onChange={(e) => {
            data.selector = e.target.value;
          }}
        />
      )}
      
      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(WaitNode);