/**
 * useMessageInput - Orchestrates all sub-hooks for the MessageInput component.
 *
 * This hook wires together draft persistence, attachments, templates, skills,
 * slash commands, suggestions, and send logic into a single return value.
 */
import { useState, useCallback, useRef, useEffect } from "react";
import { useAttachments } from "../../hooks/useAttachments";
import { useTools } from "../../hooks/useTools";
import { useTemplatesAndSkills } from "../../hooks/useTemplatesAndSkills";
import { useSuggestionsSettings } from "../../hooks/useSuggestionsSettings";
import { useAutoSuggestSkills } from "../../hooks/useAutoSuggestSkills";
import { useModeHistory } from "../../hooks/useModeHistory";
import { useAIMerge } from "../../hooks/useAIMerge";
import { useMessageDraft } from "../../hooks/useMessageDraft";

import type { MessageInputProps } from "./MessageInput.types";
import {
  getSuggestionsExpandedDefault,
  setSuggestionsExpandedStorage,
} from "./MessageInput.types";
import { useSlashCommands } from "./useSlashCommands";
import { useTemplateActions } from "./useTemplateActions";
import { useAttachmentHandlers } from "./useAttachmentHandlers";
import { useSendMessage } from "./useSendMessage";
import { useTextareaEffects, useModelCapabilities } from "./useMessageInputEffects";
import { useMessageInputHandlers } from "./useMessageInputHandlers";

export function useMessageInput(props: MessageInputProps) {
  const {
    onSend,
    isLoading,
    enableAttachments = true,
    enableWebSearch = true,
    enableForceTools = true,
    autoFocus = false,
    currentModel = null,
    chatId,
    chatWebSearchDefault = false,
    editingMessage,
    onCancelEdit,
    onSubmitEdit,
    onTemplateActivated,
    disableSend,
    disableSendReason,
    customUploadFn,
    placeholder = "Type a message...",
  } = props;
  // Read deprecated isGenerating indirectly to avoid triggering deprecation warnings
  const isGenerating = (props as { isGenerating?: boolean }).isGenerating;
  const loading = isLoading ?? isGenerating ?? false;

  // -- Draft persistence --
  const pageKey = chatId ?? "home";
  const { draft, setDraft, clearDraft } = useMessageDraft({ pageKey });
  const [message, setMessageState] = useState(draft);
  const prevPageKeyRef = useRef(pageKey);
  useEffect(() => {
    if (prevPageKeyRef.current !== pageKey) {
      prevPageKeyRef.current = pageKey;
      setMessageState(draft);
    }
  }, [pageKey, draft]);

  const setMessage = useCallback(
    (value: string) => {
      setMessageState(value);
      setDraft(value);
    },
    [setDraft],
  );

  const [webSearchEnabled, setWebSearchEnabled] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const isEditMode = !!editingMessage;

  // -- Suggestions settings --
  const {
    visible: suggestionsVisible,
    toggleVisible: toggleSuggestionsVisible,
    mergeModel,
    autoSuggest,
  } = useSuggestionsSettings();
  const [suggestionsExpanded, setSuggestionsExpandedState] = useState(
    getSuggestionsExpandedDefault,
  );
  const setSuggestionsExpanded = useCallback((expanded: boolean) => {
    setSuggestionsExpandedState(expanded);
    setSuggestionsExpandedStorage(expanded);
  }, []);

  const { history: modeHistory, recordUsage: recordModeUsage } =
    useModeHistory();
  const { mergeMessages, isMerging } = useAIMerge();

  // -- Attachments --
  const {
    attachments,
    addAttachment,
    removeAttachment,
    clearAttachments,
    isUploading,
    hasErrors,
    allUploaded,
    getUploadedIds,
  } = useAttachments(customUploadFn);

  const {
    forcedTool,
    setForcedTool,
    handleImageSelect,
    handlePDFSelect,
    handleForceTool,
    handleClearForcedTool,
  } = useAttachmentHandlers({ addAttachment });

  const { toolsByScenario } = useTools({
    chatId: enableForceTools && chatId ? chatId : undefined,
    enabled: enableForceTools && !!chatId,
  });

  // -- Templates & skills --
  const {
    templates,
    skills,
    skillsLoading,
    syncSkills,
    activeTemplate,
    setActiveTemplate,
    updateTemplateVariables,
    getFilledTemplateContent,
    clearTemplate,
    isTemplateValid,
    getTemplateMissingFields,
    selectedSkillIds,
    addSkill,
    removeSkill,
    toggleSkill,
    getSelectedSkills,
    buildSkillPayloads,
    filterCommands,
    resetAll: resetTemplatesAndSkills,
    currentModePath,
    navigateToMode,
    navigateBack,
    resetModePath,
    createTemplate,
    updateTemplate,
    deleteTemplate,
    resetTemplate,
  } = useTemplatesAndSkills();

  const {
    suggestions: suggestedSkills,
    isLoading: suggestionsLoading,
    didSearch: suggestionsDidSearch,
    dismiss: dismissSuggestion,
    dismissAll: dismissAllSuggestions,
  } = useAutoSuggestSkills({
    chatId,
    inputText: message,
    selectedSkillIds,
    enabled: autoSuggest.enabled,
    debounceMs: autoSuggest.debounceMs,
    throttleMs: autoSuggest.throttleMs,
    minInputLength: autoSuggest.minInputLength,
    minScorePercent: autoSuggest.minScorePercent,
    maxSuggestions: autoSuggest.maxSuggestions,
  });

  // -- Slash commands --
  const slashCommands = useSlashCommands({ filterCommands });

  // -- Template actions --
  const templateActions = useTemplateActions({
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
    createTemplate: (data) => { void createTemplate(data); },
    updateTemplate: (id, data) => { void updateTemplate(id, data); },
    deleteTemplate: (id) => { void deleteTemplate(id); },
    resetTemplate: (id) => { void resetTemplate(id); },
  });

  // -- Model capabilities (extracted) --
  const capabilities = useModelCapabilities({
    enableAttachments,
    enableWebSearch,
    currentModel,
    attachments,
    chatWebSearchDefault,
    setWebSearchEnabled,
  });

  // -- Send logic --
  const sendLogic = useSendMessage({
    message,
    setMessageState,
    clearDraft,
    effectiveAttachments: capabilities.effectiveAttachments,
    enableAttachments,
    enableWebSearch,
    isUploading,
    hasErrors,
    hasIncompatibleAttachments: capabilities.hasIncompatibleAttachments,
    allUploaded,
    getUploadedIds,
    clearAttachments,
    webSearchEnabled,
    setWebSearchEnabled,
    chatWebSearchDefault,
    forcedTool,
    setForcedTool,
    loading,
    isEditMode,
    onSend,
    onSubmitEdit,
    activeTemplate,
    getFilledTemplateContent,
    isTemplateValid,
    getTemplateMissingFields,
    selectedSkillIds,
    buildSkillPayloads,
    resetTemplatesAndSkills,
    textareaRef,
    disableSend,
    disableSendReason,
  });

  // -- Textarea side-effects (extracted) --
  useTextareaEffects({
    message,
    autoFocus,
    editingMessage,
    draft,
    setMessageState,
    textareaRef,
  });

  // -- Event handlers (extracted) --
  const handlers = useMessageInputHandlers({
    message,
    setMessage,
    setMessageState,
    draft,
    isEditMode,
    onCancelEdit,
    setWebSearchEnabled,
    slashCommands,
    templateActions,
    sendLogic,
  });

  return {
    // Core state
    message, setMessageState, draft, loading, isEditMode,
    textareaRef, placeholder, webSearchEnabled, setWebSearchEnabled,

    // Capabilities
    enableAttachments, enableWebSearch, enableForceTools,
    ...capabilities,

    // Attachments
    removeAttachment, isUploading, forcedTool,
    handleImageSelect, handlePDFSelect, handleForceTool,
    handleClearForcedTool, toolsByScenario,

    // Templates & skills
    templates, skills, skillsLoading, syncSkills,
    activeTemplate, updateTemplateVariables, getTemplateMissingFields,
    clearTemplate, selectedSkillIds, addSkill, removeSkill,
    toggleSkill, getSelectedSkills, currentModePath,
    navigateToMode, navigateBack, resetModePath, recordModeUsage,

    // Suggestions
    suggestionsVisible, suggestionsExpanded, setSuggestionsExpanded,
    suggestedSkills, suggestionsLoading, suggestionsDidSearch,
    dismissSuggestion, dismissAllSuggestions, modeHistory, isMerging,

    slashCommands, templateActions, sendLogic,
    handleWebSearchToggle: handlers.handleWebSearchToggle,
    handleMessageChange: handlers.handleMessageChange,
    handleKeyDown: handlers.handleKeyDown,
    chatId, onCancelEdit,
  };
}
