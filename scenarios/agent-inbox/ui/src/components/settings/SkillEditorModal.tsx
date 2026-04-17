/**
 * Modal for creating, editing, and previewing skills.
 * Simpler than TemplateEditorModal since skills have no variables.
 */

import { useCallback, useEffect, useState } from "react";
import { X, Pencil, Loader2 } from "lucide-react";
import type { Skill, SkillWithSource } from "@/lib/types/templates";
import { ItemTreeSidebar } from "@/components/shared/ItemTreeSidebar";
import { SkillEditorForm } from "./SkillEditorForm";
import { UnsavedChangesDialog } from "./UnsavedChangesDialog";
import {
  getIconComponent,
  validateSkillForm,
  buildSkillData,
  useHasUnsavedChanges,
  type SkillFormState,
} from "./skillEditorUtils";
import { useSkillEditorMultiEdit } from "./useSkillEditorMultiEdit";

interface SkillEditorModalProps {
  open: boolean;
  onClose: () => void;
  skill?: Skill;
  onSave?: (skill: Omit<Skill, "id" | "createdAt" | "updatedAt">) => void;
  readOnly?: boolean;
  onEdit?: () => void;
  allSkills?: SkillWithSource[];
  onSelectSkill?: (skill: SkillWithSource) => void;
  onSaveAll?: (skills: Array<{ id: string; data: Omit<Skill, "id" | "createdAt" | "updatedAt"> }>) => Promise<void>;
}

export function SkillEditorModal({
  open, onClose, skill, onSave, readOnly = false,
  onEdit, allSkills, onSelectSkill, onSaveAll,
}: SkillEditorModalProps) {
  const isEditing = !!skill;
  const showSidebar = !!allSkills && allSkills.length > 0;

  // Form state
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [icon, setIcon] = useState("BookOpen");
  const [modes, setModes] = useState<string[]>([]);
  const [content, setContent] = useState("");
  const [tagsInput, setTagsInput] = useState("");
  const [targetToolId, setTargetToolId] = useState("");
  const [draft, setDraft] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [showCloseConfirm, setShowCloseConfirm] = useState(false);

  const formState: SkillFormState = { name, description, icon, modes, content, tagsInput, targetToolId, draft };
  const hasUnsavedChanges = useHasUnsavedChanges(readOnly, skill, formState);

  // Multi-item editing (extracted hook)
  const multiEdit = useSkillEditorMultiEdit({
    skill, readOnly, hasUnsavedChanges, formState,
    allSkills, onSelectSkill, onSaveAll,
  });

  // Handle close with unsaved changes check
  const handleClose = useCallback(() => {
    if (hasUnsavedChanges || multiEdit.pendingChanges.size > 0) {
      setShowCloseConfirm(true);
    } else {
      onClose();
    }
  }, [hasUnsavedChanges, multiEdit.pendingChanges.size, onClose]);

  // Handle escape key
  useEffect(() => {
    if (!open) return;
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        e.stopImmediatePropagation();
        handleClose();
      }
    };
    document.addEventListener("keydown", handleEscape, { capture: true });
    return () => document.removeEventListener("keydown", handleEscape, { capture: true });
  }, [open, handleClose]);

  // Initialize form when skill changes
  useEffect(() => {
    if (skill) {
      const pending = multiEdit.pendingChanges.get(skill.id);
      if (pending) {
        setName(pending.name); setDescription(pending.description);
        setIcon(pending.icon); setModes(pending.modes);
        setContent(pending.content); setTagsInput(pending.tagsInput);
        setTargetToolId(pending.targetToolId); setDraft(pending.draft);
      } else {
        setName(skill.name); setDescription(skill.description);
        setIcon(skill.icon || "BookOpen"); setModes(skill.modes || []);
        setContent(skill.content); setTagsInput(skill.tags?.join(", ") || "");
        setTargetToolId(skill.targetToolId || ""); setDraft(skill.draft || false);
      }
    } else {
      setName(""); setDescription(""); setIcon("BookOpen"); setModes([]);
      setContent(""); setTagsInput(""); setTargetToolId(""); setDraft(false);
    }
    setErrors({});
  }, [skill, open, multiEdit.pendingChanges]);

  // Handle save
  const handleSave = useCallback(() => {
    if (readOnly || !onSave) return;
    const newErrors = validateSkillForm(formState);
    setErrors(newErrors);
    if (Object.keys(newErrors).length > 0) return;
    onSave(buildSkillData(formState));
    onClose();
  }, [readOnly, onSave, formState, onClose]);

  if (!open) return null;

  const modalTitle = readOnly ? "Skill Preview" : isEditing ? "Edit Skill" : "Create Skill";

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={handleClose} />

      <div className={`relative bg-slate-900 border border-white/10 rounded-xl w-full max-h-[90vh] shadow-xl mx-4 flex flex-col ${
        showSidebar ? "max-w-6xl" : "max-w-2xl md:max-w-5xl"
      }`}>
        {/* Header */}
        <div className="flex-shrink-0 flex items-center justify-between p-4 border-b border-white/10">
          <h2 className="text-lg font-semibold text-white">{modalTitle}</h2>
          <div className="flex items-center gap-2">
            {readOnly && onEdit && (
              <button onClick={onEdit} className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg bg-indigo-600/20 text-indigo-300 hover:bg-indigo-600/30 transition-colors" title="Edit skill">
                <Pencil className="h-4 w-4" /> Edit
              </button>
            )}
            <button onClick={handleClose} className="p-1 rounded hover:bg-white/10 text-slate-400 hover:text-white transition-colors">
              <X className="h-5 w-5" />
            </button>
          </div>
        </div>

        {/* Content with optional sidebar */}
        <div className="flex-1 min-h-0 overflow-hidden flex">
          {showSidebar && (
            <ItemTreeSidebar
              items={multiEdit.itemsForTree}
              selectedItemId={skill?.id ?? null}
              onSelectItem={(id) => {
                const selected = allSkills.find((s) => s.id === id);
                if (selected) multiEdit.handleSelectSkill(selected);
              }}
              dirtyItemIds={multiEdit.dirtyItemIds}
              expandedNodes={multiEdit.expandedTreeNodes}
              onToggleNode={multiEdit.toggleTreeNode}
              renderItemIcon={(item) => {
                const IconComp = getIconComponent(item.icon || "BookOpen");
                return <IconComp className="h-3.5 w-3.5 flex-shrink-0 text-slate-400" />;
              }}
              title="Skills"
              className={multiEdit.isSidebarCollapsed ? "" : "w-60 flex-shrink-0"}
              isCollapsed={multiEdit.isSidebarCollapsed}
              onToggleCollapse={() => multiEdit.setIsSidebarCollapsed((prev: boolean) => !prev)}
            />
          )}

          <div className="flex-1 min-h-0 p-4 overflow-hidden">
            <SkillEditorForm
              name={name} onNameChange={setName}
              description={description} onDescriptionChange={setDescription}
              icon={icon} onIconChange={setIcon}
              draft={draft} onDraftChange={setDraft}
              modes={modes} onModesChange={setModes}
              tagsInput={tagsInput} onTagsInputChange={setTagsInput}
              targetToolId={targetToolId} onTargetToolIdChange={setTargetToolId}
              content={content} onContentChange={setContent}
              errors={errors} readOnly={readOnly}
            />
          </div>
        </div>

        {/* Footer */}
        <div className="flex-shrink-0 flex items-center justify-end gap-3 p-4 border-t border-white/10">
          <button onClick={onClose} className="px-4 py-2 text-sm text-slate-400 hover:text-white transition-colors">
            {readOnly ? "Close" : "Cancel"}
          </button>
          {!readOnly && (
            <>
              {showSidebar && multiEdit.dirtyCount > 1 && onSaveAll ? (
                <button onClick={() => { void multiEdit.handleSaveAll(); }} disabled={multiEdit.isSavingAll} className="flex items-center gap-2 px-4 py-2 text-sm font-medium bg-indigo-600 text-white rounded-lg hover:bg-indigo-500 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
                  {multiEdit.isSavingAll && <Loader2 className="h-4 w-4 animate-spin" />}
                  Save All Changes ({multiEdit.dirtyCount})
                </button>
              ) : (
                <button onClick={handleSave} className="px-4 py-2 text-sm font-medium bg-indigo-600 text-white rounded-lg hover:bg-indigo-500 transition-colors">
                  {isEditing ? "Save Changes" : "Create Skill"}
                </button>
              )}
            </>
          )}
        </div>
      </div>

      {showCloseConfirm && (
        <UnsavedChangesDialog
          dirtyCount={multiEdit.dirtyCount}
          onKeepEditing={() => setShowCloseConfirm(false)}
          onDiscard={() => {
            setShowCloseConfirm(false);
            multiEdit.resetPendingChanges();
            onClose();
          }}
        />
      )}
    </div>
  );
}
