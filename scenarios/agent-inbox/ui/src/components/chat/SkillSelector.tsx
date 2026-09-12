/**
 * SkillSelector - Modal for browsing and selecting skills to attach.
 *
 * Features:
 * - Search functionality
 * - Fully keyboard navigable
 * - Preview skill content (Left Arrow -> Eye icon -> Enter)
 * - Create new skill
 * - Hierarchical navigation by modes (when 10+ skills)
 */
import { useState, useMemo, useCallback, useRef, useEffect } from "react";
import { Search, Plus, ChevronLeft, RefreshCw } from "lucide-react";
import { Dialog, DialogHeader, DialogBody, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import type { Skill } from "@/lib/types/templates";
import { SkillEditorModal } from "@/components/settings/SkillEditorModal";
import { createSkill as apiCreateSkill } from "@/data/skills";
import { SkillSelectorItem } from "./SkillSelectorItem";

const HIERARCHICAL_THRESHOLD = 10;

interface SkillSelectorProps {
  open: boolean;
  onClose: () => void;
  skills: Skill[];
  selectedSkillIds: string[];
  onToggle: (skillId: string) => void;
  onSkillCreated?: () => void;
  onSyncSkills?: () => Promise<unknown>;
  isSyncing?: boolean;
}

export function SkillSelector({
  open,
  onClose,
  skills,
  selectedSkillIds,
  onToggle,
  onSkillCreated,
  onSyncSkills,
  isSyncing = false,
}: SkillSelectorProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [focusedIndex, setFocusedIndex] = useState(-1);
  const [focusedElement, setFocusedElement] = useState<"skill" | "preview">("skill");
  const searchInputRef = useRef<HTMLInputElement>(null);
  const skillRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const previewRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const doneButtonRef = useRef<HTMLButtonElement>(null);
  const [currentModePath, setCurrentModePath] = useState<string[]>([]);
  const [previewSkill, setPreviewSkill] = useState<Skill | null>(null);
  const [previewReadOnly, setPreviewReadOnly] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);

  const useHierarchicalMode = skills.length >= HIERARCHICAL_THRESHOLD;

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

  const { displaySkills, submodes, breadcrumb } = useMemo(() => {
    if (!useHierarchicalMode || searchQuery.trim()) {
      return { displaySkills: searchFilteredSkills, submodes: [], breadcrumb: [] };
    }
    const atPath: Skill[] = [];
    const submodesSet = new Set<string>();
    for (const skill of skills) {
      if (!skill.modes || skill.modes.length === 0) {
        if (currentModePath.length === 0) atPath.push(skill);
        continue;
      }
      let matches = true;
      for (let i = 0; i < currentModePath.length; i++) {
        if (skill.modes[i] !== currentModePath[i]) { matches = false; break; }
      }
      if (!matches) continue;
      if (skill.modes.length === currentModePath.length) atPath.push(skill);
      else if (skill.modes.length > currentModePath.length) {
        const nextMode = skill.modes[currentModePath.length];
        if (nextMode) submodesSet.add(nextMode);
      }
    }
    return { displaySkills: atPath, submodes: Array.from(submodesSet).sort(), breadcrumb: currentModePath };
  }, [skills, searchFilteredSkills, useHierarchicalMode, currentModePath, searchQuery]);

  const skillsByCategory = useMemo(() => {
    const grouped = new Map<string, Skill[]>();
    for (const skill of displaySkills) {
      const skillRecord = skill as unknown as Record<string, unknown>;
      const legacyCategory = typeof skillRecord.category === "string" ? skillRecord.category : undefined;
      const category = skill.modes?.[0] || legacyCategory || "Other";
      const categorySkills = grouped.get(category) ?? [];
      if (!grouped.has(category)) grouped.set(category, categorySkills);
      categorySkills.push(skill);
    }
    return grouped;
  }, [displaySkills]);

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

  useEffect(() => {
    if (open) { setFocusedIndex(-1); setFocusedElement("skill"); setSearchQuery(""); setCurrentModePath([]); }
  }, [open]);

  useEffect(() => {
    setFocusedIndex(-1); setFocusedElement("skill");
  }, [searchQuery, currentModePath]);

  const handleClose = useCallback(() => { onClose(); setSearchQuery(""); setCurrentModePath([]); }, [onClose]);
  const navigateToMode = useCallback((mode: string) => { setCurrentModePath((prev) => [...prev, mode]); }, []);
  const navigateBack = useCallback(() => { setCurrentModePath((prev) => prev.slice(0, -1)); }, []);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      const maxIndex = displaySkills.length - 1;
      switch (e.key) {
        case "ArrowDown": e.preventDefault(); if (focusedElement === "preview") setFocusedElement("skill"); setFocusedIndex((prev) => prev < maxIndex ? prev + 1 : -1); break;
        case "ArrowUp": e.preventDefault(); if (focusedElement === "preview") setFocusedElement("skill"); setFocusedIndex((prev) => prev > -1 ? prev - 1 : maxIndex); break;
        case "ArrowLeft": if (focusedIndex >= 0 && focusedElement === "skill") { e.preventDefault(); setFocusedElement("preview"); } break;
        case "ArrowRight": if (focusedIndex >= 0 && focusedElement === "preview") { e.preventDefault(); setFocusedElement("skill"); } break;
        case "Tab":
          if (!e.shiftKey && focusedIndex === -1) { if (displaySkills.length > 0) { e.preventDefault(); setFocusedIndex(0); setFocusedElement("skill"); } }
          else if (e.shiftKey && focusedIndex === 0) { e.preventDefault(); setFocusedIndex(-1); }
          else if (!e.shiftKey && focusedIndex === maxIndex) { e.preventDefault(); doneButtonRef.current?.focus(); }
          else if (focusedIndex >= 0) { e.preventDefault(); setFocusedIndex((prev) => e.shiftKey ? prev - 1 : prev + 1); }
          break;
        case "Enter": case " ":
          if (focusedIndex >= 0) {
            e.preventDefault();
            const skill = skillsWithIndices.find((s) => s.index === focusedIndex)?.skill;
            if (skill) {
              if (focusedElement === "preview") setPreviewSkill(skill);
              else onToggle(skill.id);
            }
          }
          break;
        case "Escape": if (focusedElement === "preview") { e.preventDefault(); setFocusedElement("skill"); } break;
      }
    },
    [displaySkills.length, focusedIndex, focusedElement, skillsWithIndices, onToggle]
  );

  useEffect(() => {
    if (!open) return;
    if (focusedIndex === -1) { searchInputRef.current?.focus(); }
    else if (focusedElement === "preview" && previewRefs.current[focusedIndex]) { previewRefs.current[focusedIndex].focus(); }
    else if (skillRefs.current[focusedIndex]) { skillRefs.current[focusedIndex].focus(); skillRefs.current[focusedIndex].scrollIntoView({ block: "nearest", behavior: "smooth" }); }
  }, [focusedIndex, focusedElement, open]);

  const handleCreateSkill = useCallback(async (skillData: Omit<Skill, "id" | "createdAt" | "updatedAt">) => {
    try { await apiCreateSkill(skillData); onSkillCreated?.(); setShowCreateModal(false); }
    catch (error) { console.error("Failed to create skill:", error); }
  }, [onSkillCreated]);

  const selectedCount = selectedSkillIds.length;

  return (
    <>
      <Dialog open={open} onClose={handleClose} className="max-w-lg">
        <DialogHeader onClose={handleClose}>
          <div className="flex items-center gap-2">
            <span>Attach Skills</span>
            {selectedCount > 0 && <span className="text-sm font-normal text-amber-400">({selectedCount} selected)</span>}
          </div>
          <div className="flex items-center gap-1">
            {onSyncSkills && (
              <button onClick={() => { void onSyncSkills(); }} disabled={isSyncing} className="p-1.5 rounded-lg hover:bg-white/10 text-slate-400 hover:text-white transition-colors disabled:opacity-50" title="Sync skills from prompt-manager">
                <RefreshCw className={`h-4 w-4 ${isSyncing ? "animate-spin" : ""}`} />
              </button>
            )}
            <button onClick={() => setShowCreateModal(true)} className="p-1.5 rounded-lg hover:bg-white/10 text-slate-400 hover:text-white transition-colors" title="Create new skill">
              <Plus className="h-4 w-4" />
            </button>
          </div>
        </DialogHeader>

        <DialogBody className="space-y-4" onKeyDown={handleKeyDown}>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
            <input ref={searchInputRef} type="text" value={searchQuery} onChange={(e) => setSearchQuery(e.target.value)} placeholder="Search skills..." className="w-full pl-10 pr-4 py-2 bg-slate-800 border border-white/10 rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-amber-500/50 focus:border-amber-500/50" data-testid="skill-search-input" autoFocus />
          </div>

          {useHierarchicalMode && !searchQuery.trim() && breadcrumb.length > 0 && (
            <div className="flex items-center gap-2 text-sm">
              <button onClick={navigateBack} className="flex items-center gap-1 px-2 py-1 rounded-lg hover:bg-white/10 text-slate-400 hover:text-white transition-colors"><ChevronLeft className="h-4 w-4" />Back</button>
              <span className="text-slate-500">{breadcrumb.join(" / ")}</span>
            </div>
          )}

          {useHierarchicalMode && !searchQuery.trim() && submodes.length > 0 && (
            <div className="flex flex-wrap gap-2">
              {submodes.map((mode) => (
                <button key={mode} onClick={() => navigateToMode(mode)} className="px-3 py-1.5 text-sm rounded-lg bg-slate-800 border border-white/10 text-slate-300 hover:bg-slate-700 hover:text-white transition-colors">{mode}</button>
              ))}
            </div>
          )}

          <p className="text-xs text-slate-500">
            <kbd className="px-1 py-0.5 rounded bg-slate-700">Up</kbd>{" "}
            <kbd className="px-1 py-0.5 rounded bg-slate-700">Down</kbd> navigate,{" "}
            <kbd className="px-1 py-0.5 rounded bg-slate-700">Space</kbd> toggle,{" "}
            <kbd className="px-1 py-0.5 rounded bg-slate-700">Left</kbd> preview
          </p>

          <div className="space-y-4 max-h-[400px] overflow-y-auto" role="listbox" aria-multiselectable="true">
            {displaySkills.length === 0 && submodes.length === 0 ? (
              <div className="text-center py-8 text-slate-400">
                {searchQuery.trim() ? "No skills found" : skills.length === 0 ? (
                  <div className="space-y-2">
                    <p>No skills available</p>
                    {onSyncSkills && (
                      <button onClick={() => { void onSyncSkills(); }} disabled={isSyncing} className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm bg-slate-700 hover:bg-slate-600 text-white rounded-lg transition-colors disabled:opacity-50">
                        <RefreshCw className={`h-3.5 w-3.5 ${isSyncing ? "animate-spin" : ""}`} />
                        {isSyncing ? "Syncing..." : "Sync from prompt-manager"}
                      </button>
                    )}
                  </div>
                ) : "No skills at this level"}
              </div>
            ) : (
              Array.from(skillsByCategory.entries()).map(([category, categorySkills]) => (
                <div key={category}>
                  {(!useHierarchicalMode || searchQuery.trim() || breadcrumb.length === 0) && (
                    <h3 className="text-xs font-medium text-slate-500 uppercase tracking-wider mb-2">{category}</h3>
                  )}
                  <div className="space-y-2">
                    {categorySkills.map((skill) => {
                      const skillIndex = skillsWithIndices.find((s) => s.skill.id === skill.id)?.index ?? -1;
                      const isFocused = focusedIndex === skillIndex;
                      return (
                        <SkillSelectorItem
                          key={skill.id}
                          skill={skill}
                          isSelected={selectedSkillIds.includes(skill.id)}
                          isFocused={isFocused}
                          isSkillFocused={isFocused && focusedElement === "skill"}
                          isPreviewFocused={isFocused && focusedElement === "preview"}
                          skillIndex={skillIndex}
                          onToggle={onToggle}
                          onPreview={setPreviewSkill}
                          onFocus={(idx) => { setFocusedIndex(idx); setFocusedElement("skill"); }}
                          skillRef={(el) => { skillRefs.current[skillIndex] = el; }}
                          previewRef={(el) => { previewRefs.current[skillIndex] = el; }}
                        />
                      );
                    })}
                  </div>
                </div>
              ))
            )}
          </div>
        </DialogBody>

        <DialogFooter>
          <Button ref={doneButtonRef} variant="ghost" onClick={handleClose}>Done</Button>
        </DialogFooter>
      </Dialog>

      <SkillEditorModal
        open={!!previewSkill}
        onClose={() => { setPreviewSkill(null); setPreviewReadOnly(true); }}
        skill={previewSkill || undefined}
        readOnly={previewReadOnly}
        onEdit={() => setPreviewReadOnly(false)}
        onSave={(skillData) => {
          if (!previewSkill) return;
          void (async () => {
            try {
              const { updateSkill } = await import("@/data/skills");
              await updateSkill(previewSkill.id, skillData);
              onSkillCreated?.();
            } catch (error) { console.error("Failed to update skill:", error); }
          })();
        }}
      />

      <SkillEditorModal
        open={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onSave={(skillData) => { void handleCreateSkill(skillData); }}
      />
    </>
  );
}
