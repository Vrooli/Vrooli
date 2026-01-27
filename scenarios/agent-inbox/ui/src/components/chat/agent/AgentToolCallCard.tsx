import { useState } from "react";
import { Wrench, ChevronDown, ChevronRight, Check, X } from "lucide-react";
import type { AgentEvent } from "../../../lib/api";

interface AgentToolCallCardProps {
  event: AgentEvent;
  result?: AgentEvent; // Corresponding tool_result event if available
}

/**
 * Renders a tool call event with collapsible input/output.
 */
export function AgentToolCallCard({ event, result }: AgentToolCallCardProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  const toolName = event.tool_name || "Unknown Tool";
  const toolInput = event.tool_input;

  // Parse input for display
  let parsedInput: object | null = null;
  try {
    if (toolInput) {
      parsedInput = JSON.parse(toolInput);
    }
  } catch {
    // Keep as string if not valid JSON
  }

  return (
    <div className="my-2 rounded-lg border border-zinc-700 overflow-hidden">
      {/* Header */}
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="
          w-full flex items-center gap-3 px-4 py-3
          bg-zinc-800/50 hover:bg-zinc-800
          transition-colors text-left
        "
      >
        <div className="p-1.5 rounded-md bg-orange-500/20">
          <Wrench className="h-4 w-4 text-orange-400" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-medium text-zinc-200">{toolName}</span>
            {result && (
              result.tool_success ? (
                <span className="flex items-center gap-1 text-xs text-green-400">
                  <Check className="h-3 w-3" />
                  Success
                </span>
              ) : (
                <span className="flex items-center gap-1 text-xs text-red-400">
                  <X className="h-3 w-3" />
                  Failed
                </span>
              )
            )}
          </div>
          <div className="text-xs text-zinc-500">
            {new Date(event.timestamp).toLocaleTimeString()}
          </div>
        </div>
        {isExpanded ? (
          <ChevronDown className="h-4 w-4 text-zinc-400" />
        ) : (
          <ChevronRight className="h-4 w-4 text-zinc-400" />
        )}
      </button>

      {/* Expanded content */}
      {isExpanded && (
        <div className="border-t border-zinc-700">
          {/* Input */}
          <div className="p-4">
            <div className="text-xs font-medium text-zinc-400 mb-2">Input</div>
            <pre className="text-sm text-zinc-300 bg-zinc-900 rounded-md p-3 overflow-x-auto">
              {parsedInput
                ? JSON.stringify(parsedInput, null, 2)
                : toolInput || "(no input)"
              }
            </pre>
          </div>

          {/* Output (if result available) */}
          {result && (
            <div className="p-4 pt-0">
              <div className="text-xs font-medium text-zinc-400 mb-2">Output</div>
              <pre
                className={`
                  text-sm rounded-md p-3 overflow-x-auto max-h-64
                  ${result.tool_success
                    ? "text-zinc-300 bg-zinc-900"
                    : "text-red-300 bg-red-900/20"
                  }
                `}
              >
                {result.tool_output || "(no output)"}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default AgentToolCallCard;
