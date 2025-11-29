import { memo, FC, useState, useEffect, useCallback, useId } from 'react';
import { Handle, NodeProps, Position, useReactFlow } from 'reactflow';
import { Recycle, Sparkles } from 'lucide-react';
import { useWorkflowVariables } from '@hooks/useWorkflowVariables';
import { VariableSuggestionList } from '../components';

const UseVariableNode: FC<NodeProps> = ({ data, selected, id }) => {
  const nodeData = (data ?? {}) as Record<string, unknown>;
  const { getNodes, setNodes } = useReactFlow();
  const availableVariables = useWorkflowVariables(id);
  const nameDatalistId = useId();
  const aliasDatalistId = useId();

  const [name, setName] = useState<string>(String(nodeData.name ?? ''));
  const [storeAs, setStoreAs] = useState<string>(String(nodeData.storeAs ?? ''));
  const [transform, setTransform] = useState<string>(String(nodeData.transform ?? ''));
  const [required, setRequired] = useState<boolean>(Boolean(nodeData.required));

  useEffect(() => {
    setName(String(nodeData.name ?? ''));
  }, [nodeData.name]);
  useEffect(() => {
    setStoreAs(String(nodeData.storeAs ?? ''));
  }, [nodeData.storeAs]);
  useEffect(() => {
    setTransform(String(nodeData.transform ?? ''));
  }, [nodeData.transform]);
  useEffect(() => {
    setRequired(Boolean(nodeData.required));
  }, [nodeData.required]);

  const updateNodeData = useCallback((updates: Record<string, unknown>) => {
    const nodes = getNodes();
    setNodes(nodes.map((node) => {
      if (node.id !== id) {
        return node;
      }
      const nextData = { ...(node.data ?? {}) } as Record<string, unknown>;
      Object.entries(updates).forEach(([key, value]) => {
        if (value === undefined || value === null || value === '') {
          delete nextData[key];
        } else {
          nextData[key] = value;
        }
      });
      return { ...node, data: nextData };
    }));
  }, [getNodes, setNodes, id]);

  return (
    <div className={`workflow-node ${selected ? 'selected' : ''}`}>
      <Handle type="target" position={Position.Top} className="node-handle" />

      <div className="flex items-center gap-2 mb-2">
        <Recycle size={16} className="text-sky-300" />
        <span className="font-semibold text-sm">Use Variable</span>
      </div>

      <div className="space-y-2 text-xs">
        <div>
          <label className="text-[11px] font-semibold text-gray-400">Variable Name</label>
          <input
            type="text"
            className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            value={name}
            onChange={(event) => setName(event.target.value)}
            onBlur={() => updateNodeData({ name })}
            placeholder="loginToken"
            list={availableVariables.length > 0 ? nameDatalistId : undefined}
          />
          {availableVariables.length > 0 && (
            <datalist id={nameDatalistId}>
              {availableVariables.map((variable) => (
                <option key={`${variable.name}-${variable.sourceNodeId}`} value={variable.name}>
                  {variable.sourceLabel}
                </option>
              ))}
            </datalist>
          )}
          <VariableSuggestionList
            variables={availableVariables}
            emptyHint="Add a Set Variable or store a result earlier in the workflow to reference it here."
            onSelect={(value) => {
              setName(value);
              updateNodeData({ name: value });
            }}
          />
        </div>
        <div>
          <label className="text-[11px] font-semibold text-gray-400">Store Result As</label>
          <input
            type="text"
            className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            value={storeAs}
            onChange={(event) => setStoreAs(event.target.value)}
            onBlur={() => updateNodeData({ storeAs })}
            placeholder="Optional alias"
            list={availableVariables.length > 0 ? aliasDatalistId : undefined}
          />
          {availableVariables.length > 0 && (
            <datalist id={aliasDatalistId}>
              {availableVariables.map((variable) => (
                <option key={`${variable.name}-${variable.sourceNodeId}`} value={variable.name}>
                  {variable.sourceLabel}
                </option>
              ))}
            </datalist>
          )}
          <VariableSuggestionList
            variables={availableVariables}
            emptyHint="Aliases become new variables future nodes can reuse."
            onSelect={(value) => {
              setStoreAs(value);
              updateNodeData({ storeAs: value });
            }}
          />
        </div>
        <div>
          <label className="text-[11px] font-semibold text-gray-400 flex items-center gap-1">
            <Sparkles size={12} /> Transform Template
          </label>
          <textarea
            className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            rows={3}
            value={transform}
            onChange={(event) => setTransform(event.target.value)}
            onBlur={() => updateNodeData({ transform })}
            placeholder="Hello, {{value}}!"
          />
          <p className="text-[10px] text-gray-500 mt-1">Use {`{{value}}`} to reference the current variable contents.</p>
        </div>
        <label className="flex items-center gap-2 text-gray-400">
          <input
            type="checkbox"
            checked={required}
            onChange={(event) => {
              setRequired(event.target.checked);
              updateNodeData({ required: event.target.checked });
            }}
          />
          Fail if variable is missing
        </label>
      </div>

      <Handle type="source" position={Position.Bottom} className="node-handle" />
    </div>
  );
};

export default memo(UseVariableNode);
