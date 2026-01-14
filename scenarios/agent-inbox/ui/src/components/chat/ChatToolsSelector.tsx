/**
 * ChatToolsSelector Component
 *
 * A compact dropdown for configuring tools on a per-chat basis.
 * Shows the number of enabled tools and allows quick toggling.
 * Includes a master toggle to enable/disable all tools for the chat.
 *
 * ARCHITECTURE:
 * - Lightweight wrapper around ToolConfiguration for chat context
 * - Uses popover pattern for non-modal interaction
 * - Integrates with useTools hook for state management
 * - Supports manual tool execution via ManualToolDialog
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
import { ManualToolDialog } from "../tools/ManualToolDialog";
import { useTools } from "../../hooks/useTools";
import { useQueryClient } from "@tanstack/react-query";
import type { EffectiveTool, Chat } from "../../lib/api";
import { updateChat } from "../../lib/api";

interface ChatToolsSelectorProps {
  /** Chat ID for chat-specific configuration */
  chatId: string;
  /** Whether tools are enabled for this chat (from chat settings) */
  toolsEnabled?: boolean;
}

export function ChatToolsSelector({ chatId, toolsEnabled = true }: ChatToolsSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [selectedTool, setSelectedTool] = useState<EffectiveTool | null>(null);
  const [isTogglingMaster, setIsTogglingMaster] = useState(false);
  const popoverRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLDivElement>(null);
  const queryClient = useQueryClient();

  const {
    toolsByScenario,
    toolSet,
    scenarios,
    enabledTools,
    isLoading,
    isSyncing,
    isUpdating,
    error,
    toggleTool,
    resetTool,
    syncDiscoveredTools,
  } = useTools({ chatId });

  // Handle master toggle for enabling/disabling all tools
  const handleMasterToggle = async (enabled: boolean) => {
    setIsTogglingMaster(true);
    try {
      await updateChat(chatId, { tools_enabled: enabled });
      // Invalidate chat queries to refresh the state
      queryClient.invalidateQueries({ queryKey: ["chat", chatId] });
      queryClient.invalidateQueries({ queryKey: ["chats"] });
    } catch (err) {
      console.error("Failed to toggle tools:", err);
    } finally {
      setIsTogglingMaster(false);
    }
  };

  // Handle running a tool manually
  const handleRunTool = (tool: EffectiveTool) => {
    setSelectedTool(tool);
  };

  // Handle successful tool execution
  const handleToolSuccess = () => {
    // Refresh chat messages to show the new tool call
    queryClient.invalidateQueries({ queryKey: ["chat", chatId] });
  };

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

          {/* Master toggle */}
          <div className="p-3 border-b border-white/10">
            <div className="flex items-center justify-between">
              <div>
                <span className="text-sm font-medium text-white">Enable Tools</span>
                <p className="text-xs text-slate-500">Allow AI to use tools in this chat</p>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  checked={toolsEnabled}
                  onChange={(e) => handleMasterToggle(e.target.checked)}
                  disabled={isTogglingMaster}
                  className="sr-only peer"
                  data-testid="tools-master-toggle"
                />
                <div className="w-11 h-6 bg-slate-700 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-indigo-500/50 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[3px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-indigo-500 peer-disabled:opacity-50" />
              </label>
            </div>
          </div>

          {/* Content */}
          <div className="p-3">
            {toolsEnabled ? (
              <ToolConfiguration
                toolsByScenario={toolsByScenario}
                categories={toolSet?.categories ?? []}
                scenarioStatuses={scenarios}
                chatId={chatId}
                isLoading={isLoading}
                isSyncing={isSyncing}
                isUpdating={isUpdating}
                error={error?.message}
                onToggleTool={toggleTool}
                onResetTool={resetTool}
                onSyncTools={syncDiscoveredTools}
                onRunTool={handleRunTool}
              />
            ) : (
              <div className="text-center py-6">
                <p className="text-sm text-slate-400">
                  Tools are disabled for this chat.
                </p>
                <p className="text-xs text-slate-500 mt-1">
                  Enable tools above to configure them.
                </p>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Manual tool execution dialog */}
      {selectedTool && (
        <ManualToolDialog
          open={!!selectedTool}
          onClose={() => setSelectedTool(null)}
          tool={selectedTool}
          chatId={chatId}
          onSuccess={handleToolSuccess}
        />
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
