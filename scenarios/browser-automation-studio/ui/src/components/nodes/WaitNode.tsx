import { memo, FC } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';
import { Clock } from 'lucide-react';

const WaitNode: FC<NodeProps> = ({ data, selected }) => {
  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />
      
      <div className="flex items-center gap-2 mb-2">
        <Clock size={16} className="text-gray-400" />
        <span className="font-semibold text-sm">Wait</span>
      </div>
      
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