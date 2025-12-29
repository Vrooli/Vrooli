import { useState, useCallback, useRef, useEffect } from "react";
import { Send, Loader2, Check, X } from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import { AttachmentButton, type ForcedTool } from "./AttachmentButton";
import { AttachmentPreview } from "./AttachmentPreview";
import { WebSearchIndicator } from "./WebSearchIndicator";
import { ForcedToolIndicator } from "./ForcedToolIndicator";
import { useAttachments } from "../../hooks/useAttachments";
import { useTools } from "../../hooks/useTools";
import { supportsImages, supportsPDFs, supportsTools } from "../../lib/modelCapabilities";
import type { Model, Message } from "../../lib/api";

export interface MessagePayload {
  content: string;
  attachmentIds: string[];
  webSearchEnabled: boolean;
  forcedTool?: ForcedTool;
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

  // Edit mode detection
  const isEditMode = !!editingMessage;

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
    const hasContent = trimmedMessage || effectiveAttachments.length > 0;

    // Block send if:
    // - No content (text or attachments)
    // - Still uploading (when attachments enabled)
    // - Has upload errors (when attachments enabled)
    // - Has incompatible attachments
    // - AI is loading/generating
    if (!hasContent || loading) {
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
      content: trimmedMessage,
      attachmentIds: enableAttachments ? getUploadedIds() : [],
      webSearchEnabled: enableWebSearch ? webSearchEnabled : false,
      forcedTool: forcedTool ?? undefined,
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
  ]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
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
    [handleSubmit, isEditMode, onCancelEdit]
  );

  const handleWebSearchToggle = useCallback((enabled: boolean) => {
    setWebSearchEnabled(enabled);
  }, []);

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

  // Model supports tools (required for force tool)
  const modelSupportsToolUse = supportsTools(currentModel);

  // Determine if send button should be disabled
  const hasContent = message.trim() || effectiveAttachments.length > 0;
  const canSend = (() => {
    if (!hasContent || loading) return false;
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

      {/* Input container with buttons inside */}
      <div className="flex items-end gap-2 p-3 bg-white/5 border border-white/10 rounded-xl focus-within:ring-2 focus-within:ring-indigo-500/50 focus-within:border-transparent transition-all">
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
          />
        )}

        {/* Textarea */}
        <textarea
          ref={textareaRef}
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          disabled={loading}
          rows={1}
          className="flex-1 bg-transparent text-sm text-white placeholder:text-slate-500 resize-none focus:outline-none disabled:opacity-50 min-h-[40px]"
          data-testid="message-input"
        />

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
        </div>
        {loading && (
          <span className="text-xs text-indigo-400 flex items-center gap-1">
            <Loader2 className="h-3 w-3 animate-spin" />
            AI is responding...
          </span>
        )}
      </div>
    </div>
  );
}
