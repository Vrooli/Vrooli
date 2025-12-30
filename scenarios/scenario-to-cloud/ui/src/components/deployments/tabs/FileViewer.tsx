import { useState } from "react";
import {
  Copy,
  Check,
  FileText,
  Loader2,
  AlertTriangle,
  Download,
} from "lucide-react";
import { cn } from "../../../lib/utils";
import { formatBytes } from "../../../hooks/useLiveState";

interface FileViewerProps {
  path: string | null;
  content: string | undefined;
  isLoading: boolean;
  truncated?: boolean;
  sizeBytes?: number;
}

export function FileViewer({
  path,
  content,
  isLoading,
  truncated,
  sizeBytes,
}: FileViewerProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    if (!content) return;
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  };

  if (!path) {
    return (
      <div className="flex flex-col items-center justify-center h-48 text-slate-500">
        <FileText className="h-12 w-12 mb-2 opacity-50" />
        <p className="text-sm">Select a file to view its contents</p>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-48">
        <Loader2 className="h-8 w-8 animate-spin text-blue-400" />
      </div>
    );
  }

  if (content === undefined) {
    return (
      <div className="flex flex-col items-center justify-center h-48 text-slate-500">
        <FileText className="h-12 w-12 mb-2 opacity-50" />
        <p className="text-sm">Unable to load file content</p>
      </div>
    );
  }

  const filename = path.split("/").pop() || path;
  const language = getLanguage(filename);

  return (
    <div className="space-y-2">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm">
          <span className="text-slate-400 truncate max-w-[200px]" title={path}>
            {filename}
          </span>
          {sizeBytes !== undefined && (
            <span className="text-slate-600">
              ({formatBytes(sizeBytes)})
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handleCopy}
            className={cn(
              "flex items-center gap-1.5 px-2 py-1 rounded text-xs",
              "border border-white/10 hover:bg-white/5 transition-colors",
              copied && "text-emerald-400 border-emerald-500/30"
            )}
          >
            {copied ? (
              <>
                <Check className="h-3 w-3" />
                Copied
              </>
            ) : (
              <>
                <Copy className="h-3 w-3" />
                Copy
              </>
            )}
          </button>
        </div>
      </div>

      {/* Truncation warning */}
      {truncated && (
        <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-amber-500/10 border border-amber-500/30 text-amber-400 text-xs">
          <AlertTriangle className="h-4 w-4" />
          <span>File truncated (showing first 1MB)</span>
        </div>
      )}

      {/* Content */}
      <div className="relative">
        <pre className={cn(
          "text-xs bg-slate-950 p-4 rounded-lg overflow-auto max-h-[400px]",
          "font-mono text-slate-300 leading-relaxed",
          language && "language-" + language
        )}>
          <code>
            {content || <span className="text-slate-600 italic">Empty file</span>}
          </code>
        </pre>

        {/* Line numbers overlay */}
        {content && content.split("\n").length > 1 && (
          <div className="absolute left-0 top-0 p-4 pointer-events-none select-none">
            <div className="text-xs font-mono text-slate-600 leading-relaxed">
              {content.split("\n").map((_, i) => (
                <div key={i} className="text-right pr-4" style={{ minWidth: "3ch" }}>
                  {i + 1}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// Helper to determine language from filename
function getLanguage(filename: string): string | null {
  const ext = filename.split(".").pop()?.toLowerCase();

  const languageMap: Record<string, string> = {
    ts: "typescript",
    tsx: "typescript",
    js: "javascript",
    jsx: "javascript",
    json: "json",
    go: "go",
    py: "python",
    rs: "rust",
    sh: "bash",
    bash: "bash",
    yml: "yaml",
    yaml: "yaml",
    md: "markdown",
    sql: "sql",
    html: "html",
    css: "css",
    toml: "toml",
  };

  return ext ? languageMap[ext] || null : null;
}
