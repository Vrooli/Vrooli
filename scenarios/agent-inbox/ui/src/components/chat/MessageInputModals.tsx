import { TemplateSelector } from "./TemplateSelector";
import { SkillSelector } from "./SkillSelector";
import { TemplateEditorModal } from "./TemplateEditorModal";
import type { Skill, Template } from "@/lib/types/templates";

interface MessageInputModalsProps {
  // Template selector
  showTemplateSelector: boolean;
  onCloseTemplateSelector: () => void;
  templates: Template[];
  onSelectTemplate: (template: Template) => void;
  activeTemplateId?: string;

  // Skill selector
  showSkillSelector: boolean;
  onCloseSkillSelector: () => void;
  skills: Skill[];
  selectedSkillIds: string[];
  onToggleSkill: (id: string) => void;
  onSyncSkills: () => Promise<unknown>;
  isSyncing: boolean;

  // Template editor
  showTemplateEditor: boolean;
  onCloseTemplateEditor: () => void;
  editingTemplate?: Template;
  defaultEditorModes: string[];
  onSaveTemplate: (
    data: Omit<Template, "id" | "createdAt" | "updatedAt" | "isBuiltIn">,
  ) => void;
}

export function MessageInputModals({
  showTemplateSelector,
  onCloseTemplateSelector,
  templates,
  onSelectTemplate,
  activeTemplateId,
  showSkillSelector,
  onCloseSkillSelector,
  skills,
  selectedSkillIds,
  onToggleSkill,
  onSyncSkills,
  isSyncing,
  showTemplateEditor,
  onCloseTemplateEditor,
  editingTemplate,
  defaultEditorModes,
  onSaveTemplate,
}: MessageInputModalsProps) {
  return (
    <>
      <TemplateSelector
        open={showTemplateSelector}
        onClose={onCloseTemplateSelector}
        templates={templates}
        onSelect={onSelectTemplate}
        activeTemplateId={activeTemplateId}
      />

      <SkillSelector
        open={showSkillSelector}
        onClose={onCloseSkillSelector}
        skills={skills}
        selectedSkillIds={selectedSkillIds}
        onToggle={onToggleSkill}
        onSyncSkills={onSyncSkills}
        isSyncing={isSyncing}
      />

      {showTemplateEditor && (
        <TemplateEditorModal
          open={showTemplateEditor}
          onClose={onCloseTemplateEditor}
          template={editingTemplate}
          defaultModes={defaultEditorModes}
          onSave={onSaveTemplate}
        />
      )}
    </>
  );
}
