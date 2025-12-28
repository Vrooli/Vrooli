import { useState, useCallback, useRef, useEffect } from "react";
import { Send, Loader2 } from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import { AttachmentButton } from "./AttachmentButton";
import { AttachmentPreview } from "./AttachmentPreview";
import { WebSearchIndicator } from "./WebSearchIndicator";
import { useAttachments } from "../../hooks/useAttachments";
import { supportsImages, supportsPDFs } from "../../lib/modelCapabilities";
import type { Model } from "../../lib/api";

export interface MessagePayload {
  content: string;
  attachmentIds: string[];
  webSearchEnabled: boolean;
}

interface MessageInputProps {
  onSend: (payload: MessagePayload) => void;
  isGenerating: boolean;
  placeholder?: string;
  currentModel: Model | null;
  chatWebSearchDefault: boolean;
  onChatWebSearchDefaultChange?: (enabled: boolean) => void;
}

export function MessageInput({
  onSend,
  isGenerating,
  placeholder = "Type a message...",
  currentModel,
  chatWebSearchDefault,
  onChatWebSearchDefaultChange,
}: MessageInputProps) {
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

  // Reset web search to chat default when it changes
  useEffect(() => {
    setWebSearchEnabled(chatWebSearchDefault);
  }, [chatWebSearchDefault]);

  // Model capabilities
  const modelSupportsImages = supportsImages(currentModel);
  const modelSupportsPDFs = supportsPDFs(currentModel);

  // Check if any attachments are incompatible with the model
  const hasIncompatibleAttachments = attachments.some((att) => {
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

  const handleSubmit = useCallback(() => {
    const trimmedMessage = message.trim();
    const hasContent = trimmedMessage || attachments.length > 0;

    // Block send if:
    // - No content (text or attachments)
    // - Still uploading
    // - Has upload errors
    // - Has incompatible attachments
    // - AI is generating
    if (!hasContent || isUploading || hasErrors || hasIncompatibleAttachments || isGenerating) {
      return;
    }

    // If attachments exist but not all uploaded, wait
    if (attachments.length > 0 && !allUploaded) {
      return;
    }

    const payload: MessagePayload = {
      content: trimmedMessage,
      attachmentIds: getUploadedIds(),
      webSearchEnabled,
    };

    onSend(payload);
    setMessage("");
    clearAttachments();
    // Reset web search to chat default after sending
    setWebSearchEnabled(chatWebSearchDefault);

    // Reset height
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
    }
  }, [
    message,
    attachments,
    isUploading,
    hasErrors,
    hasIncompatibleAttachments,
    isGenerating,
    allUploaded,
    getUploadedIds,
    webSearchEnabled,
    onSend,
    clearAttachments,
    chatWebSearchDefault,
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
  const hasContent = message.trim() || attachments.length > 0;
  const canSend =
    hasContent &&
    !isGenerating &&
    !isUploading &&
    !hasErrors &&
    !hasIncompatibleAttachments &&
    (attachments.length === 0 || allUploaded);

  // Build send button tooltip
  let sendTooltip = "Send message (Enter)";
  if (isGenerating) {
    sendTooltip = "AI is responding...";
  } else if (isUploading) {
    sendTooltip = "Uploading attachments...";
  } else if (hasErrors) {
    sendTooltip = "Fix attachment errors before sending";
  } else if (hasIncompatibleAttachments) {
    sendTooltip = "Remove attachments not supported by this model";
  }

  return (
    <div className="p-4 border-t border-white/10 bg-slate-950/50" data-testid="message-input-container">
      {/* Attachment preview area */}
      {attachments.length > 0 && (
        <div className="mb-2">
          <AttachmentPreview
            attachments={attachments}
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

      <div className="flex items-end gap-2">
        {/* Attachment Button */}
        <AttachmentButton
          onImageSelect={handleImageSelect}
          onPDFSelect={handlePDFSelect}
          webSearchEnabled={webSearchEnabled}
          onWebSearchToggle={handleWebSearchToggle}
          disabled={isGenerating}
          modelSupportsImages={modelSupportsImages}
          modelSupportsPDFs={modelSupportsPDFs}
        />

        {/* Input Area */}
        <div className="flex-1 relative">
          <textarea
            ref={textareaRef}
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            disabled={isGenerating}
            rows={1}
            className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3 pr-12 text-sm text-white placeholder:text-slate-500 resize-none focus:outline-none focus:ring-2 focus:ring-indigo-500/50 disabled:opacity-50"
            data-testid="message-input"
          />
          <div className="absolute right-3 bottom-3 flex items-center gap-1">
            {!isGenerating && message.length > 0 && (
              <span className="text-xs text-slate-600">{message.length}</span>
            )}
          </div>
        </div>

        {/* Send Button */}
        <Tooltip content={sendTooltip}>
          <Button
            onClick={handleSubmit}
            disabled={!canSend}
            size="icon"
            className="h-11 w-11 shrink-0"
            data-testid="send-message-button"
          >
            {isGenerating ? (
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
          <WebSearchIndicator
            enabled={webSearchEnabled}
            onDisable={() => setWebSearchEnabled(false)}
          />
        </div>
        {isGenerating && (
          <span className="text-xs text-indigo-400 flex items-center gap-1">
            <Loader2 className="h-3 w-3 animate-spin" />
            AI is responding...
          </span>
        )}
      </div>
    </div>
  );
}
