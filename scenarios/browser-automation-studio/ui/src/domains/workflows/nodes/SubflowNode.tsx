import { memo, FC, useState, useMemo, useEffect } from 'react';
import type { NodeProps } from 'reactflow';
import { Play, ChevronDown, Settings, AlertCircle } from 'lucide-react';
import { useNodeData } from '@hooks/useNodeData';
import { useWorkflowStore } from '@stores/workflowStore';
import BaseNode from './BaseNode';

const SubflowNode: FC<NodeProps> = ({ selected, id }) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const { getValue, updateData } = useNodeData(id);
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

  const workflowId = getValue<string>('workflowId') ?? '';
  const parameters = getValue<Record<string, unknown>>('parameters');
  const outputMapping = getValue<Record<string, string>>('outputMapping');

  return (
    <BaseNode selected={selected} icon={Play} iconClassName="text-violet-400" title="Subflow" className="w-80">
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="absolute top-3 right-3 p-1 hover:bg-gray-700 rounded"
        title={isExpanded ? 'Show less' : 'Show more'}
      >
        <ChevronDown
          size={14}
          className={`transform transition-transform text-gray-400 ${isExpanded ? 'rotate-180' : ''}`}
        />
      </button>

      <div className="space-y-3">
        <div className="space-y-1">
          <label className="text-xs text-gray-400">Target workflow</label>
          <select
            value={workflowId}
            className="w-full px-2 py-1 bg-flow-bg rounded text-xs border border-gray-700 focus:border-flow-accent focus:outline-none"
            onChange={(event) => {
              const selectedId = event.target.value;
              const match = sortedWorkflows.find((workflow) => workflow.id === selectedId);
              updateData({
                workflowId: selectedId || undefined,
                workflowName: match?.name,
              });
            }}
          >
            <option value="">Select workflow…</option>
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
            value={workflowId}
            onChange={(event) => {
              const value = event.target.value.trim();
              const match = sortedWorkflows.find((workflow) => workflow.id === value);
              updateData({
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
            checked={true}
            disabled
            readOnly
            className="rounded border-gray-600 bg-flow-bg text-violet-500 opacity-60"
          />
          <label htmlFor={`wait-${id}`} className="text-xs text-gray-300">
            Wait for completion
          </label>
        </div>
        <p className="text-xs text-gray-500 -mt-2">
          Async subflows coming soon; current implementation always waits.
        </p>

        {isExpanded && (
          <div className="space-y-3 pt-3 border-t border-gray-700">
            <div className="space-y-1">
              <label className="flex items-center gap-1 text-xs text-gray-400">
                <Settings size={12} />
                Parameters (JSON)
              </label>
              <textarea
                value={parameters ? JSON.stringify(parameters, null, 2) : '{}'}
                onChange={(e) => {
                  try {
                    const parsed = JSON.parse(e.target.value);
                    updateData({ parameters: parsed });
                  } catch {
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
                Output mapping (JSON)
              </label>
              <textarea
                value={outputMapping ? JSON.stringify(outputMapping, null, 2) : '{}'}
                onChange={(e) => {
                  try {
                    const parsed = JSON.parse(e.target.value);
                    updateData({ outputMapping: parsed });
                  } catch {
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
    </BaseNode>
  );
};

export default memo(SubflowNode);
