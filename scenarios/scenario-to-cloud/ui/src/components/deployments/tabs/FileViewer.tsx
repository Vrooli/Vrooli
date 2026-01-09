import {
  FileText,
  Loader2,
  AlertTriangle,
  AlertCircle,
} from "lucide-react";
import { CodeBlock } from "../../ui/code-block";
import { formatBytes } from "../../../hooks/useLiveState";

interface FileViewerProps {
  path: string | null;
  content: string | undefined;
  isLoading: boolean;
  truncated?: boolean;
  sizeBytes?: number;
  error?: Error | null;
}

export function FileViewer({
  path,
  content,
  isLoading,
  truncated,
  sizeBytes,
  error,
}: FileViewerProps) {
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

  if (error) {
    // Parse error message to provide better context
    const errorMessage = error.message || "Unknown error";
    let displayMessage = "Unable to load file content";
    let hint = "";

    if (errorMessage.includes("path_not_allowed")) {
      displayMessage = "Access denied";
      hint = "This path is outside the allowed directory.";
    } else if (errorMessage.includes("404") || errorMessage.includes("not found")) {
      displayMessage = "File not found";
      hint = "The file may have been moved or deleted.";
    } else if (errorMessage.includes("ssh_failed")) {
      displayMessage = "SSH connection failed";
      hint = "Unable to connect to the VPS.";
    } else if (errorMessage.includes("400")) {
      displayMessage = "Invalid request";
      hint = errorMessage;
    } else {
      hint = errorMessage;
    }

    return (
      <div className="flex flex-col items-center justify-center h-48 text-slate-500">
        <AlertCircle className="h-12 w-12 mb-2 text-red-400 opacity-70" />
        <p className="text-sm font-medium text-red-400">{displayMessage}</p>
        {hint && (
          <p className="text-xs text-slate-500 mt-1 max-w-sm text-center">{hint}</p>
        )}
        <p className="text-xs text-slate-600 mt-2 font-mono">{path}</p>
      </div>
    );
  }

  if (content === undefined) {
    return (
      <div className="flex flex-col items-center justify-center h-48 text-slate-500">
        <FileText className="h-12 w-12 mb-2 opacity-50" />
        <p className="text-sm">Unable to load file content</p>
        <p className="text-xs text-slate-600 mt-1">Try selecting the file again</p>
      </div>
    );
  }

  const filename = path.split("/").pop() || path;

  // Handle empty file
  if (!content) {
    return (
      <div className="space-y-2">
        {/* File info header */}
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
        </div>

        <div className="p-6 rounded-lg border border-slate-700 bg-slate-950 text-center">
          <span className="text-slate-600 italic">Empty file</span>
        </div>
      </div>
    );
  }

  // Check if this is a binary file indicator
  const isBinaryIndicator = content.startsWith("[Binary file") ||
                            content.startsWith("[Large binary");

  return (
    <div className="space-y-2">
      {/* File info header - only show path, size, and download; CodeBlock handles copy */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm">
          <span className="text-slate-400 truncate max-w-[300px]" title={path}>
            {path}
          </span>
          {sizeBytes !== undefined && (
            <span className="text-slate-600">
              ({formatBytes(sizeBytes)})
            </span>
          )}
        </div>
      </div>

      {/* Truncation warning */}
      {truncated && (
        <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-amber-500/10 border border-amber-500/30 text-amber-400 text-xs">
          <AlertTriangle className="h-4 w-4 flex-shrink-0" />
          <span>File truncated (showing first 1MB)</span>
        </div>
      )}

      {/* Binary file warning */}
      {isBinaryIndicator ? (
        <div className="flex flex-col items-center justify-center h-48 rounded-lg border border-slate-700 bg-slate-950 text-slate-500">
          <FileText className="h-12 w-12 mb-2 opacity-50" />
          <p className="text-sm">{content}</p>
          <p className="text-xs mt-2 text-slate-600">
            Binary files cannot be previewed
          </p>
        </div>
      ) : (
        /* Code content with syntax highlighting */
        <CodeBlock
          code={content}
          filename={filename}
          maxHeight="500px"
          showLineNumbers={true}
          showHeader={true}
        />
      )}
    </div>
  );
}
