/**
 * ToolSelector - Modal for browsing and selecting a tool to force.
 *
 * Displays enabled tools grouped by scenario with search functionality.
 * Fully keyboard navigable with arrows, tab, and enter.
 */
import { useState, useMemo, useCallback, useRef, useEffect } from "react";
import {
  Wrench,
  Search,
  Check,
} from "lucide-react";
import { Dialog, DialogHeader, DialogBody, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import type { EffectiveTool } from "@/lib/api";

interface ToolSelectorProps {
  open: boolean;
  onClose: () => void;
  /** Enabled tools grouped by scenario */
  toolsByScenario: Map<string, EffectiveTool[]>;
  /** Currently forced tool, if any */
  forcedTool?: { scenario: string; toolName: string } | null;
  /** Callback when a tool is selected */
  onSelect: (scenario: string, toolName: string) => void;
  /** Callback to clear forced tool */
  onClear: () => void;
}

interface FlatTool {
  scenario: string;
  tool: EffectiveTool;
  index: number;
}

export function ToolSelector({
  open,
  onClose,
  toolsByScenario,
  forcedTool,
  onSelect,
  onClear,
}: ToolSelectorProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [focusedIndex, setFocusedIndex] = useState(-1); // -1 = search focused
  const searchInputRef = useRef<HTMLInputElement>(null);
  const toolRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const clearButtonRef = useRef<HTMLButtonElement>(null);

  // Flatten tools for filtering and indexing
  const allTools = useMemo(() => {
    const result: FlatTool[] = [];
    let index = 0;
    for (const [scenario, tools] of toolsByScenario.entries()) {
      for (const tool of tools) {
        result.push({ scenario, tool, index: index++ });
      }
    }
    return result;
  }, [toolsByScenario]);

  // Filter tools by search query
  const filteredTools = useMemo(() => {
    if (!searchQuery.trim()) return allTools;

    const query = searchQuery.toLowerCase();
    return allTools.filter(
      ({ scenario, tool }) =>
        tool.tool.name.toLowerCase().includes(query) ||
        tool.tool.description?.toLowerCase().includes(query) ||
        scenario.toLowerCase().includes(query) ||
        tool.tool.category?.toLowerCase().includes(query)
    );
  }, [allTools, searchQuery]);

  // Re-index filtered tools for navigation
  const indexedFilteredTools = useMemo(() => {
    return filteredTools.map((t, idx) => ({ ...t, index: idx }));
  }, [filteredTools]);

  // Group filtered tools by scenario for display
  const filteredByScenario = useMemo(() => {
    const grouped = new Map<string, FlatTool[]>();
    for (const tool of indexedFilteredTools) {
      const existing = grouped.get(tool.scenario) ?? [];
      grouped.set(tool.scenario, [...existing, tool]);
    }
    return grouped;
  }, [indexedFilteredTools]);

  // Reset focus when search results change
  useEffect(() => {
    setFocusedIndex(-1);
  }, [searchQuery]);

  // Reset state when modal opens
  useEffect(() => {
    if (open) {
      setFocusedIndex(-1);
      setSearchQuery("");
    }
  }, [open]);

  const handleSelect = useCallback((scenario: string, toolName: string) => {
    onSelect(scenario, toolName);
    onClose();
    setSearchQuery("");
  }, [onSelect, onClose]);

  const handleClose = useCallback(() => {
    onClose();
    setSearchQuery("");
  }, [onClose]);

  const handleClear = useCallback(() => {
    onClear();
    onClose();
    setSearchQuery("");
  }, [onClear, onClose]);

  // Handle keyboard navigation
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      const maxIndex = indexedFilteredTools.length - 1;

      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          setFocusedIndex((prev) => {
            const next = prev < maxIndex ? prev + 1 : -1;
            return next;
          });
          break;
        case "ArrowUp":
          e.preventDefault();
          setFocusedIndex((prev) => {
            const next = prev > -1 ? prev - 1 : maxIndex;
            return next;
          });
          break;
        case "Tab":
          if (!e.shiftKey && focusedIndex === -1) {
            // Going from search to first item
            if (indexedFilteredTools.length > 0) {
              e.preventDefault();
              setFocusedIndex(0);
            }
          } else if (e.shiftKey && focusedIndex === 0) {
            // Going from first item back to search
            e.preventDefault();
            setFocusedIndex(-1);
          } else if (!e.shiftKey && focusedIndex === maxIndex) {
            // At last item, focus clear/close button
            e.preventDefault();
            if (forcedTool) {
              clearButtonRef.current?.focus();
            }
          } else if (focusedIndex >= 0) {
            // Normal tab through items
            e.preventDefault();
            setFocusedIndex((prev) =>
              e.shiftKey ? prev - 1 : prev + 1
            );
          }
          break;
        case "Enter":
          if (focusedIndex >= 0 && indexedFilteredTools[focusedIndex]) {
            e.preventDefault();
            const { scenario, tool } = indexedFilteredTools[focusedIndex];
            handleSelect(scenario, tool.tool.name);
          }
          break;
      }
    },
    [indexedFilteredTools, focusedIndex, handleSelect, forcedTool]
  );

  // Focus the appropriate element when focusedIndex changes
  useEffect(() => {
    if (!open) return;

    if (focusedIndex === -1) {
      searchInputRef.current?.focus();
    } else if (toolRefs.current[focusedIndex]) {
      toolRefs.current[focusedIndex]?.focus();
      toolRefs.current[focusedIndex]?.scrollIntoView({
        block: "nearest",
        behavior: "smooth",
      });
    }
  }, [focusedIndex, open]);

  const hasTools = allTools.length > 0;

  return (
    <Dialog open={open} onClose={handleClose} className="max-w-lg">
      <DialogHeader onClose={handleClose}>
        Force Tool
        {forcedTool && (
          <span className="ml-2 text-sm font-normal text-violet-400">
            ({forcedTool.toolName})
          </span>
        )}
      </DialogHeader>
      <DialogBody className="space-y-4" onKeyDown={handleKeyDown}>
        {!hasTools ? (
          <div className="text-center py-8 text-slate-400">
            <Wrench className="h-8 w-8 mx-auto mb-3 opacity-50" />
            <p>No enabled tools available</p>
            <p className="text-sm mt-1">Enable tools in the chat settings</p>
          </div>
        ) : (
          <>
            {/* Search input */}
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
              <input
                ref={searchInputRef}
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Search tools..."
                className="w-full pl-10 pr-4 py-2 bg-slate-800 border border-white/10 rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-violet-500/50 focus:border-violet-500/50"
                data-testid="tool-search-input"
                autoFocus
              />
            </div>

            {/* Keyboard hint */}
            <p className="text-xs text-slate-500">
              Use <kbd className="px-1 py-0.5 rounded bg-slate-700">↑</kbd>{" "}
              <kbd className="px-1 py-0.5 rounded bg-slate-700">↓</kbd> to navigate,{" "}
              <kbd className="px-1 py-0.5 rounded bg-slate-700">Enter</kbd> to select
            </p>

            {/* Tools list */}
            <div className="space-y-4" role="listbox">
              {indexedFilteredTools.length === 0 ? (
                <div className="text-center py-8 text-slate-400">
                  No tools found
                </div>
              ) : (
                Array.from(filteredByScenario.entries()).map(
                  ([scenario, scenarioTools]) => (
                    <div key={scenario}>
                      <h3 className="text-xs font-medium text-slate-500 uppercase tracking-wider mb-2 flex items-center gap-2">
                        <Wrench className="h-3 w-3" />
                        {scenario}
                      </h3>
                      <div className="space-y-2">
                        {scenarioTools.map(({ tool, index }) => {
                          const isSelected =
                            forcedTool?.scenario === scenario &&
                            forcedTool?.toolName === tool.tool.name;
                          const isFocused = focusedIndex === index;

                          return (
                            <button
                              key={`${scenario}-${tool.tool.name}`}
                              ref={(el) => {
                                toolRefs.current[index] = el;
                              }}
                              onClick={() => handleSelect(scenario, tool.tool.name)}
                              onFocus={() => setFocusedIndex(index)}
                              role="option"
                              aria-selected={isSelected}
                              tabIndex={isFocused ? 0 : -1}
                              className={`
                                w-full flex items-start gap-3 p-3 rounded-lg border text-left transition-colors
                                ${
                                  isSelected
                                    ? "bg-violet-500/20 border-violet-500/50"
                                    : isFocused
                                      ? "bg-slate-700/50 border-violet-400/50 ring-2 ring-violet-500/30"
                                      : "bg-slate-800/50 border-white/10 hover:bg-slate-800 hover:border-white/20"
                                }
                              `}
                              data-testid={`tool-option-${tool.tool.name}`}
                            >
                              <div
                                className={`
                                flex-shrink-0 p-1.5 rounded-lg
                                ${isSelected ? "bg-violet-500/30" : "bg-slate-700"}
                              `}
                              >
                                <Wrench
                                  className={`h-4 w-4 ${isSelected ? "text-violet-400" : "text-slate-300"}`}
                                />
                              </div>
                              <div className="flex-1 min-w-0">
                                <div className="flex items-center gap-2">
                                  <span
                                    className={`font-medium text-sm ${isSelected ? "text-violet-300" : "text-white"}`}
                                  >
                                    {tool.tool.name}
                                  </span>
                                  {tool.tool.category && (
                                    <span className="text-xs px-1.5 py-0.5 rounded bg-slate-700 text-slate-400">
                                      {tool.tool.category}
                                    </span>
                                  )}
                                </div>
                                {tool.tool.description && (
                                  <p className="text-xs text-slate-400 mt-1 line-clamp-2">
                                    {tool.tool.description}
                                  </p>
                                )}
                              </div>
                              {isSelected && (
                                <Check className="h-4 w-4 text-violet-400 shrink-0" />
                              )}
                            </button>
                          );
                        })}
                      </div>
                    </div>
                  )
                )
              )}
            </div>
          </>
        )}
      </DialogBody>
      <DialogFooter>
        {forcedTool && (
          <Button
            ref={clearButtonRef}
            variant="ghost"
            onClick={handleClear}
            className="text-red-400 hover:text-red-300"
          >
            Clear forced tool
          </Button>
        )}
        <Button variant="ghost" onClick={handleClose}>
          {forcedTool ? "Done" : "Cancel"}
        </Button>
      </DialogFooter>
    </Dialog>
  );
}
