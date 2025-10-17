import { memo, FC, useState } from 'react';
import { Handle, Position, NodeProps, useReactFlow } from 'reactflow';
import { Play, ChevronDown, Settings } from 'lucide-react';

interface WorkflowCallData {
  workflowId?: string;
  workflowName?: string;
  parameters?: Record<string, any>;
  waitForCompletion?: boolean;
  outputMapping?: Record<string, string>;
}

const WorkflowCallNode: FC<NodeProps> = ({ data, selected, id }) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const { getNodes, setNodes } = useReactFlow();
  
  const updateNodeData = (updates: Partial<WorkflowCallData>) => {
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
    <div className={`workflow-node ${selected ? 'selected' : ''} w-80`}>
      <Handle 
        type="target" 
        position={Position.Top} 
        className="node-handle" 
      />
      
      <div className="flex items-center gap-2 mb-3">
        <Play size={16} className="text-violet-400" />
        <span className="font-semibold text-sm">Call Workflow</span>
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="ml-auto p-1 hover:bg-gray-700 rounded"
          title={isExpanded ? "Show less" : "Show more"}
        >
          <ChevronDown 
            size={14} 
            className={`transform transition-transform text-gray-400 ${isExpanded ? 'rotate-180' : ''}`}
          />
        </button>
      </div>
      <div className="space-y-3">
        <div className="space-y-1">
          <label className="text-xs text-gray-400">Target Workflow</label>
          <select 
            value={data.workflowId || ''} 
            className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
            onChange={(e) => {
              const selectedId = e.target.value;
              const selectedName = e.target.options[e.target.selectedIndex]?.text;
              updateNodeData({ 
                workflowId: selectedId,
                workflowName: selectedName !== 'Select workflow...' ? selectedName : undefined
              });
            }}
          >
            <option value="">Select workflow...</option>
            <option value="workflow-1">Example Workflow 1</option>
            <option value="workflow-2">Example Workflow 2</option>
            <option value="login-flow">Login Flow</option>
            <option value="data-extraction">Data Extraction</option>
          </select>
        </div>

        <div className="flex items-center gap-2">
          <input
            type="checkbox"
            id={`wait-${id}`}
            checked={data.waitForCompletion ?? true}
            onChange={(e) => {
              updateNodeData({ waitForCompletion: e.target.checked });
            }}
            className="rounded border-gray-600 bg-flow-bg text-violet-500 focus:ring-violet-500 focus:ring-offset-0"
          />
          <label htmlFor={`wait-${id}`} className="text-xs text-gray-300">Wait for completion</label>
        </div>

        {isExpanded && (
          <div className="space-y-3 pt-3 border-t border-gray-700">
            <div className="space-y-1">
              <label className="flex items-center gap-1 text-xs text-gray-400">
                <Settings size={12} />
                Parameters (JSON)
              </label>
              <textarea
                value={data.parameters ? JSON.stringify(data.parameters, null, 2) : '{}'}
                onChange={(e) => {
                  try {
                    const parsed = JSON.parse(e.target.value);
                    updateNodeData({ parameters: parsed });
                  } catch (err) {
                    // Invalid JSON - don't update
                  }
                }}
                className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none font-mono h-16 resize-none"
                placeholder='{"param1": "value1"}'
              />
            </div>

            <div className="space-y-1">
              <label className="flex items-center gap-1 text-xs text-gray-400">
                <Settings size={12} />
                Output Mapping (JSON)
              </label>
              <textarea
                value={data.outputMapping ? JSON.stringify(data.outputMapping, null, 2) : '{}'}
                onChange={(e) => {
                  try {
                    const parsed = JSON.parse(e.target.value);
                    updateNodeData({ outputMapping: parsed });
                  } catch (err) {
                    // Invalid JSON - don't update
                  }
                }}
                className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none font-mono h-12 resize-none"
                placeholder='{"workflowOutput": "myVariable"}'
              />
            </div>
          </div>
        )}
      </div>
      
      <Handle 
        type="source" 
        position={Position.Bottom} 
        className="node-handle" 
      />
    </div>
  );
};

export default memo(WorkflowCallNode);