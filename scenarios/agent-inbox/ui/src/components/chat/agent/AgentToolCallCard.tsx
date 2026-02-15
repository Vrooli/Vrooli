import { Wrench } from "lucide-react";
import type { ToolCardProps } from "./tools";
import { ToolCardShell } from "./tools/ToolCardShell";
import { CodeBlock } from "../../markdown/components/CodeBlock";

/**
 * Generic fallback for tool call events that don't have a dedicated component.
 * Renders collapsible input/output using CodeBlock for syntax highlighting.
 */
export function AgentToolCallCard({ event, result, viewMode }: ToolCardProps) {
  const toolName = event.tool_name || "Unknown Tool";

  // Format input for display
  let displayInput: string | null = null;
  if (event.tool_input) {
    try {
      displayInput = JSON.stringify(JSON.parse(event.tool_input), null, 2);
    } catch {
      displayInput = event.tool_input;
    }
  }

  return (
    <ToolCardShell
      icon={<Wrench className="h-4 w-4 text-orange-400" />}
      iconBg="bg-orange-500/20"
      toolName={toolName}
      timestamp={event.timestamp}
      result={result}
      compact={viewMode === "compact"}
    >
      {/* Input */}
      {displayInput && (
        <div className="p-4 pb-2">
          <div className="text-xs font-medium text-zinc-400 mb-1">Input</div>
          <CodeBlock code={displayInput} language="json" />
        </div>
      )}

      {/* Output */}
      {result?.tool_output && (
        <div className="px-4 pb-4">
          <div className="text-xs font-medium text-zinc-400 mb-1">Output</div>
          <div className={result.tool_success === false ? "border border-red-500/30 rounded-lg overflow-hidden" : ""}>
            <CodeBlock code={result.tool_output} language="text" />
          </div>
        </div>
      )}
    </ToolCardShell>
  );
}

export default AgentToolCallCard;
