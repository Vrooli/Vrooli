import { memo, FC } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';
import { Database } from 'lucide-react';

const ExtractNode: FC<NodeProps> = ({ data, selected }) => {
  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />
      
      <div className="flex items-center gap-2 mb-2">
        <Database size={16} className="text-pink-400" />
        <span className="font-semibold text-sm">Extract Data</span>
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
      
      <select
        className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none mb-2"
        defaultValue={data.extractType || 'text'}
        onChange={(e) => {
          data.extractType = e.target.value;
        }}
      >
        <option value="text">Text Content</option>
        <option value="attribute">Attribute</option>
        <option value="html">Inner HTML</option>
        <option value="value">Input Value</option>
      </select>
      
      {data.extractType === 'attribute' && (
        <input
          type="text"
          placeholder="Attribute name..."
          className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
          defaultValue={data.attribute || ''}
          onChange={(e) => {
            data.attribute = e.target.value;
          }}
        />
      )}
      
      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(ExtractNode);