/**
 * TemplateSelector - Modal for browsing and selecting message templates.
 *
 * Displays templates in a grid layout with search functionality.
 * Fully keyboard navigable with arrows, tab, and enter.
 */
import { useState, useMemo, useCallback, useRef, useEffect } from "react";
import {
  FileText,
  Sparkles,
  Code,
  Search,
  Construction,
  type LucideIcon,
} from "lucide-react";
import { Dialog, DialogHeader, DialogBody } from "@/components/ui/dialog";
import type { Template } from "@/lib/types/templates";

// Map icon names to Lucide components
const ICON_MAP: Record<string, LucideIcon> = {
  Sparkles,
  Code,
  FileText,
};

interface TemplateSelectorProps {
  open: boolean;
  onClose: () => void;
  templates: Template[];
  onSelect: (template: Template) => void;
  activeTemplateId?: string;
}

export function TemplateSelector({
  open,
  onClose,
  templates,
  onSelect,
  activeTemplateId,
}: TemplateSelectorProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [focusedIndex, setFocusedIndex] = useState(-1); // -1 = search focused
  const searchInputRef = useRef<HTMLInputElement>(null);
  const templateRefs = useRef<(HTMLButtonElement | null)[]>([]);

  // Filter templates by search query
  const filteredTemplates = useMemo(() => {
    if (!searchQuery.trim()) return templates;

    const query = searchQuery.toLowerCase();
    return templates.filter(
      (t) =>
        t.name.toLowerCase().includes(query) ||
        t.description.toLowerCase().includes(query) ||
        t.category?.toLowerCase().includes(query)
    );
  }, [templates, searchQuery]);

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

  const handleSelect = useCallback((template: Template) => {
    onSelect(template);
    onClose();
    setSearchQuery("");
  }, [onSelect, onClose]);

  const handleClose = useCallback(() => {
    onClose();
    setSearchQuery("");
  }, [onClose]);

  // Handle keyboard navigation
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      const maxIndex = filteredTemplates.length - 1;

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
          // Let tab work naturally between search and items
          if (!e.shiftKey && focusedIndex === -1) {
            // Going from search to first item
            if (filteredTemplates.length > 0) {
              e.preventDefault();
              setFocusedIndex(0);
            }
          } else if (e.shiftKey && focusedIndex === 0) {
            // Going from first item back to search
            e.preventDefault();
            setFocusedIndex(-1);
          } else if (!e.shiftKey && focusedIndex === maxIndex) {
            // At last item, wrap to search
            e.preventDefault();
            setFocusedIndex(-1);
          } else if (e.shiftKey && focusedIndex === -1) {
            // At search going back, let it close dialog or wrap
            e.preventDefault();
            setFocusedIndex(maxIndex);
          } else if (focusedIndex >= 0) {
            // Normal tab through items
            e.preventDefault();
            setFocusedIndex((prev) =>
              e.shiftKey ? prev - 1 : prev + 1
            );
          }
          break;
        case "Enter":
          if (focusedIndex >= 0 && filteredTemplates[focusedIndex]) {
            e.preventDefault();
            handleSelect(filteredTemplates[focusedIndex]);
          }
          break;
      }
    },
    [filteredTemplates, focusedIndex, handleSelect]
  );

  // Focus the appropriate element when focusedIndex changes
  useEffect(() => {
    if (!open) return;

    if (focusedIndex === -1) {
      searchInputRef.current?.focus();
    } else if (templateRefs.current[focusedIndex]) {
      templateRefs.current[focusedIndex]?.focus();
      templateRefs.current[focusedIndex]?.scrollIntoView({
        block: "nearest",
        behavior: "smooth",
      });
    }
  }, [focusedIndex, open]);

  return (
    <Dialog open={open} onClose={handleClose} className="max-w-lg">
      <DialogHeader onClose={handleClose}>Select Template</DialogHeader>
      <DialogBody className="space-y-4" onKeyDown={handleKeyDown}>
        {/* Search input */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
          <input
            ref={searchInputRef}
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search templates..."
            className="w-full pl-10 pr-4 py-2 bg-slate-800 border border-white/10 rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500/50"
            data-testid="template-search-input"
            autoFocus
          />
        </div>

        {/* Keyboard hint */}
        <p className="text-xs text-slate-500">
          Use <kbd className="px-1 py-0.5 rounded bg-slate-700">↑</kbd>{" "}
          <kbd className="px-1 py-0.5 rounded bg-slate-700">↓</kbd> to navigate,{" "}
          <kbd className="px-1 py-0.5 rounded bg-slate-700">Enter</kbd> to select
        </p>

        {/* Templates grid */}
        <div className="grid grid-cols-1 gap-3" role="listbox">
          {filteredTemplates.length === 0 ? (
            <div className="text-center py-8 text-slate-400">
              No templates found
            </div>
          ) : (
            filteredTemplates.map((template, index) => {
              const IconComponent = template.icon
                ? ICON_MAP[template.icon] || FileText
                : FileText;
              const isActive = template.id === activeTemplateId;
              const isFocused = focusedIndex === index;

              return (
                <button
                  key={template.id}
                  ref={(el) => {
                    templateRefs.current[index] = el;
                  }}
                  onClick={() => handleSelect(template)}
                  onFocus={() => setFocusedIndex(index)}
                  role="option"
                  aria-selected={isActive}
                  tabIndex={isFocused ? 0 : -1}
                  className={`
                    flex items-start gap-3 p-4 rounded-lg border text-left transition-colors
                    ${
                      isActive
                        ? "bg-blue-500/20 border-blue-500/50"
                        : isFocused
                          ? "bg-slate-700/50 border-blue-400/50 ring-2 ring-blue-500/30"
                          : "bg-slate-800/50 border-white/10 hover:bg-slate-800 hover:border-white/20"
                    }
                  `}
                  data-testid={`template-option-${template.id}`}
                >
                  <div
                    className={`
                    flex-shrink-0 p-2 rounded-lg
                    ${isActive ? "bg-blue-500/30" : "bg-slate-700"}
                  `}
                  >
                    <IconComponent
                      className={`h-5 w-5 ${isActive ? "text-blue-400" : "text-slate-300"}`}
                    />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span
                        className={`font-medium ${isActive ? "text-blue-300" : "text-white"}`}
                      >
                        {template.name}
                      </span>
                      {template.category && (
                        <span className="text-xs px-2 py-0.5 rounded-full bg-slate-700 text-slate-400">
                          {template.category}
                        </span>
                      )}
                      {template.draft && (
                        <span
                          className="flex items-center gap-1 px-1.5 py-0.5 text-[10px] rounded bg-orange-900/50 text-orange-400 border border-orange-500/30"
                          title="This template is a draft and may not be fully working"
                        >
                          <Construction className="h-2.5 w-2.5" />
                          Draft
                        </span>
                      )}
                    </div>
                    <p className="text-sm text-slate-400 mt-1 line-clamp-2">
                      {template.description}
                    </p>
                    {template.variables.length > 0 && (
                      <p className="text-xs text-slate-500 mt-2">
                        {template.variables.length} variable
                        {template.variables.length !== 1 ? "s" : ""} to fill
                      </p>
                    )}
                  </div>
                </button>
              );
            })
          )}
        </div>
      </DialogBody>
    </Dialog>
  );
}
