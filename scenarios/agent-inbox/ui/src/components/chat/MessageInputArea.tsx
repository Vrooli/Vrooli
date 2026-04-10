/**
 * MessageInputArea - The core textarea, attachment button, slash popup, and send button.
 *
 * Extracted from MessageInput to keep each module under 300 lines.
 */
import { Send, Loader2, Check } from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import { AttachmentButton } from "./AttachmentButton";
import { AIMergeOverlay } from "./AIMergeOverlay";
import { SlashCommandPopup } from "./SlashCommandPopup";
import type { useMessageInput } from "./useMessageInput";

interface MessageInputAreaProps {
  state: ReturnType<typeof useMessageInput>;
  inputTestId: string;
  sendButtonTestId: string;
}

export function MessageInputArea({ state, inputTestId, sendButtonTestId }: MessageInputAreaProps) {
  const {
    message,
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
    handleMessageChange,
    handleKeyDown,
    chatId,
    setWebSearchEnabled,
    isMerging,
  } = state;

  return (
    <div className="relative flex items-end gap-1.5 sm:gap-2 p-2 sm:p-3 bg-white/5 border border-white/10 rounded-xl focus-within:ring-2 focus-within:ring-indigo-500/50 focus-within:border-transparent transition-all">
      <AIMergeOverlay
        isOpen={templateActions.showMergeOverlay}
        existingMessage={templateActions.savedMessage}
        templateName={templateActions.pendingTemplate?.name || ""}
        isMerging={isMerging}
        onAction={templateActions.handleMergeAction}
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
          value={message}
          onChange={handleMessageChange}
          onKeyDown={handleKeyDown}
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

      {!loading && message.length > 0 && (
        <span className="hidden sm:inline text-xs text-slate-600 self-end pb-2">
          {message.length}
        </span>
      )}

      <Tooltip content={sendLogic.sendTooltip}>
        <Button
          onClick={sendLogic.handleSubmit}
          disabled={!sendLogic.canSend}
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
}
