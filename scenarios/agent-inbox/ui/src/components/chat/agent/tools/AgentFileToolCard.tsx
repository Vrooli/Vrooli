import { FileText, FolderSearch, Search, FileEdit, FilePlus } from "lucide-react";
import type { ToolCardProps } from ".";
import { ToolCardShell } from "./ToolCardShell";
import { CodeBlock } from "../../../markdown/components/CodeBlock";

/** Icon and label config per file tool. */
const FILE_TOOL_CONFIG: Record<string, {
  icon: React.ReactNode;
  iconBg: string;
  label: string;
}> = {
  Read: {
    icon: <FileText className="h-4 w-4 text-blue-400" />,
    iconBg: "bg-blue-500/20",
    label: "Read",
  },
  Write: {
    icon: <FilePlus className="h-4 w-4 text-green-400" />,
    iconBg: "bg-green-500/20",
    label: "Write",
  },
  Edit: {
    icon: <FileEdit className="h-4 w-4 text-yellow-400" />,
    iconBg: "bg-yellow-500/20",
    label: "Edit",
  },
  Glob: {
    icon: <FolderSearch className="h-4 w-4 text-purple-400" />,
    iconBg: "bg-purple-500/20",
    label: "Glob",
  },
  Grep: {
    icon: <Search className="h-4 w-4 text-cyan-400" />,
    iconBg: "bg-cyan-500/20",
    label: "Grep",
  },
};

const DEFAULT_CONFIG = {
  icon: <FileText className="h-4 w-4 text-zinc-400" />,
  iconBg: "bg-zinc-600/20",
  label: "File",
};

/**
 * Renders Read, Write, Edit, Glob, and Grep tool call events.
 *
 * Collapsed: Shows the file path or search pattern prominently.
 * Expanded: Shows file content or search results via CodeBlock,
 *           with language detection from the file extension.
 */
export function AgentFileToolCard({ event, result }: ToolCardProps) {
  const toolName = event.tool_name || "File";
  const config = FILE_TOOL_CONFIG[toolName] || DEFAULT_CONFIG;
  const parsed = parseToolInput(event.tool_input);

  // Extract the most meaningful summary from tool input
  const summary = extractSummary(toolName, parsed);

  // Detect language from file path for syntax highlighting
  const filePath = (parsed?.file_path || parsed?.path || "") as string;
  const language = detectLanguageFromPath(filePath);

  return (
    <ToolCardShell
      icon={config.icon}
      iconBg={config.iconBg}
      toolName={config.label}
      summary={summary}
      timestamp={event.timestamp}
      result={result}
    >
      {/* Input details */}
      {parsed && (
        <div className="p-4 pb-2">
          <div className="text-xs font-medium text-zinc-400 mb-1">Input</div>
          <CodeBlock code={JSON.stringify(parsed, null, 2)} language="json" />
        </div>
      )}

      {/* Output */}
      {result?.tool_output && (
        <div className="px-4 pb-4">
          <div className="text-xs font-medium text-zinc-400 mb-1">Output</div>
          <div className={result.tool_success === false ? "border border-red-500/30 rounded-lg overflow-hidden" : ""}>
            <CodeBlock code={result.tool_output} language={language} />
          </div>
        </div>
      )}
    </ToolCardShell>
  );
}

function extractSummary(toolName: string, parsed: Record<string, unknown> | null): string {
  if (!parsed) return "(no input)";

  switch (toolName) {
    case "Read":
    case "Write":
    case "Edit":
      return (parsed.file_path as string) || "(no path)";
    case "Glob":
      return (parsed.pattern as string) || "(no pattern)";
    case "Grep":
      return `${(parsed.pattern as string) || "?"} in ${(parsed.path as string) || "."}`;
    default:
      return (parsed.file_path as string) || (parsed.path as string) || "(no path)";
  }
}

function detectLanguageFromPath(path: string): string {
  const ext = path.split(".").pop()?.toLowerCase();
  const EXT_MAP: Record<string, string> = {
    ts: "typescript", tsx: "tsx", js: "javascript", jsx: "jsx",
    py: "python", go: "go", rs: "rust", java: "java",
    json: "json", yaml: "yaml", yml: "yaml", toml: "text",
    md: "markdown", html: "html", css: "css", sql: "sql",
    sh: "bash", bash: "bash", zsh: "bash",
    c: "c", cpp: "cpp", h: "c", hpp: "cpp",
    rb: "ruby", php: "php", swift: "swift", kt: "kotlin",
  };
  return (ext && EXT_MAP[ext]) || "text";
}

function parseToolInput(input?: string): Record<string, unknown> | null {
  if (!input) return null;
  try {
    return JSON.parse(input) as Record<string, unknown>;
  } catch {
    return null;
  }
}
