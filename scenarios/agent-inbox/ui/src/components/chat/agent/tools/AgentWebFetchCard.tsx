import { Globe, Search } from "lucide-react";
import type { ToolCardProps } from ".";
import { ToolCardShell } from "./ToolCardShell";
import { CodeBlock } from "../../../markdown/components/CodeBlock";

/**
 * Renders WebFetch and WebSearch tool call events.
 *
 * Collapsed: Shows the URL (WebFetch) or query (WebSearch) prominently.
 * Expanded: Shows fetched content or search results via CodeBlock.
 */
export function AgentWebFetchCard({ event, result }: ToolCardProps) {
  const toolName = event.tool_name || "WebFetch";
  const isSearch = toolName === "WebSearch";
  const parsed = parseToolInput(event.tool_input);

  const url = parsed?.url as string | undefined;
  const query = parsed?.query as string | undefined;
  const prompt = parsed?.prompt as string | undefined;

  const summary = isSearch
    ? query || "(no query)"
    : url || "(no URL)";

  return (
    <ToolCardShell
      icon={isSearch
        ? <Search className="h-4 w-4 text-emerald-400" />
        : <Globe className="h-4 w-4 text-sky-400" />
      }
      iconBg={isSearch ? "bg-emerald-500/20" : "bg-sky-500/20"}
      toolName={isSearch ? "WebSearch" : "WebFetch"}
      summary={summary}
      timestamp={event.timestamp}
      result={result}
    >
      {/* URL as clickable link (WebFetch) */}
      {url && (
        <div className="px-4 pt-3">
          <div className="text-xs font-medium text-zinc-400 mb-1">URL</div>
          <a
            href={url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-sm text-sky-400 hover:text-sky-300 hover:underline break-all"
          >
            {url}
          </a>
        </div>
      )}

      {/* Prompt (WebFetch) */}
      {prompt && (
        <div className="px-4 pt-3">
          <div className="text-xs font-medium text-zinc-400 mb-1">Prompt</div>
          <p className="text-sm text-zinc-300">{prompt}</p>
        </div>
      )}

      {/* Query (WebSearch) */}
      {isSearch && query && (
        <div className="px-4 pt-3">
          <div className="text-xs font-medium text-zinc-400 mb-1">Query</div>
          <p className="text-sm text-zinc-300">{query}</p>
        </div>
      )}

      {/* Output */}
      {result?.tool_output && (
        <div className="p-4">
          <div className="text-xs font-medium text-zinc-400 mb-1">Response</div>
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
