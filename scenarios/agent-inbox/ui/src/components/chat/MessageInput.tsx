import { useState, useCallback, useRef, useEffect, useMemo } from "react";
import { Send, Loader2, Check, X } from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import { AttachmentButton, type ForcedTool } from "./AttachmentButton";
import { AttachmentPreview } from "./AttachmentPreview";
import { WebSearchIndicator } from "./WebSearchIndicator";
import { ForcedToolIndicator } from "./ForcedToolIndicator";
import { TemplateIndicator } from "./TemplateIndicator";
import { SkillIndicator } from "./SkillIndicator";
import { TemplateSelector } from "./TemplateSelector";
import { SkillSelector } from "./SkillSelector";
import { ToolSelector } from "./ToolSelector";
import { TemplateVariableForm } from "./TemplateVariableForm";
import { SlashCommandPopup } from "./SlashCommandPopup";
import { Suggestions } from "./Suggestions";
import { AIMergeOverlay } from "./AIMergeOverlay";
import { TemplateEditorModal } from "./TemplateEditorModal";
import { useAttachments } from "../../hooks/useAttachments";
import { useTools } from "../../hooks/useTools";
import { useTemplatesAndSkills } from "../../hooks/useTemplatesAndSkills";
import { useSuggestionsSettings } from "../../hooks/useSuggestionsSettings";
import { useModeHistory } from "../../hooks/useModeHistory";
import { useAIMerge } from "../../hooks/useAIMerge";
import { supportsImages, supportsPDFs, supportsTools } from "../../lib/modelCapabilities";
import { getTemplateById } from "@/data/templates";
import { getSkillById } from "@/data/skills";
import type { Model, Message } from "../../lib/api";
import type { SlashCommand, Template, MergeAction } from "@/lib/types/templates";

export interface MessagePayload {
  content: string;
  attachmentIds: string[];
  webSearchEnabled: boolean;
  forcedTool?: ForcedTool;
  skillIds?: string[];
}

interface MessageInputProps {
  onSend: (payload: MessagePayload) => void;
  isLoading?: boolean;
  placeholder?: string;
  /** Enable attachment support (images, PDFs). Requires currentModel. Default: true */
  enableAttachments?: boolean;
  /** Enable web search toggle. Requires chatWebSearchDefault. Default: true */
  enableWebSearch?: boolean;
  /** Enable force tool selection. Requires chatId. Default: true */
  enableForceTools?: boolean;
  /** Auto-focus the textarea on mount. Default: false */
  autoFocus?: boolean;
  currentModel?: Model | null;
  /** Current chat ID (required for force tool selection) */
  chatId?: string;
  chatWebSearchDefault?: boolean;
  onChatWebSearchDefaultChange?: (enabled: boolean) => void;
  /** @deprecated Use isLoading instead */
  isGenerating?: boolean;
  /** Message being edited (enables edit mode when set) */
  editingMessage?: Message | null;
  /** Callback when edit is cancelled */
  onCancelEdit?: () => void;
  /** Callback when edit is submitted */
  onSubmitEdit?: (payload: MessagePayload) => void;
}

export function MessageInput({
  onSend,
  isLoading,
  isGenerating,
  placeholder = "Type a message...",
  enableAttachments = true,
  enableWebSearch = true,
  enableForceTools = true,
  autoFocus = false,
  currentModel = null,
  chatId,
  chatWebSearchDefault = false,
  onChatWebSearchDefaultChange,
  editingMessage,
  onCancelEdit,
  onSubmitEdit,
}: MessageInputProps) {
  // Support both isLoading and deprecated isGenerating
  const loading = isLoading ?? isGenerating ?? false;
  const [message, setMessage] = useState("");
  const [webSearchEnabled, setWebSearchEnabled] = useState(false);
  const [forcedTool, setForcedTool] = useState<ForcedTool | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Modal state
  const [showTemplateSelector, setShowTemplateSelector] = useState(false);
  const [showSkillSelector, setShowSkillSelector] = useState(false);
  const [showToolSelector, setShowToolSelector] = useState(false);
  const [showVariableForm, setShowVariableForm] = useState(true);
  const [shouldFocusTemplateForm, setShouldFocusTemplateForm] = useState(false);

  // Slash command state
  const [slashPopupOpen, setSlashPopupOpen] = useState(false);
  const [slashQuery, setSlashQuery] = useState("");
  const [slashSelectedIndex, setSlashSelectedIndex] = useState(0);
  const [slashPopupPosition, setSlashPopupPosition] = useState({ bottom: 60, left: 0 });

  // AI merge overlay state
  const [showMergeOverlay, setShowMergeOverlay] = useState(false);
  const [pendingTemplate, setPendingTemplate] = useState<Template | null>(null);
  const [savedMessage, setSavedMessage] = useState("");

  // Template editor state
  const [showTemplateEditor, setShowTemplateEditor] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<Template | undefined>(undefined);
  const [defaultEditorModes, setDefaultEditorModes] = useState<string[]>([]);

  // Edit mode detection
  const isEditMode = !!editingMessage;

  // Suggestions settings
  const {
    visible: suggestionsVisible,
    toggleVisible: toggleSuggestionsVisible,
    mergeModel,
  } = useSuggestionsSettings();

  // Mode history for frecency
  const { history: modeHistory, recordUsage: recordModeUsage } = useModeHistory();

  // AI merge functionality
  const { mergeMessages, isMerging } = useAIMerge();

  const {
    attachments,
    addAttachment,
    removeAttachment,
    clearAttachments,
    isUploading,
    hasErrors,
    allUploaded,
    getUploadedIds,
  } = useAttachments();

  // Get tools for force tool selection (only if enabled and have chatId)
  const { toolsByScenario, enabledTools } = useTools({
    chatId: enableForceTools && chatId ? chatId : undefined,
    enabled: enableForceTools && !!chatId,
  });

  // Templates and skills
  const {
    templates,
    skills,
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
    filterCommands,
    resetAll: resetTemplatesAndSkills,
    // Navigation
    currentModePath,
    navigateToMode,
    navigateBack,
    resetModePath,
    // CRUD
    createTemplate,
    updateTemplate,
    deleteTemplate,
    hideTemplate,
  } = useTemplatesAndSkills();

  // Only use attachments if enabled
  const effectiveAttachments = enableAttachments ? attachments : [];

  // Model capabilities (only relevant when attachments are enabled)
  const modelSupportsImages = enableAttachments && supportsImages(currentModel);
  const modelSupportsPDFs = enableAttachments && supportsPDFs(currentModel);
  // Web search requires tool support (it uses tools under the hood)
  const modelSupportsWebSearch = enableWebSearch && supportsTools(currentModel);

  // Reset web search to chat default when it changes, or disable if model doesn't support it
  useEffect(() => {
    if (enableWebSearch) {
      if (!modelSupportsWebSearch) {
        // Disable web search if model doesn't support it
        setWebSearchEnabled(false);
      } else {
        setWebSearchEnabled(chatWebSearchDefault);
      }
    }
  }, [chatWebSearchDefault, enableWebSearch, modelSupportsWebSearch]);

  // Check if any attachments are incompatible with the model
  const hasIncompatibleAttachments = effectiveAttachments.some((att) => {
    if (att.type === "image" && !modelSupportsImages) return true;
    if (att.type === "pdf" && !modelSupportsPDFs) return true;
    return false;
  });

  // Filtered slash commands
  const filteredSlashCommands = useMemo(
    () => filterCommands(slashQuery),
    [filterCommands, slashQuery]
  );

  // Auto-close slash popup when no results and user has typed something
  useEffect(() => {
    if (slashPopupOpen && slashQuery.length > 0 && filteredSlashCommands.length === 0) {
      setSlashPopupOpen(false);
    }
  }, [slashPopupOpen, slashQuery, filteredSlashCommands.length]);

  // Auto-resize textarea
  useEffect(() => {
    const textarea = textareaRef.current;
    if (textarea) {
      textarea.style.height = "auto";
      textarea.style.height = `${Math.min(textarea.scrollHeight, 200)}px`;
    }
  }, [message]);

  // Auto-focus on mount if enabled
  useEffect(() => {
    if (autoFocus && textareaRef.current) {
      textareaRef.current.focus();
    }
  }, [autoFocus]);

  // Populate textarea when entering edit mode
  useEffect(() => {
    if (editingMessage) {
      setMessage(editingMessage.content);
      // Focus and move cursor to end
      if (textareaRef.current) {
        textareaRef.current.focus();
        textareaRef.current.setSelectionRange(
          editingMessage.content.length,
          editingMessage.content.length
        );
      }
    }
  }, [editingMessage]);

  const handleSubmit = useCallback(() => {
    const trimmedMessage = message.trim();

    // Get filled template content if active, otherwise use raw message
    const finalContent = activeTemplate
      ? getFilledTemplateContent()
      : trimmedMessage;

    const hasContent = finalContent.trim() || effectiveAttachments.length > 0;

    // Block send if:
    // - No content (text or attachments)
    // - Still uploading (when attachments enabled)
    // - Has upload errors (when attachments enabled)
    // - Has incompatible attachments
    // - AI is loading/generating
    // - Template has missing required fields
    if (!hasContent || loading) {
      return;
    }

    if (activeTemplate && !isTemplateValid()) {
      return;
    }

    if (enableAttachments) {
      if (isUploading || hasErrors || hasIncompatibleAttachments) {
        return;
      }
      // If attachments exist but not all uploaded, wait
      if (effectiveAttachments.length > 0 && !allUploaded) {
        return;
      }
    }

    const payload: MessagePayload = {
      content: finalContent.trim(),
      attachmentIds: enableAttachments ? getUploadedIds() : [],
      webSearchEnabled: enableWebSearch ? webSearchEnabled : false,
      forcedTool: forcedTool ?? undefined,
      skillIds: selectedSkillIds.length > 0 ? selectedSkillIds : undefined,
    };

    // Call appropriate handler based on mode
    if (isEditMode && onSubmitEdit) {
      onSubmitEdit(payload);
    } else {
      onSend(payload);
    }

    setMessage("");
    if (enableAttachments) {
      clearAttachments();
    }
    // Reset web search to chat default after sending
    if (enableWebSearch) {
      setWebSearchEnabled(chatWebSearchDefault);
    }
    // Reset forced tool after sending
    setForcedTool(null);
    // Reset templates and skills after sending
    resetTemplatesAndSkills();

    // Reset height
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
    }
  }, [
    message,
    effectiveAttachments,
    isUploading,
    hasErrors,
    hasIncompatibleAttachments,
    loading,
    allUploaded,
    getUploadedIds,
    webSearchEnabled,
    forcedTool,
    onSend,
    clearAttachments,
    chatWebSearchDefault,
    enableAttachments,
    enableWebSearch,
    isEditMode,
    onSubmitEdit,
    activeTemplate,
    getFilledTemplateContent,
    isTemplateValid,
    selectedSkillIds,
    resetTemplatesAndSkills,
  ]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      // Handle slash command popup navigation
      if (slashPopupOpen) {
        if (e.key === "ArrowDown") {
          e.preventDefault();
          setSlashSelectedIndex((prev) =>
            prev < filteredSlashCommands.length - 1 ? prev + 1 : 0
          );
          return;
        }
        if (e.key === "ArrowUp") {
          e.preventDefault();
          setSlashSelectedIndex((prev) =>
            prev > 0 ? prev - 1 : filteredSlashCommands.length - 1
          );
          return;
        }
        if (e.key === "Enter") {
          e.preventDefault();
          if (filteredSlashCommands[slashSelectedIndex]) {
            handleSlashCommandSelect(filteredSlashCommands[slashSelectedIndex]);
          }
          return;
        }
        if (e.key === "Escape") {
          e.preventDefault();
          setSlashPopupOpen(false);
          return;
        }
      }

      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        handleSubmit();
      }
      // Escape cancels edit mode
      if (e.key === "Escape" && isEditMode && onCancelEdit) {
        e.preventDefault();
        setMessage("");
        onCancelEdit();
      }
    },
    [handleSubmit, isEditMode, onCancelEdit, slashPopupOpen, filteredSlashCommands, slashSelectedIndex]
  );

  const handleWebSearchToggle = useCallback((enabled: boolean) => {
    setWebSearchEnabled(enabled);
  }, []);

  // Handle message change with slash command detection
  const handleMessageChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      const value = e.target.value;
      const cursorPosition = e.target.selectionStart;
      setMessage(value);

      // Check for slash command trigger
      const textBeforeCursor = value.slice(0, cursorPosition);
      const lastNewlineIndex = textBeforeCursor.lastIndexOf("\n");
      const lineStart = lastNewlineIndex + 1;
      const lineBeforeCursor = textBeforeCursor.slice(lineStart);

      // Match "/" at start of line followed by optional word characters
      const slashMatch = lineBeforeCursor.match(/^\/(\S*)$/);

      if (slashMatch) {
        const query = slashMatch[1];
        setSlashQuery(query);
        setSlashPopupOpen(true);
        setSlashSelectedIndex(0);
        // Position popup above the cursor
        setSlashPopupPosition({ bottom: 60, left: 8 });
      } else {
        setSlashPopupOpen(false);
      }
    },
    []
  );

  // Handle template selection (may show merge overlay if message has content)
  const handleTemplateSelect = useCallback(
    (template: Template) => {
      // If message has content, show merge overlay
      if (message.trim()) {
        setSavedMessage(message);
        setPendingTemplate(template);
        setShowMergeOverlay(true);
      } else {
        // No existing content, just set the template
        setActiveTemplate(template);
        setShowVariableForm(true);
        setShouldFocusTemplateForm(true);
      }
    },
    [message, setActiveTemplate]
  );

  // Handle merge overlay action
  const handleMergeAction = useCallback(
    async (action: MergeAction) => {
      if (!pendingTemplate) return;

      switch (action) {
        case "overwrite":
          // Clear message and use template
          setMessage("");
          setActiveTemplate(pendingTemplate);
          setShowVariableForm(true);
          setShouldFocusTemplateForm(true);
          break;
        case "merge":
          // Use AI to merge message with template
          if (chatId) {
            try {
              const filledTemplate = pendingTemplate.content; // Use unfilled template for merge
              const mergedContent = await mergeMessages(
                savedMessage,
                filledTemplate,
                mergeModel,
                chatId
              );
              setMessage(mergedContent);
              // Don't set template - the merged content is the final message
            } catch (error) {
              // On error, just overwrite
              setMessage("");
              setActiveTemplate(pendingTemplate);
              setShowVariableForm(true);
              setShouldFocusTemplateForm(true);
            }
          }
          break;
        case "cancel":
          // Restore original message
          setMessage(savedMessage);
          break;
      }

      // Clean up overlay state
      setShowMergeOverlay(false);
      setPendingTemplate(null);
      setSavedMessage("");
    },
    [pendingTemplate, savedMessage, chatId, mergeModel, mergeMessages, setActiveTemplate]
  );

  // Handle slash command selection
  const handleSlashCommandSelect = useCallback(
    (command: SlashCommand) => {
      setSlashPopupOpen(false);

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
          setWebSearchEnabled(true);
          break;
        case "direct-template":
          const template = getTemplateById(command.id);
          if (template) {
            handleTemplateSelect(template);
          }
          break;
        case "direct-skill":
          addSkill(command.id);
          break;
        case "tool":
          // Check for suggestions toggle
          if (command.id === "suggestions") {
            toggleSuggestionsVisible();
          } else {
            setShowToolSelector(true);
          }
          break;
      }
    },
    [message, handleTemplateSelect, addSkill, toggleSuggestionsVisible]
  );

  const handleImageSelect = useCallback(
    (file: File) => {
      addAttachment(file, "image");
    },
    [addAttachment]
  );

  const handlePDFSelect = useCallback(
    (file: File) => {
      addAttachment(file, "pdf");
    },
    [addAttachment]
  );

  const handleForceTool = useCallback((scenario: string, toolName: string) => {
    setForcedTool({ scenario, toolName });
  }, []);

  const handleClearForcedTool = useCallback(() => {
    setForcedTool(null);
  }, []);

  // Template editor handlers
  const handleOpenTemplateEditor = useCallback((template?: Template) => {
    setEditingTemplate(template);
    setDefaultEditorModes(currentModePath);
    setShowTemplateEditor(true);
  }, [currentModePath]);

  const handleCloseTemplateEditor = useCallback(() => {
    setShowTemplateEditor(false);
    setEditingTemplate(undefined);
    setDefaultEditorModes([]);
  }, []);

  const handleSaveTemplate = useCallback(
    (templateData: Omit<Template, "id" | "createdAt" | "updatedAt" | "isBuiltIn">) => {
      if (editingTemplate) {
        updateTemplate(editingTemplate.id, templateData);
      } else {
        createTemplate(templateData);
      }
      handleCloseTemplateEditor();
    },
    [editingTemplate, createTemplate, updateTemplate, handleCloseTemplateEditor]
  );

  const handleDeleteTemplateFromSuggestions = useCallback(
    (templateId: string) => {
      if (window.confirm("Are you sure you want to delete this template?")) {
        deleteTemplate(templateId);
      }
    },
    [deleteTemplate]
  );

  // Model supports tools (required for force tool)
  const modelSupportsToolUse = supportsTools(currentModel);

  // Determine if send button should be disabled
  const finalContent = activeTemplate ? getFilledTemplateContent() : message;
  const hasContent = finalContent.trim() || effectiveAttachments.length > 0;
  const canSend = (() => {
    if (!hasContent || loading) return false;
    if (activeTemplate && !isTemplateValid()) return false;
    if (enableAttachments) {
      if (isUploading || hasErrors || hasIncompatibleAttachments) return false;
      if (effectiveAttachments.length > 0 && !allUploaded) return false;
    }
    return true;
  })();

  // Build send button tooltip
  let sendTooltip = isEditMode ? "Save edit (Enter)" : "Send message (Enter)";
  if (loading) {
    sendTooltip = "AI is responding...";
  } else if (activeTemplate && !isTemplateValid()) {
    const missing = getTemplateMissingFields();
    sendTooltip = `Fill required fields: ${missing.join(", ")}`;
  } else if (enableAttachments && isUploading) {
    sendTooltip = "Uploading attachments...";
  } else if (enableAttachments && hasErrors) {
    sendTooltip = "Fix attachment errors before sending";
  } else if (enableAttachments && hasIncompatibleAttachments) {
    sendTooltip = "Remove attachments not supported by this model";
  }

  return (
    <div className="p-4" data-testid="message-input-container">
      {/* Edit mode banner */}
      {isEditMode && (
        <div className="mb-2 px-3 py-2 bg-amber-500/20 border border-amber-500/30 rounded-lg flex items-center justify-between">
          <span className="text-sm text-amber-300">Editing message</span>
          <button
            onClick={() => {
              setMessage("");
              onCancelEdit?.();
            }}
            className="text-xs text-amber-400 hover:text-amber-200 flex items-center gap-1"
          >
            <X className="h-3 w-3" />
            Cancel
          </button>
        </div>
      )}

      {/* Suggestions panel */}
      {suggestionsVisible && !isEditMode && (
        <Suggestions
          templates={templates}
          currentModePath={currentModePath}
          modeHistory={modeHistory}
          onSelectTemplate={handleTemplateSelect}
          onNavigateToMode={navigateToMode}
          onNavigateBack={navigateBack}
          onResetPath={resetModePath}
          onEditTemplate={handleOpenTemplateEditor}
          onDeleteTemplate={handleDeleteTemplateFromSuggestions}
          onHideTemplate={hideTemplate}
          onCreateTemplate={(modes) => {
            setDefaultEditorModes(modes);
            setEditingTemplate(undefined);
            setShowTemplateEditor(true);
          }}
          onRecordModeUsage={recordModeUsage}
        />
      )}

      {/* Attachment preview area */}
      {enableAttachments && effectiveAttachments.length > 0 && (
        <div className="mb-2">
          <AttachmentPreview
            attachments={effectiveAttachments}
            onRemove={removeAttachment}
            isUploading={isUploading}
          />
          {hasIncompatibleAttachments && (
            <div className="px-4 py-1 text-xs text-red-400">
              Some attachments are not supported by the selected model
            </div>
          )}
        </div>
      )}

      {/* Template variable form (collapsible, above input) */}
      {activeTemplate && showVariableForm && (
        <div className="mb-2 rounded-xl border border-white/10 overflow-hidden">
          <TemplateVariableForm
            activeTemplate={activeTemplate}
            onUpdateVariables={updateTemplateVariables}
            missingFields={getTemplateMissingFields()}
            autoFocus={shouldFocusTemplateForm}
            onTabOut={() => {
              setShouldFocusTemplateForm(false);
              textareaRef.current?.focus();
            }}
          />
        </div>
      )}

      {/* Input container with buttons inside */}
      <div className="relative flex items-end gap-2 p-3 bg-white/5 border border-white/10 rounded-xl focus-within:ring-2 focus-within:ring-indigo-500/50 focus-within:border-transparent transition-all">
        {/* AI Merge Overlay */}
        <AIMergeOverlay
          isOpen={showMergeOverlay}
          existingMessage={savedMessage}
          templateName={pendingTemplate?.name || ""}
          isMerging={isMerging}
          onAction={handleMergeAction}
        />

        {/* Attachment Button (inside, on left) */}
        {enableAttachments && (
          <AttachmentButton
            onImageSelect={handleImageSelect}
            onPDFSelect={handlePDFSelect}
            webSearchEnabled={webSearchEnabled}
            onWebSearchToggle={enableWebSearch ? handleWebSearchToggle : undefined}
            disabled={loading}
            modelSupportsImages={modelSupportsImages}
            modelSupportsPDFs={modelSupportsPDFs}
            modelSupportsWebSearch={modelSupportsWebSearch}
            enabledToolsByScenario={enableForceTools && chatId ? toolsByScenario : undefined}
            forcedTool={forcedTool}
            onForceTool={enableForceTools && chatId && modelSupportsToolUse ? handleForceTool : undefined}
            modelSupportsTools={modelSupportsToolUse}
            onOpenTemplateSelector={() => setShowTemplateSelector(true)}
            onOpenSkillSelector={() => setShowSkillSelector(true)}
            activeTemplate={activeTemplate?.template}
            selectedSkillCount={selectedSkillIds.length}
          />
        )}

        {/* Textarea with relative positioning for slash popup */}
        <div className="relative flex-1">
          <textarea
            ref={textareaRef}
            value={message}
            onChange={handleMessageChange}
            onKeyDown={handleKeyDown}
            placeholder={activeTemplate ? "Template variables above..." : placeholder}
            disabled={loading}
            rows={1}
            className="w-full bg-transparent text-sm text-white placeholder:text-slate-500 resize-none focus:outline-none disabled:opacity-50 min-h-[40px]"
            data-testid="message-input"
          />

          {/* Slash command popup */}
          {slashPopupOpen && (
            <SlashCommandPopup
              commands={filteredSlashCommands}
              selectedIndex={slashSelectedIndex}
              onSelect={handleSlashCommandSelect}
              onClose={() => setSlashPopupOpen(false)}
              position={slashPopupPosition}
            />
          )}
        </div>

        {/* Character count */}
        {!loading && message.length > 0 && (
          <span className="text-xs text-slate-600 self-end pb-2">{message.length}</span>
        )}

        {/* Send/Save Button (inside, on right) */}
        <Tooltip content={sendTooltip}>
          <Button
            onClick={handleSubmit}
            disabled={!canSend}
            size="icon"
            className={`h-10 w-10 shrink-0 ${isEditMode ? "bg-amber-600 hover:bg-amber-500" : ""}`}
            data-testid={isEditMode ? "save-edit-button" : "send-message-button"}
          >
            {loading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : isEditMode ? (
              <Check className="h-4 w-4" />
            ) : (
              <Send className="h-4 w-4" />
            )}
          </Button>
        </Tooltip>
      </div>

      {/* Footer with keyboard hint and web search indicator */}
      <div className="flex items-center justify-between mt-2 px-1">
        <div className="flex items-center gap-3">
          <p className="text-xs text-slate-600">
            {isEditMode ? (
              <>
                Press <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-slate-400">Enter</kbd> to
                save, <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-slate-400">Escape</kbd>{" "}
                to cancel
              </>
            ) : (
              <>
                Press <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-slate-400">Enter</kbd> to
                send, <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-slate-400">Shift+Enter</kbd>{" "}
                for new line
              </>
            )}
          </p>
          {enableWebSearch && modelSupportsWebSearch && (
            <WebSearchIndicator
              enabled={webSearchEnabled}
              onDisable={() => setWebSearchEnabled(false)}
            />
          )}
          {forcedTool && (
            <ForcedToolIndicator
              scenario={forcedTool.scenario}
              toolName={forcedTool.toolName}
              onClear={handleClearForcedTool}
            />
          )}
          {activeTemplate && (
            <TemplateIndicator
              template={activeTemplate.template}
              onClear={clearTemplate}
              onEdit={() => setShowVariableForm(true)}
            />
          )}
          {selectedSkillIds.length > 0 && (
            <SkillIndicator
              skills={getSelectedSkills()}
              onRemove={removeSkill}
              onAdd={() => setShowSkillSelector(true)}
            />
          )}
        </div>
        {loading && (
          <span className="text-xs text-indigo-400 flex items-center gap-1">
            <Loader2 className="h-3 w-3 animate-spin" />
            AI is responding...
          </span>
        )}
      </div>

      {/* Template Selector Modal */}
      <TemplateSelector
        open={showTemplateSelector}
        onClose={() => {
          setShowTemplateSelector(false);
          textareaRef.current?.focus();
        }}
        templates={templates}
        onSelect={(template) => {
          setShowTemplateSelector(false);
          handleTemplateSelect(template);
        }}
        activeTemplateId={activeTemplate?.template.id}
      />

      {/* Skill Selector Modal */}
      <SkillSelector
        open={showSkillSelector}
        onClose={() => {
          setShowSkillSelector(false);
          textareaRef.current?.focus();
        }}
        skills={skills}
        selectedSkillIds={selectedSkillIds}
        onToggle={toggleSkill}
      />

      {/* Tool Selector Modal */}
      <ToolSelector
        open={showToolSelector}
        onClose={() => {
          setShowToolSelector(false);
          textareaRef.current?.focus();
        }}
        toolsByScenario={toolsByScenario}
        forcedTool={forcedTool}
        onSelect={handleForceTool}
        onClear={handleClearForcedTool}
      />

      {/* Template Editor Modal */}
      <TemplateEditorModal
        open={showTemplateEditor}
        onClose={handleCloseTemplateEditor}
        template={editingTemplate}
        defaultModes={defaultEditorModes}
        onSave={handleSaveTemplate}
      />
    </div>
  );
}
