import { useState, useCallback } from "react";
import { getTemplateById } from "@/data/templates";
import type { MergeAction, SlashCommand, Template } from "@/lib/types/templates";

interface UseTemplateActionsOptions {
  message: string;
  setMessage: (value: string) => void;
  chatId?: string;
  mergeModel: string;
  /** From useAIMerge */
  mergeMessages: (
    existing: string,
    template: string,
    model: string,
    chatId: string,
  ) => Promise<string>;
  /** From useTemplatesAndSkills */
  setActiveTemplate: (template: Template) => void;
  onTemplateActivated?: (
    templateId: string,
    toolIds: string[],
  ) => Promise<void>;
  /** From useTemplatesAndSkills */
  addSkill: (id: string) => void;
  toggleSuggestionsVisible: () => void;
  currentModePath: string[];
  /** From useTemplatesAndSkills */
  createTemplate: (
    data: Omit<Template, "id" | "createdAt" | "updatedAt" | "isBuiltIn">,
  ) => void;
  updateTemplate: (
    id: string,
    data: Omit<Template, "id" | "createdAt" | "updatedAt" | "isBuiltIn">,
  ) => void;
  deleteTemplate: (id: string) => void;
  resetTemplate: (id: string) => void;
}

export function useTemplateActions({
  message,
  setMessage,
  chatId,
  mergeModel,
  mergeMessages,
  setActiveTemplate,
  onTemplateActivated,
  addSkill,
  toggleSuggestionsVisible,
  currentModePath,
  createTemplate,
  updateTemplate,
  deleteTemplate,
  resetTemplate,
}: UseTemplateActionsOptions) {
  // Modal state
  const [showTemplateSelector, setShowTemplateSelector] = useState(false);
  const [showSkillSelector, setShowSkillSelector] = useState(false);
  const [showToolSelector, setShowToolSelector] = useState(false);
  const [showVariableForm, setShowVariableForm] = useState(true);
  const [shouldFocusTemplateForm, setShouldFocusTemplateForm] = useState(false);

  // AI merge overlay state
  const [showMergeOverlay, setShowMergeOverlay] = useState(false);
  const [pendingTemplate, setPendingTemplate] = useState<Template | null>(null);
  const [savedMessage, setSavedMessage] = useState("");

  // Template editor state
  const [showTemplateEditor, setShowTemplateEditor] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<
    Template | undefined
  >(undefined);
  const [defaultEditorModes, setDefaultEditorModes] = useState<string[]>([]);

  // Handle template selection (may show merge overlay if message has content)
  const handleTemplateSelect = useCallback(
    async (template: Template) => {
      if (message.trim()) {
        setSavedMessage(message);
        setPendingTemplate(template);
        setShowMergeOverlay(true);
      } else {
        setActiveTemplate(template);
        setShowVariableForm(true);
        setShouldFocusTemplateForm(true);

        if (template.suggestedToolIds?.length && onTemplateActivated) {
          await onTemplateActivated(template.id, template.suggestedToolIds);
        }
      }
    },
    [message, setActiveTemplate, onTemplateActivated],
  );

  // Handle merge overlay action
  const handleMergeAction = useCallback(
    async (action: MergeAction) => {
      if (!pendingTemplate) return;

      switch (action) {
        case "overwrite":
          setMessage("");
          setActiveTemplate(pendingTemplate);
          setShowVariableForm(true);
          setShouldFocusTemplateForm(true);

          if (pendingTemplate.suggestedToolIds?.length && onTemplateActivated) {
            await onTemplateActivated(
              pendingTemplate.id,
              pendingTemplate.suggestedToolIds,
            );
          }
          break;
        case "merge":
          if (chatId) {
            try {
              const filledTemplate = pendingTemplate.content;
              const mergedContent = await mergeMessages(
                savedMessage,
                filledTemplate,
                mergeModel,
                chatId,
              );
              setMessage(mergedContent);
            } catch {
              setMessage("");
              setActiveTemplate(pendingTemplate);
              setShowVariableForm(true);
              setShouldFocusTemplateForm(true);

              if (
                pendingTemplate.suggestedToolIds?.length &&
                onTemplateActivated
              ) {
                await onTemplateActivated(
                  pendingTemplate.id,
                  pendingTemplate.suggestedToolIds,
                );
              }
            }
          }
          break;
        case "cancel":
          setMessage(savedMessage);
          break;
      }

      setShowMergeOverlay(false);
      setPendingTemplate(null);
      setSavedMessage("");
    },
    [
      pendingTemplate,
      savedMessage,
      chatId,
      mergeModel,
      mergeMessages,
      setActiveTemplate,
      onTemplateActivated,
      setMessage,
    ],
  );

  // Handle slash command selection
  const handleSlashCommandSelect = useCallback(
    (command: SlashCommand) => {
      // Clear the slash command text from input
      const slashStart = message.lastIndexOf("/");
      if (slashStart !== -1) {
        setMessage(message.slice(0, slashStart));
      }

      switch (command.type) {
        case "template":
          setShowTemplateSelector(true);
          break;
        case "skill":
          setShowSkillSelector(true);
          break;
        case "search":
          // Handled by caller (setWebSearchEnabled)
          break;
        case "direct-template":
          void getTemplateById(command.id).then((template) => {
            if (template) {
              void handleTemplateSelect(template);
            }
          });
          break;
        case "direct-skill":
          addSkill(command.id);
          break;
        case "tool":
          if (command.id === "suggestions") {
            toggleSuggestionsVisible();
          } else {
            setShowToolSelector(true);
          }
          break;
      }
    },
    [message, setMessage, handleTemplateSelect, addSkill, toggleSuggestionsVisible],
  );

  // Template editor handlers
  const handleOpenTemplateEditor = useCallback(
    (template?: Template) => {
      setEditingTemplate(template);
      setDefaultEditorModes(currentModePath);
      setShowTemplateEditor(true);
    },
    [currentModePath],
  );

  const handleCloseTemplateEditor = useCallback(() => {
    setShowTemplateEditor(false);
    setEditingTemplate(undefined);
    setDefaultEditorModes([]);
  }, []);

  const handleSaveTemplate = useCallback(
    (
      templateData: Omit<
        Template,
        "id" | "createdAt" | "updatedAt" | "isBuiltIn"
      >,
    ) => {
      if (editingTemplate) {
        updateTemplate(editingTemplate.id, templateData);
      } else {
        createTemplate(templateData);
      }
      handleCloseTemplateEditor();
    },
    [editingTemplate, createTemplate, updateTemplate, handleCloseTemplateEditor],
  );

  const handleDeleteTemplateFromSuggestions = useCallback(
    (templateId: string) => {
      if (window.confirm("Are you sure you want to delete this template?")) {
        deleteTemplate(templateId);
      }
    },
    [deleteTemplate],
  );

  const handleResetTemplateFromSuggestions = useCallback(
    (templateId: string) => {
      if (window.confirm("Reset this template to its default state?")) {
        resetTemplate(templateId);
      }
    },
    [resetTemplate],
  );

  return {
    // Modal state
    showTemplateSelector,
    setShowTemplateSelector,
    showSkillSelector,
    setShowSkillSelector,
    showToolSelector,
    setShowToolSelector,
    showVariableForm,
    setShowVariableForm,
    shouldFocusTemplateForm,
    setShouldFocusTemplateForm,

    // AI merge overlay
    showMergeOverlay,
    pendingTemplate,
    savedMessage,

    // Template editor
    showTemplateEditor,
    setShowTemplateEditor,
    editingTemplate,
    setEditingTemplate,
    defaultEditorModes,
    setDefaultEditorModes,

    // Handlers
    handleTemplateSelect,
    handleMergeAction,
    handleSlashCommandSelect,
    handleOpenTemplateEditor,
    handleCloseTemplateEditor,
    handleSaveTemplate,
    handleDeleteTemplateFromSuggestions,
    handleResetTemplateFromSuggestions,
  };
}
