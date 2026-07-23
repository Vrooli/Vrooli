import { useRef, useState } from "react";
import { ExternalLink, X } from "lucide-react";
import { cn } from "../../lib/utils";
import { Popover } from "../ui/popover";
import { detailPathFromNodeId } from "../../app/routes/route-paths";
import type { AgentSessionContextRef, AgentSessionContextType } from "../../types";

export interface ComposerContextChip extends AgentSessionContextRef {
  title: string;
  subtitle?: string;
  /** Graph node id or route used to open the item's detail view. */
  nodeId?: string;
}

interface ContextChipTrayProps {
  items: ComposerContextChip[];
  onRemove?: (type: AgentSessionContextType, ref: string) => void;
  /** Navigate to a chip's detail; wired to the router by the composer host. */
  onOpen?: (path: string) => void;
  /** Sent-message chips should grow naturally instead of using composer overflow. */
  constrainHeight?: boolean;
  className?: string;
  testId?: string;
}

/** Resolve a chip's node id to a navigable route, or null if it has none. */
function chipOpenPath(nodeId?: string): string | null {
  if (!nodeId) return null;
  // Some context nodeIds are already app routes (e.g. "/sessions/x", "/operations").
  if (nodeId.startsWith("/")) return nodeId;
  return detailPathFromNodeId(nodeId);
}

export function ContextChipTray({ items, onRemove, onOpen, constrainHeight = true, className, testId }: ContextChipTrayProps) {
  const [openKey, setOpenKey] = useState<string | null>(null);
  const triggerRef = useRef<HTMLElement | null>(null);

  if (items.length === 0) return null;

  const openItem = items.find((item) => `${item.type}:${item.ref}` === openKey) ?? null;
  const openPath = chipOpenPath(openItem?.nodeId);

  return (
    <div className={cn("flex flex-wrap gap-1.5", constrainHeight && "max-h-20 overflow-y-auto", className)} data-testid={testId}>
      {items.map((item) => {
        const key = `${item.type}:${item.ref}`;
        return (
          <span
            key={key}
            className="inline-flex max-w-full items-center gap-1 rounded border border-cyan-500/25 bg-cyan-500/10 px-2 py-1 text-xs text-cyan-100"
          >
            <button
              type="button"
              className="flex min-w-0 items-center rounded text-left hover:text-white focus:outline-none focus-visible:ring-1 focus-visible:ring-cyan-400"
              aria-haspopup="dialog"
              aria-expanded={openKey === key}
              onClick={(event) => {
                triggerRef.current = event.currentTarget;
                setOpenKey((current) => (current === key ? null : key));
              }}
              data-testid={testId ? `${testId}-chip` : undefined}
            >
              <span className="min-w-0 truncate">
                <span className="text-cyan-300">{labelForContextType(item.type)}</span> {item.title}
              </span>
            </button>
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
        );
      })}

      {openItem && (
        <Popover
          isOpen
          onClose={() => setOpenKey(null)}
          triggerRef={triggerRef}
          placement="top-start"
          className="w-64 p-3"
          testId={testId ? `${testId}-detail` : undefined}
        >
          <p className="text-[11px] font-medium uppercase tracking-wide text-cyan-300">
            {labelForContextType(openItem.type)}
          </p>
          <p className="mt-0.5 break-words text-sm font-medium text-slate-100">{openItem.title}</p>
          {openItem.subtitle && (
            <p className="mt-1 break-words text-xs text-slate-400">{openItem.subtitle}</p>
          )}
          <p className="mt-1 break-all text-[11px] text-slate-600">{openItem.ref}</p>

          <div className="mt-3 flex flex-col gap-1">
            {openPath && onOpen && (
              <button
                type="button"
                className="flex items-center gap-2 rounded px-2 py-1 text-left text-xs text-slate-200 hover:bg-slate-800"
                onClick={() => {
                  onOpen(openPath);
                  setOpenKey(null);
                }}
                data-testid={testId ? `${testId}-detail-open` : undefined}
              >
                <ExternalLink className="h-3.5 w-3.5 text-cyan-400" />
                Open details
              </button>
            )}
            {onRemove && (
              <button
                type="button"
                className="flex items-center gap-2 rounded px-2 py-1 text-left text-xs text-red-300 hover:bg-red-500/10"
                onClick={() => {
                  onRemove(openItem.type, openItem.ref);
                  setOpenKey(null);
                }}
              >
                <X className="h-3.5 w-3.5" />
                Remove from context
              </button>
            )}
          </div>
        </Popover>
      )}
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
      return "Archived workflow";
    case "operations_briefing":
      return "Briefing";
    case "startup_brief":
      return "Startup";
    default:
      return type.replace(/_/g, " ");
  }
}
