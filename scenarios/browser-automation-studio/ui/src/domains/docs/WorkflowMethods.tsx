import { MarkdownRenderer } from "./MarkdownRenderer";
import { WORKFLOW_METHODS_CONTENT } from "./content/workflowMethods";

export function WorkflowMethods() {
  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-4xl mx-auto p-6">
        <MarkdownRenderer content={WORKFLOW_METHODS_CONTENT} />
      </div>
    </div>
  );
}
