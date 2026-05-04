/**
 * MessageInputArea - The core textarea, attachment button, slash popup, and send button.
 *
 * Extracted from MessageInput to keep each module under 300 lines.
 */
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Send, Loader2, Check } from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import { AttachmentButton } from "./AttachmentButton";
import { AIMergeOverlay } from "./AIMergeOverlay";
import { SlashCommandPopup } from "./SlashCommandPopup";
import type { useMessageInput } from "./useMessageInput";

type MessageInputAreaState = Pick<
  ReturnType<typeof useMessageInput>,
  | "message"
  | "setMessage"
  | "draft"
  | "loading"
  | "isEditMode"
  | "textareaRef"
  | "placeholder"
  | "webSearchEnabled"
  | "enableAttachments"
  | "enableWebSearch"
  | "enableForceTools"
  | "modelSupportsImages"
  | "modelSupportsPDFs"
  | "modelSupportsWebSearch"
  | "modelSupportsToolUse"
  | "handleImageSelect"
  | "handlePDFSelect"
  | "handleForceTool"
  | "forcedTool"
  | "toolsByScenario"
  | "activeTemplate"
  | "selectedSkillIds"
  | "slashCommands"
  | "templateActions"
  | "sendLogic"
  | "handleWebSearchToggle"
  | "handleKeyDown"
  | "chatId"
  | "setWebSearchEnabled"
  | "isMerging"
>;

interface MessageInputAreaProps {
  state: MessageInputAreaState;
  inputTestId: string;
  sendButtonTestId: string;
}

export const MessageInputArea = memo(function MessageInputArea({ state, inputTestId, sendButtonTestId }: MessageInputAreaProps) {
  const {
    message,
    setMessage,
    loading,
    isEditMode,
    textareaRef,
    placeholder,
    webSearchEnabled,
    enableAttachments,
    enableWebSearch,
    enableForceTools,
    modelSupportsImages,
    modelSupportsPDFs,
    modelSupportsWebSearch,
    modelSupportsToolUse,
    handleImageSelect,
    handlePDFSelect,
    handleForceTool,
    forcedTool,
    toolsByScenario,
    activeTemplate,
    selectedSkillIds,
    slashCommands,
    templateActions,
    sendLogic,
    handleWebSearchToggle,
    handleKeyDown,
    chatId,
    setWebSearchEnabled,
    isMerging,
  } = state;
  const {
    handleMessageChangeSlash,
    slashPopupOpen,
  } = slashCommands;
  const [draftValue, setDraftValue] = useState(message);
  const draftValueRef = useRef(draftValue);

  useEffect(() => {
    draftValueRef.current = draftValue;
  }, [draftValue]);

  useEffect(() => {
    setDraftValue(message);
  }, [message]);

  useEffect(() => {
    if (draftValue === message) return;
    const timeout = window.setTimeout(() => {
      setMessage(draftValueRef.current);
    }, 180);
    return () => window.clearTimeout(timeout);
  }, [draftValue, message, setMessage]);

  const canSendDraft = useMemo(() => sendLogic.canSendMessage(draftValue), [draftValue, sendLogic]);

  const flushDraft = useCallback(() => {
    setMessage(draftValueRef.current);
  }, [setMessage]);

  const handleDraftChange = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    const value = e.target.value;
    setDraftValue(value);
    handleMessageChangeSlash(value, e.target.selectionStart);
    if (slashPopupOpen || /(?:^|\s)\/[^\s]*$/.test(value)) {
      setMessage(value);
    }
  }, [handleMessageChangeSlash, setMessage, slashPopupOpen]);

  const handleDraftKeyDown = useCallback((e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && (e.ctrlKey || e.metaKey) && !slashPopupOpen) {
      e.preventDefault();
      flushDraft();
      sendLogic.handleSubmitWithMessage(draftValueRef.current);
      return;
    }
    handleKeyDown(e);
    if (e.key === "Escape" && isEditMode) {
      setDraftValue(state.draft);
    }
  }, [flushDraft, handleKeyDown, isEditMode, sendLogic, slashPopupOpen, state.draft]);

  const handleSendClick = useCallback(() => {
    flushDraft();
    sendLogic.handleSubmitWithMessage(draftValueRef.current);
  }, [flushDraft, sendLogic]);

  return (
    <div className="relative flex items-end gap-1.5 sm:gap-2 p-2 sm:p-3 bg-white/5 border border-white/10 rounded-xl focus-within:ring-2 focus-within:ring-indigo-500/50 focus-within:border-transparent transition-all">
      <AIMergeOverlay
        isOpen={templateActions.showMergeOverlay}
        existingMessage={templateActions.savedMessage}
        templateName={templateActions.pendingTemplate?.name || ""}
        isMerging={isMerging}
        onAction={(action) => { void templateActions.handleMergeAction(action); }}
      />

      {enableAttachments && (
        <AttachmentButton
          onImageSelect={handleImageSelect}
          onPDFSelect={handlePDFSelect}
          webSearchEnabled={webSearchEnabled}
          onWebSearchToggle={
            enableWebSearch ? handleWebSearchToggle : undefined
          }
          disabled={loading}
          modelSupportsImages={modelSupportsImages}
          modelSupportsPDFs={modelSupportsPDFs}
          modelSupportsWebSearch={modelSupportsWebSearch}
          enabledToolsByScenario={
            enableForceTools && chatId ? toolsByScenario : undefined
          }
          forcedTool={forcedTool}
          onForceTool={
            enableForceTools && chatId && modelSupportsToolUse
              ? handleForceTool
              : undefined
          }
          modelSupportsTools={modelSupportsToolUse}
          onOpenTemplateSelector={() =>
            templateActions.setShowTemplateSelector(true)
          }
          onOpenSkillSelector={() =>
            templateActions.setShowSkillSelector(true)
          }
          activeTemplate={activeTemplate?.template}
          selectedSkillCount={selectedSkillIds.length}
        />
      )}

      <div className="relative flex-1">
        <textarea
          ref={textareaRef}
          value={draftValue}
          onChange={handleDraftChange}
          onKeyDown={handleDraftKeyDown}
          placeholder={
            activeTemplate ? "Template variables above..." : placeholder
          }
          disabled={loading}
          rows={1}
          className="w-full bg-transparent text-sm text-white placeholder:text-slate-500 resize-none focus:outline-none disabled:opacity-50 min-h-[36px] sm:min-h-[40px]"
          data-testid={inputTestId}
          aria-label="Message input"
        />

        {slashCommands.slashPopupOpen && (
          <SlashCommandPopup
            commands={slashCommands.filteredSlashCommands}
            selectedIndex={slashCommands.slashSelectedIndex}
            onSelect={(cmd) => {
              if (cmd.type === "search") setWebSearchEnabled(true);
              templateActions.handleSlashCommandSelect(cmd);
              slashCommands.setSlashPopupOpen(false);
            }}
            onClose={() => slashCommands.setSlashPopupOpen(false)}
            position={slashCommands.slashPopupPosition}
          />
        )}
      </div>

      {!loading && draftValue.length > 0 && (
        <span className="hidden sm:inline text-xs text-slate-600 self-end pb-2">
          {draftValue.length}
        </span>
      )}

      <Tooltip content={sendLogic.sendTooltip}>
        <Button
          onClick={handleSendClick}
          disabled={!canSendDraft}
          size="icon"
          className={`h-9 w-9 sm:h-10 sm:w-10 shrink-0 ${isEditMode ? "bg-amber-600 hover:bg-amber-500" : ""}`}
          data-testid={
            isEditMode ? "save-edit-button" : sendButtonTestId
          }
          aria-label={isEditMode ? "Save edit" : "Send message"}
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
  );
});
