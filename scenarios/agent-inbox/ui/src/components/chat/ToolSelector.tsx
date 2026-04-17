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
} from "lucide-react";
import { Dialog, DialogHeader, DialogBody, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import type { EffectiveTool } from "@/lib/api";
import { ToolOptionButton, type FlatTool } from "./ToolOptionButton";

interface ToolSelectorProps {
  open: boolean;
  onClose: () => void;
  toolsByScenario: Map<string, EffectiveTool[]>;
  forcedTool?: { scenario: string; toolName: string } | null;
  onSelect: (scenario: string, toolName: string) => void;
  onClear: () => void;
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
  const [focusedIndex, setFocusedIndex] = useState(-1);
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
        tool.tool.description.toLowerCase().includes(query) ||
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

  useEffect(() => { setFocusedIndex(-1); }, [searchQuery]);
  useEffect(() => {
    if (open) { setFocusedIndex(-1); setSearchQuery(""); }
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
          setFocusedIndex((prev) => prev < maxIndex ? prev + 1 : -1);
          break;
        case "ArrowUp":
          e.preventDefault();
          setFocusedIndex((prev) => prev > -1 ? prev - 1 : maxIndex);
          break;
        case "Tab":
          if (!e.shiftKey && focusedIndex === -1) {
            if (indexedFilteredTools.length > 0) { e.preventDefault(); setFocusedIndex(0); }
          } else if (e.shiftKey && focusedIndex === 0) {
            e.preventDefault(); setFocusedIndex(-1);
          } else if (!e.shiftKey && focusedIndex === maxIndex) {
            e.preventDefault();
            if (forcedTool) { clearButtonRef.current?.focus(); }
          } else if (focusedIndex >= 0) {
            e.preventDefault();
            setFocusedIndex((prev) => e.shiftKey ? prev - 1 : prev + 1);
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

  useEffect(() => {
    if (!open) return;
    if (focusedIndex === -1) {
      searchInputRef.current?.focus();
    } else if (toolRefs.current[focusedIndex]) {
      toolRefs.current[focusedIndex].focus();
      toolRefs.current[focusedIndex].scrollIntoView({ block: "nearest", behavior: "smooth" });
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
            <p className="text-xs text-slate-500">
              Use <kbd className="px-1 py-0.5 rounded bg-slate-700">&#8593;</kbd>{" "}
              <kbd className="px-1 py-0.5 rounded bg-slate-700">&#8595;</kbd> to navigate,{" "}
              <kbd className="px-1 py-0.5 rounded bg-slate-700">Enter</kbd> to select
            </p>
            <div className="space-y-4" role="listbox">
              {indexedFilteredTools.length === 0 ? (
                <div className="text-center py-8 text-slate-400">No tools found</div>
              ) : (
                Array.from(filteredByScenario.entries()).map(
                  ([scenario, scenarioTools]) => (
                    <div key={scenario}>
                      <h3 className="text-xs font-medium text-slate-500 uppercase tracking-wider mb-2 flex items-center gap-2">
                        <Wrench className="h-3 w-3" />
                        {scenario}
                      </h3>
                      <div className="space-y-2">
                        {scenarioTools.map(({ tool, index }) => (
                          <ToolOptionButton
                            key={`${scenario}-${tool.tool.name}`}
                            scenario={scenario}
                            tool={tool}
                            index={index}
                            isSelected={forcedTool?.scenario === scenario && forcedTool.toolName === tool.tool.name}
                            isFocused={focusedIndex === index}
                            onSelect={handleSelect}
                            onFocus={setFocusedIndex}
                            buttonRef={(el) => { toolRefs.current[index] = el; }}
                          />
                        ))}
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
          <Button ref={clearButtonRef} variant="ghost" onClick={handleClear} className="text-red-400 hover:text-red-300">
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
