import {
  FileText,
  Loader2,
  AlertTriangle,
} from "lucide-react";
import { CodeBlock } from "../../ui/code-block";
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
