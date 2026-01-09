import { useRef, useEffect, useState } from "react";
import {
  Terminal,
  Copy,
  Check,
  ArrowDown,
  FileText,
} from "lucide-react";
import { getLogLevelInfo } from "../../../hooks/useLiveState";
import { cn } from "../../../lib/utils";
import type { LogEntry } from "../../../lib/api";

interface LogViewerProps {
  logs: LogEntry[];
  total: number;
  sources: string[];
}

export function LogViewer({ logs, total, sources }: LogViewerProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);
  const [copied, setCopied] = useState(false);

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (autoScroll && containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [logs, autoScroll]);

  const handleScroll = () => {
    if (containerRef.current) {
      const { scrollTop, scrollHeight, clientHeight } = containerRef.current;
      const isAtBottom = scrollHeight - scrollTop - clientHeight < 50;
      setAutoScroll(isAtBottom);
    }
  };

  const handleCopyLogs = async () => {
    const text = logs.map(l => `[${l.timestamp}] [${l.level}] [${l.source}] ${l.message}`).join("\n");
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const scrollToBottom = () => {
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
      setAutoScroll(true);
    }
  };

  if (logs.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-400">
        <Terminal className="h-12 w-12 mb-4 opacity-50" />
        <p className="text-sm">No logs available</p>
        <p className="text-xs text-slate-500 mt-1">
          Logs will appear once the deployment is running
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {/* Log stats header */}
      <div className="flex items-center justify-between px-3 py-2 rounded-lg border border-white/10 bg-slate-900/50">
        <div className="flex items-center gap-4 text-xs text-slate-400">
          <span className="flex items-center gap-1">
            <FileText className="h-3 w-3" />
            {total} lines
          </span>
          <span className="text-slate-600">|</span>
          <span>
            Sources: {sources.length > 0 ? sources.join(", ") : "none"}
          </span>
        </div>

        <div className="flex items-center gap-2">
          {/* Copy button */}
          <button
            onClick={handleCopyLogs}
            className={cn(
              "flex items-center gap-1.5 px-2 py-1 rounded text-xs transition-colors",
              "border border-white/10 hover:bg-white/5",
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

          {/* Scroll to bottom button (only show when not at bottom) */}
          {!autoScroll && (
            <button
              onClick={scrollToBottom}
              className="flex items-center gap-1.5 px-2 py-1 rounded text-xs transition-colors border border-blue-500/30 bg-blue-500/10 text-blue-400 hover:bg-blue-500/20"
            >
              <ArrowDown className="h-3 w-3" />
              Latest
            </button>
          )}
        </div>
      </div>

      {/* Log container */}
      <div
        ref={containerRef}
        onScroll={handleScroll}
        className="h-[500px] overflow-y-auto rounded-lg border border-white/10 bg-slate-950 font-mono text-xs"
      >
        <table className="w-full">
          <tbody>
            {logs.map((log, idx) => {
              const levelInfo = getLogLevelInfo(log.level);
              return (
                <tr
                  key={idx}
                  className={cn(
                    "border-b border-white/5 hover:bg-white/5",
                    log.level === "ERROR" && "bg-red-500/5"
                  )}
                >
                  {/* Timestamp */}
                  <td className="py-1 px-2 text-slate-500 whitespace-nowrap align-top w-[180px]">
                    {formatLogTimestamp(log.timestamp)}
                  </td>

                  {/* Level badge */}
                  <td className="py-1 px-2 align-top w-[60px]">
                    <span
                      className={cn(
                        "inline-block px-1.5 py-0.5 rounded text-[10px] font-medium uppercase",
                        levelInfo.color,
                        levelInfo.bg
                      )}
                    >
                      {log.level}
                    </span>
                  </td>

                  {/* Source */}
                  <td className="py-1 px-2 text-purple-400 whitespace-nowrap align-top w-[100px]">
                    {log.source}
                  </td>

                  {/* Message */}
                  <td className="py-1 px-2 text-slate-300 break-all">
                    <LogMessage message={log.message} />
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {/* Auto-scroll indicator */}
      <div className="flex items-center justify-end gap-2 text-xs text-slate-500">
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={autoScroll}
            onChange={(e) => setAutoScroll(e.target.checked)}
            className="rounded border-white/10 bg-slate-800 text-blue-500"
          />
          Auto-scroll to latest
        </label>
      </div>
    </div>
  );
}

// Format log timestamp for display
function formatLogTimestamp(timestamp: string): string {
  try {
    const date = new Date(timestamp);
    if (isNaN(date.getTime())) {
      return timestamp; // Return as-is if not a valid date
    }
    return date.toISOString().slice(11, 23); // HH:mm:ss.SSS
  } catch {
    return timestamp;
  }
}

// Log message with syntax highlighting for common patterns
function LogMessage({ message }: { message: string }) {
  // Highlight HTTP methods
  const parts = message.split(/(\b(?:GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\b)/g);

  // Highlight numbers
  const highlightNumbers = (text: string) => {
    return text.split(/(\d+(?:\.\d+)?(?:ms|s|m|h|MB|GB|KB|B)?)/g).map((part, i) => {
      if (/^\d+(?:\.\d+)?(?:ms|s|m|h|MB|GB|KB|B)?$/.test(part)) {
        return (
          <span key={i} className="text-amber-400">
            {part}
          </span>
        );
      }
      return part;
    });
  };

  // Highlight URLs and paths
  const highlightPaths = (text: string) => {
    return text.split(/((?:\/[\w\-\.]+)+|https?:\/\/[^\s]+)/g).map((part, i) => {
      if (part.startsWith("/") || part.startsWith("http")) {
        return (
          <span key={i} className="text-cyan-400">
            {part}
          </span>
        );
      }
      return highlightNumbers(part);
    });
  };

  return (
    <>
      {parts.map((part, i) => {
        if (/^(?:GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)$/.test(part)) {
          return (
            <span key={i} className="text-emerald-400 font-medium">
              {part}
            </span>
          );
        }
        return <span key={i}>{highlightPaths(part)}</span>;
      })}
    </>
  );
}
