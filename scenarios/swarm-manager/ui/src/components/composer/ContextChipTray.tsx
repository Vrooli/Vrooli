import { X } from "lucide-react";
import { cn } from "../../lib/utils";
import type { AgentSessionContextRef, AgentSessionContextType } from "../../types";

export interface ComposerContextChip extends AgentSessionContextRef {
  title: string;
  subtitle?: string;
}

interface ContextChipTrayProps {
  items: ComposerContextChip[];
  onRemove?: (type: AgentSessionContextType, ref: string) => void;
  className?: string;
  testId?: string;
}

export function ContextChipTray({ items, onRemove, className, testId }: ContextChipTrayProps) {
  if (items.length === 0) return null;

  return (
    <div className={cn("flex max-h-20 flex-wrap gap-1.5 overflow-y-auto", className)} data-testid={testId}>
      {items.map((item) => (
        <span
          key={`${item.type}:${item.ref}`}
          className="inline-flex max-w-full items-center gap-1 rounded border border-cyan-500/25 bg-cyan-500/10 px-2 py-1 text-xs text-cyan-100"
          title={item.subtitle ?? item.ref}
        >
          <span className="min-w-0 truncate">
            <span className="text-cyan-300">{labelForContextType(item.type)}</span> {item.title}
          </span>
          {onRemove && (
            <button
              type="button"
              className="shrink-0 rounded text-cyan-200 hover:bg-cyan-400/15 hover:text-white"
              onClick={() => onRemove(item.type, item.ref)}
              aria-label={`Remove ${item.title}`}
            >
              <X className="h-3 w-3" />
            </button>
          )}
        </span>
      ))}
    </div>
  );
}

function labelForContextType(type: AgentSessionContextType): string {
  switch (type) {
    case "backlog_item":
      return "Item";
    case "agent_activity":
      return "Activity";
    case "operating_mode":
      return "Mode";
    default:
      return type.replace(/_/g, " ");
  }
}
