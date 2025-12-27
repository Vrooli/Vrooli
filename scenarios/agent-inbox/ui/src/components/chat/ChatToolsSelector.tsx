/**
 * ChatToolsSelector Component
 *
 * A compact dropdown for configuring tools on a per-chat basis.
 * Shows the number of enabled tools and allows quick toggling.
 *
 * ARCHITECTURE:
 * - Lightweight wrapper around ToolConfiguration for chat context
 * - Uses popover pattern for non-modal interaction
 * - Integrates with useTools hook for state management
 *
 * TESTING SEAMS:
 * - All state managed via useTools hook
 * - Test IDs on interactive elements
 */

import { useState, useRef, useEffect } from "react";
import { Wrench, X } from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import { ToolConfiguration } from "../settings/ToolConfiguration";
import { useTools } from "../../hooks/useTools";

interface ChatToolsSelectorProps {
  /** Chat ID for chat-specific configuration */
  chatId: string;
}

export function ChatToolsSelector({ chatId }: ChatToolsSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const popoverRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLDivElement>(null);

  const {
    toolsByScenario,
    toolSet,
    scenarios,
    enabledTools,
    isLoading,
    isRefreshing,
    isUpdating,
    error,
    toggleTool,
    resetTool,
    refreshToolRegistry,
  } = useTools({ chatId });

  // Close popover when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        popoverRef.current &&
        !popoverRef.current.contains(event.target as Node) &&
        triggerRef.current &&
        !triggerRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    }

    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
      return () => document.removeEventListener("mousedown", handleClickOutside);
    }
  }, [isOpen]);

  // Close on escape
  useEffect(() => {
    function handleEscape(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setIsOpen(false);
      }
    }

    if (isOpen) {
      document.addEventListener("keydown", handleEscape);
      return () => document.removeEventListener("keydown", handleEscape);
    }
  }, [isOpen]);

  const enabledCount = enabledTools.length;
  const totalCount = toolSet?.tools.length ?? 0;

  // Check if any tools have chat-specific overrides
  const hasOverrides = enabledTools.some((t) => t.source === "chat");

  return (
    <div className="relative" ref={triggerRef}>
      {/* Trigger button */}
      <Tooltip content={`${enabledCount} tools enabled`}>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setIsOpen(!isOpen)}
          className={`gap-1.5 ${hasOverrides ? "text-indigo-400" : ""}`}
          data-testid="chat-tools-trigger"
        >
          <Wrench className="h-4 w-4" />
          <span className="text-xs">
            {enabledCount}/{totalCount}
          </span>
        </Button>
      </Tooltip>

      {/* Popover */}
      {isOpen && (
        <div
          ref={popoverRef}
          className="absolute right-0 top-full mt-2 z-50 w-96 max-h-[70vh] overflow-auto rounded-lg border border-white/10 bg-slate-900 shadow-xl"
          data-testid="chat-tools-popover"
        >
          {/* Header */}
          <div className="sticky top-0 z-10 flex items-center justify-between p-3 border-b border-white/10 bg-slate-900">
            <h3 className="text-sm font-medium text-white">Chat Tools</h3>
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setIsOpen(false)}
              className="h-6 w-6"
              data-testid="chat-tools-close"
            >
              <X className="h-4 w-4" />
            </Button>
          </div>

          {/* Content */}
          <div className="p-3">
            <ToolConfiguration
              toolsByScenario={toolsByScenario}
              categories={toolSet?.categories ?? []}
              scenarioStatuses={scenarios}
              chatId={chatId}
              isLoading={isLoading}
              isRefreshing={isRefreshing}
              isUpdating={isUpdating}
              error={error?.message}
              onToggleTool={toggleTool}
              onResetTool={resetTool}
              onRefresh={refreshToolRegistry}
            />
          </div>
        </div>
      )}
    </div>
  );
}

/**
 * Compact tool indicator that just shows the count.
 * Use this when space is limited (e.g., mobile header).
 */
export function ToolCountBadge({ chatId }: { chatId: string }) {
  const { enabledTools, toolSet, isLoading } = useTools({ chatId });

  if (isLoading) {
    return null;
  }

  const enabledCount = enabledTools.length;
  const totalCount = toolSet?.tools.length ?? 0;

  if (totalCount === 0) {
    return null;
  }

  return (
    <span
      className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded bg-white/10 text-xs text-slate-400"
      data-testid="tool-count-badge"
    >
      <Wrench className="h-3 w-3" />
      {enabledCount}
    </span>
  );
}
