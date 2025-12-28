import { useState, useCallback, useRef, useEffect } from "react";
import { Send, Loader2 } from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import { AttachmentButton } from "./AttachmentButton";
import { AttachmentPreview } from "./AttachmentPreview";
import { WebSearchIndicator } from "./WebSearchIndicator";
import { useAttachments } from "../../hooks/useAttachments";
import { supportsImages, supportsPDFs, supportsTools } from "../../lib/modelCapabilities";
import type { Model } from "../../lib/api";

export interface MessagePayload {
  content: string;
  attachmentIds: string[];
  webSearchEnabled: boolean;
}

interface MessageInputProps {
  onSend: (payload: MessagePayload) => void;
  isLoading?: boolean;
  placeholder?: string;
  /** Enable attachment support (images, PDFs). Requires currentModel. Default: true */
  enableAttachments?: boolean;
  /** Enable web search toggle. Requires chatWebSearchDefault. Default: true */
  enableWebSearch?: boolean;
  /** Auto-focus the textarea on mount. Default: false */
  autoFocus?: boolean;
  currentModel?: Model | null;
  chatWebSearchDefault?: boolean;
  onChatWebSearchDefaultChange?: (enabled: boolean) => void;
  /** @deprecated Use isLoading instead */
  isGenerating?: boolean;
}

export function MessageInput({
  onSend,
  isLoading,
  isGenerating,
  placeholder = "Type a message...",
  enableAttachments = true,
  enableWebSearch = true,
  autoFocus = false,
  currentModel = null,
  chatWebSearchDefault = false,
  onChatWebSearchDefaultChange,
}: MessageInputProps) {
  // Support both isLoading and deprecated isGenerating
  const loading = isLoading ?? isGenerating ?? false;
  const [message, setMessage] = useState("");
  const [webSearchEnabled, setWebSearchEnabled] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

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
    };

    onSend(payload);
    setMessage("");
    if (enableAttachments) {
      clearAttachments();
    }
    // Reset web search to chat default after sending
    if (enableWebSearch) {
      setWebSearchEnabled(chatWebSearchDefault);
    }

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
    onSend,
    clearAttachments,
    chatWebSearchDefault,
    enableAttachments,
    enableWebSearch,
  ]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        handleSubmit();
      }
    },
    [handleSubmit]
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
  let sendTooltip = "Send message (Enter)";
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

        {/* Send Button (inside, on right) */}
        <Tooltip content={sendTooltip}>
          <Button
            onClick={handleSubmit}
            disabled={!canSend}
            size="icon"
            className="h-10 w-10 shrink-0"
            data-testid="send-message-button"
          >
            {loading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
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
            Press <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-slate-400">Enter</kbd> to
            send, <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-slate-400">Shift+Enter</kbd>{" "}
            for new line
          </p>
          {enableWebSearch && modelSupportsWebSearch && (
            <WebSearchIndicator
              enabled={webSearchEnabled}
              onDisable={() => setWebSearchEnabled(false)}
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
