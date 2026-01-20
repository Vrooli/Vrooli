/**
 * AsyncResultChip - Reference chip displayed in MessageInput area.
 *
 * Similar to attachment previews, shows that an async operation result
 * will be included in the next message for context.
 */

import { Paperclip, X, ExternalLink, CheckCircle2 } from "lucide-react";
import { Button } from "../ui/button";

/** Reference to an async operation result */
export interface AsyncResultReference {
  tool_call_id: string;
  tool_name: string;
  status: string;
  summary: string;
}

interface AsyncResultChipProps {
  reference: AsyncResultReference;
  onRemove: () => void;
  onViewFull: () => void;
}

/** Format tool name for display */
function formatToolName(name: string): string {
  return name
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}

export function AsyncResultChip({
  reference,
  onRemove,
  onViewFull,
}: AsyncResultChipProps) {
  return (
    <div className="inline-flex items-center gap-2 px-3 py-1.5 bg-indigo-500/10 border border-indigo-500/20 rounded-lg text-sm">
      {/* Icon */}
      <Paperclip className="h-3.5 w-3.5 text-indigo-400" />

      {/* Tool name and status */}
      <div className="flex items-center gap-1.5">
        <span className="text-indigo-300 font-medium">
          {formatToolName(reference.tool_name)} result
        </span>
        <CheckCircle2 className="h-3.5 w-3.5 text-emerald-400" />
      </div>

      {/* View full button */}
      <Button
        variant="ghost"
        size="sm"
        onClick={onViewFull}
        className="h-5 w-5 p-0 text-indigo-400 hover:text-indigo-300"
      >
        <ExternalLink className="h-3 w-3" />
      </Button>

      {/* Remove button */}
      <Button
        variant="ghost"
        size="sm"
        onClick={onRemove}
        className="h-5 w-5 p-0 text-slate-400 hover:text-slate-200"
      >
        <X className="h-3 w-3" />
      </Button>
    </div>
  );
}

/** Container for multiple result chips */
export function AsyncResultChipsContainer({
  references,
  onRemove,
  onViewFull,
}: {
  references: AsyncResultReference[];
  onRemove: (toolCallId: string) => void;
  onViewFull: (reference: AsyncResultReference) => void;
}) {
  if (references.length === 0) return null;

  return (
    <div className="flex flex-wrap gap-2 mb-2">
      {references.map((ref) => (
        <AsyncResultChip
          key={ref.tool_call_id}
          reference={ref}
          onRemove={() => onRemove(ref.tool_call_id)}
          onViewFull={() => onViewFull(ref)}
        />
      ))}
    </div>
  );
}

/** Summarize a result for display in the chip */
export function summarizeResult(result: unknown, maxLength = 100): string {
  if (result === null || result === undefined) {
    return "No result data";
  }

  if (typeof result === "string") {
    return result.length > maxLength ? result.slice(0, maxLength - 3) + "..." : result;
  }

  if (typeof result === "object") {
    const obj = result as Record<string, unknown>;

    // Try common summary fields
    if (typeof obj.message === "string") {
      return obj.message.length > maxLength
        ? obj.message.slice(0, maxLength - 3) + "..."
        : obj.message;
    }
    if (typeof obj.summary === "string") {
      return obj.summary.length > maxLength
        ? obj.summary.slice(0, maxLength - 3) + "..."
        : obj.summary;
    }

    // Count files if present
    if (Array.isArray(obj.files)) {
      return `Created ${obj.files.length} file${obj.files.length !== 1 ? "s" : ""}`;
    }
    if (typeof obj.files_created === "number") {
      return `Created ${obj.files_created} file${obj.files_created !== 1 ? "s" : ""}`;
    }

    // Default to JSON preview
    const jsonStr = JSON.stringify(result);
    return jsonStr.length > maxLength ? jsonStr.slice(0, maxLength - 3) + "..." : jsonStr;
  }

  return String(result).slice(0, maxLength);
}
