import { TemplateSelector } from "./TemplateSelector";
import { SkillSelector } from "./SkillSelector";
import { ToolSelector } from "./ToolSelector";
import { TemplateEditorModal } from "./TemplateEditorModal";
import type { ForcedTool } from "./AttachmentButton";
import type { EffectiveTool } from "@/lib/api";
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

  // Tool selector
  showToolSelector: boolean;
  onCloseToolSelector: () => void;
  toolsByScenario: Map<string, EffectiveTool[]>;
  forcedTool: ForcedTool | null;
  onSelectTool: (scenario: string, toolName: string) => void;
  onClearTool: () => void;

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
  showToolSelector,
  onCloseToolSelector,
  toolsByScenario,
  forcedTool,
  onSelectTool,
  onClearTool,
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

      <ToolSelector
        open={showToolSelector}
        onClose={onCloseToolSelector}
        toolsByScenario={toolsByScenario}
        forcedTool={forcedTool}
        onSelect={onSelectTool}
        onClear={onClearTool}
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
