import { memo, FC, useState, useEffect, useId } from 'react';
import type { NodeProps } from 'reactflow';
import { Recycle, Sparkles } from 'lucide-react';
import { useNodeData } from '@hooks/useNodeData';
import { useWorkflowVariables } from '@hooks/useWorkflowVariables';
import { VariableSuggestionList } from '../components';
import BaseNode from './BaseNode';

const UseVariableNode: FC<NodeProps> = ({ selected, id }) => {
  const { getValue, updateData } = useNodeData(id);
  const availableVariables = useWorkflowVariables(id);
  const nameDatalistId = useId();
  const aliasDatalistId = useId();

  const [name, setName] = useState<string>(getValue<string>('name') ?? '');
  const [storeAs, setStoreAs] = useState<string>(getValue<string>('storeAs') ?? '');
  const [transform, setTransform] = useState<string>(getValue<string>('transform') ?? '');
  const [required, setRequired] = useState<boolean>(getValue<boolean>('required') ?? false);

  useEffect(() => {
    setName(getValue<string>('name') ?? '');
  }, [getValue]);

  useEffect(() => {
    setStoreAs(getValue<string>('storeAs') ?? '');
  }, [getValue]);

  useEffect(() => {
    setTransform(getValue<string>('transform') ?? '');
  }, [getValue]);

  useEffect(() => {
    setRequired(getValue<boolean>('required') ?? false);
  }, [getValue]);

  return (
    <BaseNode selected={selected} icon={Recycle} iconClassName="text-sky-300" title="Use Variable">
      <div className="space-y-2 text-xs">
        <div>
          <label className="text-[11px] font-semibold text-gray-400">Variable Name</label>
          <input
            type="text"
            className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            value={name}
            onChange={(event) => setName(event.target.value)}
            onBlur={() => updateData({ name })}
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
              updateData({ name: value });
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
            onBlur={() => updateData({ storeAs })}
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
              updateData({ storeAs: value });
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
            onBlur={() => updateData({ transform })}
            placeholder="Hello, {{value}}!"
          />
          <p className="text-[10px] text-gray-500 mt-1">
            Use {`{{value}}`} to reference the current variable contents.
          </p>
        </div>
        <label className="flex items-center gap-2 text-gray-400">
          <input
            type="checkbox"
            checked={required}
            onChange={(event) => {
              setRequired(event.target.checked);
              updateData({ required: event.target.checked });
            }}
          />
          Fail if variable is missing
        </label>
      </div>
    </BaseNode>
  );
};

export default memo(UseVariableNode);
