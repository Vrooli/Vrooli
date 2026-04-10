import { Terminal } from "lucide-react";
import type { ToolCardProps } from ".";
import { ToolCardShell } from "./ToolCardShell";
import { CodeBlock } from "../../../markdown/components/CodeBlock";

/**
 * Renders a Bash tool call event.
 *
 * Collapsed: Shows the description prominently (what the command does).
 * Expanded: Shows the full command via CodeBlock with bash syntax highlighting,
 *           and the output (if result available).
 */
export function AgentBashCard({ event, result, viewMode }: ToolCardProps) {
  const parsed = parseToolInput(event.tool_input);
  const description = parsed?.description as string | undefined;
  const command = parsed?.command as string | undefined;

  return (
    <ToolCardShell
      icon={<Terminal className="h-4 w-4 text-orange-400" />}
      iconBg="bg-orange-500/20"
      toolName="Bash"
      summary={description || command || "(no command)"}
      timestamp={event.timestamp}
      result={result}
      compact={viewMode === "compact"}
    >
      {/* Command */}
      {command && (
        <div className="p-4 pb-2">
          <div className="text-xs font-medium text-zinc-400 mb-1">Command</div>
          <CodeBlock code={command} language="bash" />
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

function parseToolInput(input?: string): Record<string, unknown> | null {
  if (!input) return null;
  try {
    return JSON.parse(input) as Record<string, unknown>;
  } catch {
    return null;
  }
}
