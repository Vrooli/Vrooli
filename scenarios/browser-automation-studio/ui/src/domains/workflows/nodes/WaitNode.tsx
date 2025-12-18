import { memo, FC, useState, useEffect } from 'react';
import { Handle, Position, NodeProps, useReactFlow } from 'reactflow';
import { Clock, Globe } from 'lucide-react';
import { useUpstreamUrl } from '@hooks/useUpstreamUrl';

const WaitNode: FC<NodeProps> = ({ data, selected, id }) => {
  const upstreamUrl = useUpstreamUrl(id);
  const { getNodes, setNodes } = useReactFlow();

  const [waitType, setWaitType] = useState(data.waitType || 'time');
  const [duration, setDuration] = useState(data.duration || 1000);
  const [selector, setSelector] = useState(data.selector || '');

  useEffect(() => {
    setWaitType(data.waitType || 'time');
  }, [data.waitType]);

  useEffect(() => {
    setDuration(data.duration || 1000);
  }, [data.duration]);

  useEffect(() => {
    setSelector(data.selector || '');
  }, [data.selector]);

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
        value={waitType}
        onChange={(e) => {
          const newType = e.target.value;
          setWaitType(newType);
          updateNodeData({ waitType: newType });
        }}
      >
        <option value="time">Wait for time</option>
        <option value="element">Wait for element</option>
        <option value="navigation">Wait for navigation</option>
      </select>
      
      {waitType === 'time' ? (
        <input
          type="number"
          placeholder="Milliseconds..."
          className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
          value={duration}
          onChange={(e) => setDuration(parseInt(e.target.value) || 0)}
          onBlur={() => updateNodeData({ duration })}
        />
      ) : (
        <input
          type="text"
          placeholder="CSS Selector..."
          className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
          value={selector}
          onChange={(e) => setSelector(e.target.value)}
          onBlur={() => updateNodeData({ selector })}
        />
      )}
      
      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(WaitNode);