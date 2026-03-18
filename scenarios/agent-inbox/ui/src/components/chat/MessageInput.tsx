import { X } from "lucide-react";
import { AttachmentPreview } from "./AttachmentPreview";
import { TemplateVariableForm } from "./TemplateVariableForm";
import { SuggestionsPanel } from "./SuggestionsPanel";
import { MessageInputFooter } from "./MessageInputFooter";
import { MessageInputModals } from "./MessageInputModals";
import { MessageInputArea } from "./MessageInputArea";
import { selectorsManifest } from "../../consts/selectors";

import type { MessageInputProps } from "./MessageInput.types";
import { useMessageInput } from "./useMessageInput";

// Re-export public API (consumers import from this path)
export type { MessagePayload } from "./MessageInput.types";

export function MessageInput(props: MessageInputProps) {
  const {
    activeTemplateId,
    onTemplateDeactivate,
  } = props;

  const messageInputTestIds = {
    container:
      selectorsManifest.selectors["messageInput.container"]?.testId ??
      "message-input-container",
    suggestionsToggle:
      selectorsManifest.selectors["messageInput.suggestionsToggle"]?.testId ??
      "suggestions-toggle",
    input:
      selectorsManifest.selectors["messageInput.input"]?.testId ??
      "message-input",
    sendButton:
      selectorsManifest.selectors["messageInput.sendButton"]?.testId ??
      "send-message-button",
  };

  const state = useMessageInput(props);

  const {
    draft,
    isEditMode,
    textareaRef,
    webSearchEnabled,
    setWebSearchEnabled,
    enableAttachments,
    enableWebSearch,
    modelSupportsWebSearch,
    effectiveAttachments,
    hasIncompatibleAttachments,
    removeAttachment,
    isUploading,
    forcedTool,
    handleClearForcedTool,
    templates,
    skills,
    skillsLoading,
    syncSkills,
    activeTemplate,
    updateTemplateVariables,
    getTemplateMissingFields,
    clearTemplate,
    selectedSkillIds,
    addSkill,
    removeSkill,
    toggleSkill,
    getSelectedSkills,
    currentModePath,
    navigateToMode,
    navigateBack,
    resetModePath,
    recordModeUsage,
    suggestionsVisible,
    suggestionsExpanded,
    setSuggestionsExpanded,
    suggestedSkills,
    suggestionsLoading,
    suggestionsDidSearch,
    dismissSuggestion,
    dismissAllSuggestions,
    modeHistory,
    templateActions,
    sendLogic,
    loading,
    handleForceTool,
    handleClearForcedTool: _hcft,
    toolsByScenario,
    setMessageState,
    onCancelEdit,
    chatId,
  } = state;

  return (
    <div
      className="p-2 sm:p-4 pb-[calc(env(safe-area-inset-bottom)+0.5rem)]"
      data-testid={messageInputTestIds.container}
    >
      {/* Edit mode banner */}
      {isEditMode && (
        <div className="mb-2 px-3 py-2 bg-amber-500/20 border border-amber-500/30 rounded-lg flex items-center justify-between">
          <span className="text-sm text-amber-300">Editing message</span>
          <button
            onClick={() => {
              setMessageState(draft);
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
        <SuggestionsPanel
          suggestionsExpanded={suggestionsExpanded}
          setSuggestionsExpanded={setSuggestionsExpanded}
          suggestionsToggleTestId={messageInputTestIds.suggestionsToggle}
          templates={templates}
          currentModePath={currentModePath}
          modeHistory={modeHistory}
          handleTemplateSelect={templateActions.handleTemplateSelect}
          navigateToMode={navigateToMode}
          navigateBack={navigateBack}
          resetModePath={resetModePath}
          handleOpenTemplateEditor={templateActions.handleOpenTemplateEditor}
          handleDeleteTemplateFromSuggestions={
            templateActions.handleDeleteTemplateFromSuggestions
          }
          handleResetTemplateFromSuggestions={
            templateActions.handleResetTemplateFromSuggestions
          }
          setDefaultEditorModes={templateActions.setDefaultEditorModes}
          setEditingTemplate={templateActions.setEditingTemplate}
          setShowTemplateEditor={templateActions.setShowTemplateEditor}
          recordModeUsage={recordModeUsage}
        />
      )}

      {/* Attachment preview */}
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

      {/* Template variable form */}
      {activeTemplate && templateActions.showVariableForm && (
        <div className="mb-2 rounded-xl border border-white/10 overflow-hidden">
          <TemplateVariableForm
            activeTemplate={activeTemplate}
            onUpdateVariables={updateTemplateVariables}
            missingFields={getTemplateMissingFields()}
            autoFocus={templateActions.shouldFocusTemplateForm}
            onTabOut={() => {
              templateActions.setShouldFocusTemplateForm(false);
              textareaRef.current?.focus();
            }}
          />
        </div>
      )}

      {/* Input container (extracted) */}
      <MessageInputArea
        state={state}
        inputTestId={messageInputTestIds.input}
        sendButtonTestId={messageInputTestIds.sendButton}
      />

      {/* Footer with keyboard hints and indicators */}
      <MessageInputFooter
        isEditMode={isEditMode}
        modKey={sendLogic.modKey}
        loading={loading}
        enableWebSearch={enableWebSearch}
        modelSupportsWebSearch={modelSupportsWebSearch}
        webSearchEnabled={webSearchEnabled}
        setWebSearchEnabled={setWebSearchEnabled}
        forcedTool={forcedTool}
        handleClearForcedTool={handleClearForcedTool}
        activeTemplate={activeTemplate}
        clearTemplate={clearTemplate}
        setShowVariableForm={templateActions.setShowVariableForm}
        activeTemplateId={activeTemplateId}
        onTemplateDeactivate={onTemplateDeactivate}
        selectedSkillIds={selectedSkillIds}
        getSelectedSkills={getSelectedSkills}
        removeSkill={removeSkill}
        setShowSkillSelector={templateActions.setShowSkillSelector}
        suggestedSkills={suggestedSkills}
        suggestionsLoading={suggestionsLoading}
        suggestionsDidSearch={suggestionsDidSearch}
        addSkill={addSkill}
        dismissSuggestion={dismissSuggestion}
        dismissAllSuggestions={dismissAllSuggestions}
      />

      {/* Modals */}
      <MessageInputModals
        showTemplateSelector={templateActions.showTemplateSelector}
        onCloseTemplateSelector={() => {
          templateActions.setShowTemplateSelector(false);
          textareaRef.current?.focus();
        }}
        templates={templates}
        onSelectTemplate={(template) => {
          templateActions.setShowTemplateSelector(false);
          templateActions.handleTemplateSelect(template);
        }}
        activeTemplateId={activeTemplate?.template.id}
        showSkillSelector={templateActions.showSkillSelector}
        onCloseSkillSelector={() => {
          templateActions.setShowSkillSelector(false);
          textareaRef.current?.focus();
        }}
        skills={skills}
        selectedSkillIds={selectedSkillIds}
        onToggleSkill={toggleSkill}
        onSyncSkills={syncSkills}
        isSyncing={skillsLoading}
        showToolSelector={templateActions.showToolSelector}
        onCloseToolSelector={() => {
          templateActions.setShowToolSelector(false);
          textareaRef.current?.focus();
        }}
        toolsByScenario={toolsByScenario}
        forcedTool={forcedTool}
        onSelectTool={handleForceTool}
        onClearTool={handleClearForcedTool}
        showTemplateEditor={templateActions.showTemplateEditor}
        onCloseTemplateEditor={templateActions.handleCloseTemplateEditor}
        editingTemplate={templateActions.editingTemplate}
        defaultEditorModes={templateActions.defaultEditorModes}
        onSaveTemplate={templateActions.handleSaveTemplate}
      />
    </div>
  );
}
