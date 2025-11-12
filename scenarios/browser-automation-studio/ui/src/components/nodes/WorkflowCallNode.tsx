import { memo, FC, useState, useMemo, useEffect } from 'react';
import { Handle, Position, NodeProps, useReactFlow } from 'reactflow';
import { Play, ChevronDown, Settings, AlertCircle } from 'lucide-react';
import { useWorkflowStore } from '@stores/workflowStore';

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
  const workflows = useWorkflowStore((state) => state.workflows);
  const loadWorkflows = useWorkflowStore((state) => state.loadWorkflows);

  useEffect(() => {
    if (!workflows || workflows.length === 0) {
      void loadWorkflows().catch(() => undefined);
    }
  }, [workflows?.length, loadWorkflows]);

  const sortedWorkflows = useMemo(() => {
    if (!workflows || workflows.length === 0) {
      return [] as { id: string; name: string }[];
    }
    return [...workflows]
      .filter((workflow) => typeof workflow?.id === 'string')
      .map((workflow) => ({ id: workflow.id, name: workflow.name }))
      .sort((a, b) => a.name.localeCompare(b.name));
  }, [workflows]);
  
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
            onChange={(event) => {
              const selectedId = event.target.value;
              const match = sortedWorkflows.find((workflow) => workflow.id === selectedId);
              updateNodeData({
                workflowId: selectedId || undefined,
                workflowName: match?.name,
              });
            }}
          >
            <option value="">Select workflow...</option>
            {sortedWorkflows.map((workflow) => (
              <option key={workflow.id} value={workflow.id}>
                {workflow.name}
              </option>
            ))}
          </select>
          {!sortedWorkflows.length && (
            <p className="text-xs text-amber-400 flex items-center gap-1">
              <AlertCircle size={12} /> Load or create workflows to enable linking.
            </p>
          )}
        </div>

        <div className="space-y-1">
          <label className="text-xs text-gray-400">Workflow ID</label>
          <input
            type="text"
            value={data.workflowId || ''}
            onChange={(event) => {
              const value = event.target.value.trim();
              const match = sortedWorkflows.find((workflow) => workflow.id === value);
              updateNodeData({
                workflowId: value || undefined,
                workflowName: match?.name,
              });
            }}
            placeholder="123e4567-e89b-12d3-a456-426614174000"
            className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
          />
        </div>

        <div className="flex items-center gap-2">
          <input
            type="checkbox"
            id={`wait-${id}`}
            checked={data.waitForCompletion ?? true}
            disabled
            readOnly
            className="rounded border-gray-600 bg-flow-bg text-violet-500 opacity-60"
          />
          <label htmlFor={`wait-${id}`} className="text-xs text-gray-300">Wait for completion</label>
        </div>
        <p className="text-xs text-gray-500 -mt-2">Async workflow calls coming soon; current implementation always waits.</p>

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
