import { memo, FC } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';
import { Camera, Globe } from 'lucide-react';
import { useUpstreamUrl } from '../../hooks/useUpstreamUrl';

const ScreenshotNode: FC<NodeProps> = ({ data, selected, id }) => {
  const upstreamUrl = useUpstreamUrl(id);

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />
      
      <div className="flex items-center gap-2 mb-2">
        <Camera size={16} className="text-purple-400" />
        <span className="font-semibold text-sm">Screenshot</span>
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
        placeholder="Screenshot name..."
        className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none mb-2"
        defaultValue={data.name || ''}
        onChange={(e) => {
          data.name = e.target.value;
        }}
      />
      
      <label className="flex items-center gap-2 text-xs">
        <input
          type="checkbox"
          className="rounded"
          defaultChecked={data.fullPage || false}
          onChange={(e) => {
            data.fullPage = e.target.checked;
          }}
        />
        <span>Full page</span>
      </label>
      
      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(ScreenshotNode);