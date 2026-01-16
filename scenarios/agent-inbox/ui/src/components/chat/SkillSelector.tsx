/**
 * SkillSelector - Modal for browsing and selecting skills to attach.
 *
 * Features:
 * - Search functionality
 * - Fully keyboard navigable
 * - Preview skill content (Left Arrow → Eye icon → Enter)
 * - Create new skill
 * - Hierarchical navigation by modes (when 10+ skills)
 */
import { useState, useMemo, useCallback, useRef, useEffect, type ComponentType, type SVGProps } from "react";
import {
  BookOpen,
  Search,
  Check,
  Eye,
  Plus,
  ChevronLeft,
  Construction,
} from "lucide-react";
import * as LucideIcons from "lucide-react";
import { Dialog, DialogHeader, DialogBody, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import type { Skill } from "@/lib/types/templates";
import { SkillEditorModal } from "@/components/settings/SkillEditorModal";
import { createSkill as apiCreateSkill } from "@/data/skills";

// Type for Lucide icon components
type IconComponent = ComponentType<SVGProps<SVGSVGElement> & { className?: string }>;

// Get icon component from name
function getIconComponent(name: string): IconComponent {
  const Icon = (LucideIcons as unknown as Record<string, IconComponent>)[name];
  return Icon || BookOpen;
}

// Threshold for switching to hierarchical mode
const HIERARCHICAL_THRESHOLD = 10;

interface SkillSelectorProps {
  open: boolean;
  onClose: () => void;
  skills: Skill[];
  selectedSkillIds: string[];
  onToggle: (skillId: string) => void;
  onSkillCreated?: () => void; // Callback when a new skill is created
}

export function SkillSelector({
  open,
  onClose,
  skills,
  selectedSkillIds,
  onToggle,
  onSkillCreated,
}: SkillSelectorProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [focusedIndex, setFocusedIndex] = useState(-1); // -1 = search focused
  const [focusedElement, setFocusedElement] = useState<"skill" | "preview">("skill");
  const searchInputRef = useRef<HTMLInputElement>(null);
  const skillRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const previewRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const doneButtonRef = useRef<HTMLButtonElement>(null);

  // Hierarchical navigation state
  const [currentModePath, setCurrentModePath] = useState<string[]>([]);

  // Modal states
  const [previewSkill, setPreviewSkill] = useState<Skill | null>(null);
  const [previewReadOnly, setPreviewReadOnly] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);

  // Determine if we should use hierarchical mode
  const useHierarchicalMode = skills.length >= HIERARCHICAL_THRESHOLD;

  // Filter skills by search query
  const searchFilteredSkills = useMemo(() => {
    if (!searchQuery.trim()) return skills;

    const query = searchQuery.toLowerCase();
    return skills.filter(
      (s) =>
        s.name.toLowerCase().includes(query) ||
        s.description.toLowerCase().includes(query) ||
        s.modes?.some((m) => m.toLowerCase().includes(query)) ||
        s.tags?.some((t) => t.toLowerCase().includes(query))
    );
  }, [skills, searchQuery]);

  // Get skills/submodes at current path (for hierarchical mode)
  const { displaySkills, submodes, breadcrumb } = useMemo(() => {
    if (!useHierarchicalMode || searchQuery.trim()) {
      // Flat mode or searching: show all filtered skills grouped by first mode
      return {
        displaySkills: searchFilteredSkills,
        submodes: [],
        breadcrumb: [],
      };
    }

    // Hierarchical mode: filter by current path
    const atPath: Skill[] = [];
    const submodesSet = new Set<string>();

    for (const skill of skills) {
      if (!skill.modes || skill.modes.length === 0) {
        // Skills without modes show at root
        if (currentModePath.length === 0) {
          atPath.push(skill);
        }
        continue;
      }

      // Check if skill's modes match current path
      let matches = true;
      for (let i = 0; i < currentModePath.length; i++) {
        if (skill.modes[i] !== currentModePath[i]) {
          matches = false;
          break;
        }
      }

      if (!matches) continue;

      // Skill matches current path
      if (skill.modes.length === currentModePath.length) {
        // Exactly at this level
        atPath.push(skill);
      } else if (skill.modes.length > currentModePath.length) {
        // Has deeper levels - add next submode
        const nextMode = skill.modes[currentModePath.length];
        if (nextMode) {
          submodesSet.add(nextMode);
        }
      }
    }

    return {
      displaySkills: atPath,
      submodes: Array.from(submodesSet).sort(),
      breadcrumb: currentModePath,
    };
  }, [skills, searchFilteredSkills, useHierarchicalMode, currentModePath, searchQuery]);

  // Group skills by first mode (for flat display)
  const skillsByCategory = useMemo(() => {
    const grouped = new Map<string, Skill[]>();

    for (const skill of displaySkills) {
      const category = skill.modes?.[0] || skill.category || "Other";
      const categorySkills = grouped.get(category) ?? [];
      if (!grouped.has(category)) {
        grouped.set(category, categorySkills);
      }
      categorySkills.push(skill);
    }

    return grouped;
  }, [displaySkills]);

  // Build flat list of skills with indices
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

  // Reset state when modal opens
  useEffect(() => {
    if (open) {
      setFocusedIndex(-1);
      setFocusedElement("skill");
      setSearchQuery("");
      setCurrentModePath([]);
    }
  }, [open]);

  // Reset focus when search or path changes
  useEffect(() => {
    setFocusedIndex(-1);
    setFocusedElement("skill");
  }, [searchQuery, currentModePath]);

  const handleClose = useCallback(() => {
    onClose();
    setSearchQuery("");
    setCurrentModePath([]);
  }, [onClose]);

  // Navigation helpers
  const navigateToMode = useCallback((mode: string) => {
    setCurrentModePath((prev) => [...prev, mode]);
  }, []);

  const navigateBack = useCallback(() => {
    setCurrentModePath((prev) => prev.slice(0, -1));
  }, []);

  // Handle keyboard navigation
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      const maxIndex = displaySkills.length - 1;

      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          if (focusedElement === "preview") {
            setFocusedElement("skill");
          }
          setFocusedIndex((prev) => {
            const next = prev < maxIndex ? prev + 1 : -1;
            return next;
          });
          break;

        case "ArrowUp":
          e.preventDefault();
          if (focusedElement === "preview") {
            setFocusedElement("skill");
          }
          setFocusedIndex((prev) => {
            const next = prev > -1 ? prev - 1 : maxIndex;
            return next;
          });
          break;

        case "ArrowLeft":
          if (focusedIndex >= 0 && focusedElement === "skill") {
            e.preventDefault();
            setFocusedElement("preview");
          }
          break;

        case "ArrowRight":
          if (focusedIndex >= 0 && focusedElement === "preview") {
            e.preventDefault();
            setFocusedElement("skill");
          }
          break;

        case "Tab":
          if (!e.shiftKey && focusedIndex === -1) {
            if (displaySkills.length > 0) {
              e.preventDefault();
              setFocusedIndex(0);
              setFocusedElement("skill");
            }
          } else if (e.shiftKey && focusedIndex === 0) {
            e.preventDefault();
            setFocusedIndex(-1);
          } else if (!e.shiftKey && focusedIndex === maxIndex) {
            e.preventDefault();
            doneButtonRef.current?.focus();
          } else if (focusedIndex >= 0) {
            e.preventDefault();
            setFocusedIndex((prev) => e.shiftKey ? prev - 1 : prev + 1);
          }
          break;

        case "Enter":
        case " ":
          if (focusedIndex >= 0) {
            e.preventDefault();
            const skill = skillsWithIndices.find((s) => s.index === focusedIndex)?.skill;
            if (skill) {
              if (focusedElement === "preview") {
                setPreviewSkill(skill);
              } else {
                onToggle(skill.id);
              }
            }
          }
          break;

        case "Escape":
          if (focusedElement === "preview") {
            e.preventDefault();
            setFocusedElement("skill");
          }
          break;
      }
    },
    [displaySkills.length, focusedIndex, focusedElement, skillsWithIndices, onToggle]
  );

  // Focus management
  useEffect(() => {
    if (!open) return;

    if (focusedIndex === -1) {
      searchInputRef.current?.focus();
    } else if (focusedElement === "preview" && previewRefs.current[focusedIndex]) {
      previewRefs.current[focusedIndex]?.focus();
    } else if (skillRefs.current[focusedIndex]) {
      skillRefs.current[focusedIndex]?.focus();
      skillRefs.current[focusedIndex]?.scrollIntoView({
        block: "nearest",
        behavior: "smooth",
      });
    }
  }, [focusedIndex, focusedElement, open]);

  // Handle new skill creation
  const handleCreateSkill = useCallback(
    async (skillData: Omit<Skill, "id" | "createdAt" | "updatedAt">) => {
      try {
        await apiCreateSkill(skillData);
        onSkillCreated?.();
        setShowCreateModal(false);
      } catch (error) {
        console.error("Failed to create skill:", error);
      }
    },
    [onSkillCreated]
  );

  const selectedCount = selectedSkillIds.length;

  return (
    <>
      <Dialog open={open} onClose={handleClose} className="max-w-lg">
        <DialogHeader onClose={handleClose}>
          <div className="flex items-center gap-2">
            <span>Attach Skills</span>
            {selectedCount > 0 && (
              <span className="text-sm font-normal text-amber-400">
                ({selectedCount} selected)
              </span>
            )}
          </div>
          <button
            onClick={() => setShowCreateModal(true)}
            className="p-1.5 rounded-lg hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
            title="Create new skill"
          >
            <Plus className="h-4 w-4" />
          </button>
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

          {/* Breadcrumb navigation (hierarchical mode only) */}
          {useHierarchicalMode && !searchQuery.trim() && breadcrumb.length > 0 && (
            <div className="flex items-center gap-2 text-sm">
              <button
                onClick={navigateBack}
                className="flex items-center gap-1 px-2 py-1 rounded-lg hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
              >
                <ChevronLeft className="h-4 w-4" />
                Back
              </button>
              <span className="text-slate-500">{breadcrumb.join(" / ")}</span>
            </div>
          )}

          {/* Submodes (hierarchical mode only) */}
          {useHierarchicalMode && !searchQuery.trim() && submodes.length > 0 && (
            <div className="flex flex-wrap gap-2">
              {submodes.map((mode) => (
                <button
                  key={mode}
                  onClick={() => navigateToMode(mode)}
                  className="px-3 py-1.5 text-sm rounded-lg bg-slate-800 border border-white/10 text-slate-300 hover:bg-slate-700 hover:text-white transition-colors"
                >
                  {mode}
                </button>
              ))}
            </div>
          )}

          {/* Keyboard hint */}
          <p className="text-xs text-slate-500">
            <kbd className="px-1 py-0.5 rounded bg-slate-700">↑</kbd>{" "}
            <kbd className="px-1 py-0.5 rounded bg-slate-700">↓</kbd> navigate,{" "}
            <kbd className="px-1 py-0.5 rounded bg-slate-700">Space</kbd> toggle,{" "}
            <kbd className="px-1 py-0.5 rounded bg-slate-700">←</kbd> preview
          </p>

          {/* Skills list */}
          <div className="space-y-4 max-h-[400px] overflow-y-auto" role="listbox" aria-multiselectable="true">
            {displaySkills.length === 0 && submodes.length === 0 ? (
              <div className="text-center py-8 text-slate-400">
                {searchQuery.trim() ? "No skills found" : "No skills at this level"}
              </div>
            ) : (
              Array.from(skillsByCategory.entries()).map(([category, categorySkills]) => (
                <div key={category}>
                  {(!useHierarchicalMode || searchQuery.trim() || breadcrumb.length === 0) && (
                    <h3 className="text-xs font-medium text-slate-500 uppercase tracking-wider mb-2">
                      {category}
                    </h3>
                  )}
                  <div className="space-y-2">
                    {categorySkills.map((skill) => {
                      const IconComponent = getIconComponent(skill.icon || "BookOpen");
                      const isSelected = selectedSkillIds.includes(skill.id);
                      const skillIndex = skillsWithIndices.find(
                        (s) => s.skill.id === skill.id
                      )?.index ?? -1;
                      const isFocused = focusedIndex === skillIndex;
                      const isPreviewFocused = isFocused && focusedElement === "preview";
                      const isSkillFocused = isFocused && focusedElement === "skill";

                      return (
                        <div
                          key={skill.id}
                          className={`
                            flex items-start gap-3 p-3 rounded-lg border transition-colors
                            ${
                              isSelected
                                ? "bg-amber-500/20 border-amber-500/50"
                                : isFocused
                                  ? "bg-slate-700/50 border-amber-400/50"
                                  : "bg-slate-800/50 border-white/10 hover:bg-slate-800 hover:border-white/20"
                            }
                          `}
                        >
                          {/* Preview button */}
                          <button
                            ref={(el) => { previewRefs.current[skillIndex] = el; }}
                            onClick={() => setPreviewSkill(skill)}
                            tabIndex={isPreviewFocused ? 0 : -1}
                            className={`
                              flex-shrink-0 p-1.5 rounded-lg transition-colors
                              ${isPreviewFocused
                                ? "bg-indigo-500/30 ring-2 ring-indigo-500/50"
                                : "bg-slate-700 hover:bg-slate-600"
                              }
                            `}
                            title="Preview skill"
                          >
                            <Eye className={`h-4 w-4 ${isPreviewFocused ? "text-indigo-300" : "text-slate-400"}`} />
                          </button>

                          {/* Main skill button */}
                          <button
                            ref={(el) => { skillRefs.current[skillIndex] = el; }}
                            onClick={() => onToggle(skill.id)}
                            onFocus={() => {
                              setFocusedIndex(skillIndex);
                              setFocusedElement("skill");
                            }}
                            role="option"
                            aria-selected={isSelected}
                            tabIndex={isSkillFocused ? 0 : -1}
                            className="flex-1 text-left min-w-0"
                          >
                            <div className="flex items-start gap-3">
                              <div className={`flex-shrink-0 p-1.5 rounded-lg ${isSelected ? "bg-amber-500/30" : "bg-slate-700"}`}>
                                <IconComponent className={`h-4 w-4 ${isSelected ? "text-amber-400" : "text-slate-300"}`} />
                              </div>
                              <div className="flex-1 min-w-0">
                                <div className="flex items-center gap-2">
                                  <span className={`font-medium text-sm ${isSelected ? "text-amber-300" : "text-white"}`}>
                                    {skill.name}
                                  </span>
                                  {skill.draft && (
                                    <span
                                      className="flex items-center gap-1 px-1.5 py-0.5 text-[10px] rounded bg-orange-900/50 text-orange-400 border border-orange-500/30"
                                      title="This skill is a draft and may not be fully working"
                                    >
                                      <Construction className="h-2.5 w-2.5" />
                                      Draft
                                    </span>
                                  )}
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
                            </div>
                          </button>

                          {/* Selection checkbox */}
                          <button
                            type="button"
                            onClick={() => onToggle(skill.id)}
                            tabIndex={-1}
                            className={`
                              flex-shrink-0 w-5 h-5 rounded border flex items-center justify-center transition-colors
                              ${isSelected ? "bg-amber-500 border-amber-500" : "border-white/20 hover:border-amber-400/50"}
                            `}
                            aria-label={isSelected ? "Deselect skill" : "Select skill"}
                          >
                            {isSelected && <Check className="h-3 w-3 text-white" />}
                          </button>
                        </div>
                      );
                    })}
                  </div>
                </div>
              ))
            )}
          </div>
        </DialogBody>

        <DialogFooter>
          <Button ref={doneButtonRef} variant="ghost" onClick={handleClose}>
            Done
          </Button>
        </DialogFooter>
      </Dialog>

      {/* Preview/Edit modal */}
      <SkillEditorModal
        open={!!previewSkill}
        onClose={() => {
          setPreviewSkill(null);
          setPreviewReadOnly(true);
        }}
        skill={previewSkill || undefined}
        readOnly={previewReadOnly}
        onEdit={() => setPreviewReadOnly(false)}
        onSave={async (skillData) => {
          if (!previewSkill) return;
          try {
            const { updateSkill } = await import("@/data/skills");
            await updateSkill(previewSkill.id, skillData);
            onSkillCreated?.(); // Refresh skills list
          } catch (error) {
            console.error("Failed to update skill:", error);
          }
        }}
      />

      {/* Create skill modal */}
      <SkillEditorModal
        open={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onSave={handleCreateSkill}
      />
    </>
  );
}
