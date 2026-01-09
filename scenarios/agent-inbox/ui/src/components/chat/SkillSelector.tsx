/**
 * SkillSelector - Modal for browsing and selecting skills to attach.
 *
 * Displays skills as a checklist with search functionality.
 * Fully keyboard navigable with arrows, tab, and enter/space.
 */
import { useState, useMemo, useCallback, useRef, useEffect } from "react";
import {
  BookOpen,
  Building2,
  TestTube,
  Search,
  Check,
  type LucideIcon,
} from "lucide-react";
import { Dialog, DialogHeader, DialogBody, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import type { Skill } from "@/lib/types/templates";

// Map icon names to Lucide components
const ICON_MAP: Record<string, LucideIcon> = {
  Building2,
  TestTube,
  BookOpen,
};

interface SkillSelectorProps {
  open: boolean;
  onClose: () => void;
  skills: Skill[];
  selectedSkillIds: string[];
  onToggle: (skillId: string) => void;
}

export function SkillSelector({
  open,
  onClose,
  skills,
  selectedSkillIds,
  onToggle,
}: SkillSelectorProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [focusedIndex, setFocusedIndex] = useState(-1); // -1 = search focused
  const searchInputRef = useRef<HTMLInputElement>(null);
  const skillRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const doneButtonRef = useRef<HTMLButtonElement>(null);

  // Filter skills by search query
  const filteredSkills = useMemo(() => {
    if (!searchQuery.trim()) return skills;

    const query = searchQuery.toLowerCase();
    return skills.filter(
      (s) =>
        s.name.toLowerCase().includes(query) ||
        s.description.toLowerCase().includes(query) ||
        s.category?.toLowerCase().includes(query) ||
        s.tags?.some((t) => t.toLowerCase().includes(query))
    );
  }, [skills, searchQuery]);

  // Group skills by category
  const skillsByCategory = useMemo(() => {
    const grouped = new Map<string, Skill[]>();

    for (const skill of filteredSkills) {
      const category = skill.category || "Other";
      if (!grouped.has(category)) {
        grouped.set(category, []);
      }
      grouped.get(category)!.push(skill);
    }

    return grouped;
  }, [filteredSkills]);

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

  const handleClose = useCallback(() => {
    onClose();
    setSearchQuery("");
  }, [onClose]);

  // Handle keyboard navigation
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      const maxIndex = filteredSkills.length - 1;

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
            if (filteredSkills.length > 0) {
              e.preventDefault();
              setFocusedIndex(0);
            }
          } else if (e.shiftKey && focusedIndex === 0) {
            // Going from first item back to search
            e.preventDefault();
            setFocusedIndex(-1);
          } else if (!e.shiftKey && focusedIndex === maxIndex) {
            // At last item, focus done button
            e.preventDefault();
            doneButtonRef.current?.focus();
          } else if (focusedIndex >= 0) {
            // Normal tab through items
            e.preventDefault();
            setFocusedIndex((prev) =>
              e.shiftKey ? prev - 1 : prev + 1
            );
          }
          break;
        case "Enter":
        case " ":
          if (focusedIndex >= 0 && filteredSkills[focusedIndex]) {
            e.preventDefault();
            onToggle(filteredSkills[focusedIndex].id);
          }
          break;
      }
    },
    [filteredSkills, focusedIndex, onToggle]
  );

  // Focus the appropriate element when focusedIndex changes
  useEffect(() => {
    if (!open) return;

    if (focusedIndex === -1) {
      searchInputRef.current?.focus();
    } else if (skillRefs.current[focusedIndex]) {
      skillRefs.current[focusedIndex]?.focus();
      skillRefs.current[focusedIndex]?.scrollIntoView({
        block: "nearest",
        behavior: "smooth",
      });
    }
  }, [focusedIndex, open]);

  const selectedCount = selectedSkillIds.length;

  // Build a flat list of skills with their indices for ref tracking
  const skillsWithIndices = useMemo(() => {
    let index = 0;
    const result: { skill: Skill; index: number; category: string }[] = [];
    for (const [category, categorySkills] of skillsByCategory.entries()) {
      for (const skill of categorySkills) {
        result.push({ skill, index: index++, category });
      }
    }
    return result;
  }, [skillsByCategory]);

  return (
    <Dialog open={open} onClose={handleClose} className="max-w-lg">
      <DialogHeader onClose={handleClose}>
        Attach Skills
        {selectedCount > 0 && (
          <span className="ml-2 text-sm font-normal text-amber-400">
            ({selectedCount} selected)
          </span>
        )}
      </DialogHeader>
      <DialogBody className="space-y-4" onKeyDown={handleKeyDown}>
        {/* Search input */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
          <input
            ref={searchInputRef}
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search skills..."
            className="w-full pl-10 pr-4 py-2 bg-slate-800 border border-white/10 rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-amber-500/50 focus:border-amber-500/50"
            data-testid="skill-search-input"
            autoFocus
          />
        </div>

        {/* Keyboard hint */}
        <p className="text-xs text-slate-500">
          Use <kbd className="px-1 py-0.5 rounded bg-slate-700">↑</kbd>{" "}
          <kbd className="px-1 py-0.5 rounded bg-slate-700">↓</kbd> to navigate,{" "}
          <kbd className="px-1 py-0.5 rounded bg-slate-700">Space</kbd> to toggle
        </p>

        {/* Skills list */}
        <div className="space-y-4" role="listbox" aria-multiselectable="true">
          {filteredSkills.length === 0 ? (
            <div className="text-center py-8 text-slate-400">
              No skills found
            </div>
          ) : (
            Array.from(skillsByCategory.entries()).map(
              ([category, categorySkills]) => (
                <div key={category}>
                  <h3 className="text-xs font-medium text-slate-500 uppercase tracking-wider mb-2">
                    {category}
                  </h3>
                  <div className="space-y-2">
                    {categorySkills.map((skill) => {
                      const IconComponent = skill.icon
                        ? ICON_MAP[skill.icon] || BookOpen
                        : BookOpen;
                      const isSelected = selectedSkillIds.includes(skill.id);
                      const skillIndex = skillsWithIndices.find(
                        (s) => s.skill.id === skill.id
                      )?.index ?? -1;
                      const isFocused = focusedIndex === skillIndex;

                      return (
                        <button
                          key={skill.id}
                          ref={(el) => {
                            skillRefs.current[skillIndex] = el;
                          }}
                          onClick={() => onToggle(skill.id)}
                          onFocus={() => setFocusedIndex(skillIndex)}
                          role="option"
                          aria-selected={isSelected}
                          tabIndex={isFocused ? 0 : -1}
                          className={`
                            w-full flex items-start gap-3 p-3 rounded-lg border text-left transition-colors
                            ${
                              isSelected
                                ? "bg-amber-500/20 border-amber-500/50"
                                : isFocused
                                  ? "bg-slate-700/50 border-amber-400/50 ring-2 ring-amber-500/30"
                                  : "bg-slate-800/50 border-white/10 hover:bg-slate-800 hover:border-white/20"
                            }
                          `}
                          data-testid={`skill-option-${skill.id}`}
                        >
                          <div
                            className={`
                            flex-shrink-0 p-1.5 rounded-lg
                            ${isSelected ? "bg-amber-500/30" : "bg-slate-700"}
                          `}
                          >
                            <IconComponent
                              className={`h-4 w-4 ${isSelected ? "text-amber-400" : "text-slate-300"}`}
                            />
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <span
                                className={`font-medium text-sm ${isSelected ? "text-amber-300" : "text-white"}`}
                              >
                                {skill.name}
                              </span>
                              {skill.tags && skill.tags.length > 0 && (
                                <div className="flex gap-1">
                                  {skill.tags.slice(0, 2).map((tag) => (
                                    <span
                                      key={tag}
                                      className="text-xs px-1.5 py-0.5 rounded bg-slate-700 text-slate-400"
                                    >
                                      {tag}
                                    </span>
                                  ))}
                                </div>
                              )}
                            </div>
                            <p className="text-xs text-slate-400 mt-1 line-clamp-2">
                              {skill.description}
                            </p>
                          </div>
                          <div
                            className={`
                            flex-shrink-0 w-5 h-5 rounded border flex items-center justify-center
                            ${
                              isSelected
                                ? "bg-amber-500 border-amber-500"
                                : "border-white/20"
                            }
                          `}
                          >
                            {isSelected && (
                              <Check className="h-3 w-3 text-white" />
                            )}
                          </div>
                        </button>
                      );
                    })}
                  </div>
                </div>
              )
            )
          )}
        </div>
      </DialogBody>
      <DialogFooter>
        <Button ref={doneButtonRef} variant="ghost" onClick={handleClose}>
          Done
        </Button>
      </DialogFooter>
    </Dialog>
  );
}
