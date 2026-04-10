import { Layers } from "lucide-react";
import type { ToolCardProps } from ".";
import { ToolCardShell } from "./ToolCardShell";
import { CodeBlock } from "../../../markdown/components/CodeBlock";

/**
 * Renders Task (subagent) tool call events.
 *
 * Collapsed: Shows the task description and subagent type.
 * Expanded: Shows the full prompt and output.
 */
export function AgentTaskCard({ event, result, viewMode }: ToolCardProps) {
  const parsed = parseToolInput(event.tool_input);

  const description = (parsed?.description as string) || "(no description)";
  const subagentType = parsed?.subagent_type as string | undefined;
  const prompt = parsed?.prompt as string | undefined;

  const summary = subagentType
    ? `[${subagentType}] ${description}`
    : description;

  return (
    <ToolCardShell
      icon={<Layers className="h-4 w-4 text-violet-400" />}
      iconBg="bg-violet-500/20"
      toolName="Task"
      summary={summary}
      timestamp={event.timestamp}
      result={result}
      compact={viewMode === "compact"}
    >
      {/* Full prompt */}
      {prompt && (
        <div className="p-4 pb-2">
          <div className="text-xs font-medium text-zinc-400 mb-1">Prompt</div>
          <CodeBlock code={prompt} language="text" />
        </div>
      )}

      {/* Subagent type and other params */}
      {parsed && (
        <div className="px-4 pb-2">
          <div className="text-xs font-medium text-zinc-400 mb-1">Parameters</div>
          <div className="flex flex-wrap gap-2">
            {subagentType && (
              <span className="px-2 py-0.5 text-xs rounded bg-violet-500/20 text-violet-300">
                {subagentType}
              </span>
            )}
            {typeof parsed.mode === "string" && (
              <span className="px-2 py-0.5 text-xs rounded bg-zinc-700 text-zinc-300">
                mode: {parsed.mode}
              </span>
            )}
            {typeof parsed.model === "string" && (
              <span className="px-2 py-0.5 text-xs rounded bg-zinc-700 text-zinc-300">
                model: {parsed.model}
              </span>
            )}
          </div>
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
