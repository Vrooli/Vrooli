import { memo, FC, useId } from 'react';
import type { NodeProps } from 'reactflow';
import { Recycle, Sparkles } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import { useWorkflowVariables } from '@hooks/useWorkflowVariables';
import { useSyncedString, useSyncedBoolean, textInputHandler } from '@hooks/useSyncedField';
import type { SetVariableParams } from '@utils/actionParams';
import { VariableSuggestionList } from '../components';
import BaseNode from './BaseNode';
import { NodeTextArea, NodeCheckbox } from './fields';

const UseVariableNode: FC<NodeProps> = ({ selected, id }) => {
  // V2 Native: Use action params as source of truth
  const { params, updateParams } = useActionParams<SetVariableParams>(id);
  const availableVariables = useWorkflowVariables(id);
  const nameDatalistId = useId();
  const aliasDatalistId = useId();

  // Action params fields using useSyncedField hooks
  const name = useSyncedString(params?.name ?? '', {
    onCommit: (v) => updateParams({ name: v || undefined }),
  });
  const storeAs = useSyncedString(params?.storeAs ?? '', {
    onCommit: (v) => updateParams({ storeAs: v || undefined }),
  });
  const transform = useSyncedString(params?.transform ?? '', {
    onCommit: (v) => updateParams({ transform: v || undefined }),
  });
  const required = useSyncedBoolean(params?.required ?? false, {
    onCommit: (v) => updateParams({ required: v }),
  });

  return (
    <BaseNode selected={selected} icon={Recycle} iconClassName="text-sky-300" title="Use Variable">
      <div className="space-y-2 text-xs">
        {/* Variable Name - custom layout for datalist autocomplete */}
        <div>
          <label className="text-[11px] font-semibold text-gray-400">Variable Name</label>
          <input
            type="text"
            className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            value={name.value}
            onChange={textInputHandler(name.setValue)}
            onBlur={name.commit}
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
              name.setValue(value);
              updateParams({ name: value });
            }}
          />
        </div>
        {/* Store Result As - custom layout for datalist autocomplete */}
        <div>
          <label className="text-[11px] font-semibold text-gray-400">Store Result As</label>
          <input
            type="text"
            className="w-full px-2 py-1 mt-1 bg-flow-bg rounded border border-gray-700 focus:border-flow-accent focus:outline-none"
            value={storeAs.value}
            onChange={textInputHandler(storeAs.setValue)}
            onBlur={storeAs.commit}
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
              storeAs.setValue(value);
              updateParams({ storeAs: value });
            }}
          />
        </div>
        <div>
          <label className="text-[11px] font-semibold text-gray-400 flex items-center gap-1">
            <Sparkles size={12} /> Transform Template
          </label>
          <NodeTextArea
            field={transform}
            label=""
            rows={3}
            placeholder="Hello, {{value}}!"
            description={`Use {{value}} to reference the current variable contents.`}
          />
        </div>
        <NodeCheckbox field={required} label="Fail if variable is missing" />
      </div>
    </BaseNode>
  );
};

export default memo(UseVariableNode);
