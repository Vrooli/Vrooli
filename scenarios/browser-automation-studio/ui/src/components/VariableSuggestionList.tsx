import { memo } from "react";
import type { WorkflowVariableInfo } from "../hooks/useWorkflowVariables";
import { testIds } from "../consts/selectors";

interface VariableSuggestionListProps {
  variables: WorkflowVariableInfo[];
  emptyHint?: string;
  onSelect?: (value: string) => void;
}

function VariableSuggestionList({
  variables,
  emptyHint,
  onSelect,
}: VariableSuggestionListProps) {
  if (!variables || variables.length === 0) {
    return (
      <p
        className="mt-1 text-[10px] text-gray-500"
        data-testid={testIds.variableSuggestionsEmpty}
      >
        {emptyHint ??
          "Define a variable earlier in the workflow to reference it here."}
      </p>
    );
  }

  return (
    <div
      className="mt-1 flex flex-wrap gap-1"
      data-testid={testIds.variableSuggestions}
    >
      {variables.map((variable) => (
        <button
          key={`${variable.name}-${variable.sourceNodeId}`}
          type="button"
          className="px-2 py-0.5 text-[10px] rounded border border-gray-700 text-gray-300 hover:text-white hover:border-flow-accent transition-colors"
          title={variable.sourceLabel}
          onClick={() => onSelect?.(variable.name)}
        >
          {variable.name}
        </button>
      ))}
    </div>
  );
}

export default memo(VariableSuggestionList);
