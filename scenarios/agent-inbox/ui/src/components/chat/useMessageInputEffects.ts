/**
 * useMessageInputEffects - Textarea side-effects and model capability checks.
 *
 * Extracted from useMessageInput to keep each module under 300 lines.
 */
import { useEffect, useMemo } from "react";
import {
  supportsImages,
  supportsPDFs,
  supportsTools,
} from "../../lib/modelCapabilities";
import type { AttachmentState } from "../../hooks/useAttachments";
import type { Model } from "../../lib/api";

interface UseTextareaEffectsParams {
  message: string;
  autoFocus: boolean;
  editingMessage: { content: string } | null | undefined;
  draft: string;
  setMessageState: (value: string) => void;
  textareaRef: React.RefObject<HTMLTextAreaElement | null>;
}

/** Auto-resize textarea, handle autoFocus, and sync editing state. */
export function useTextareaEffects({
  message,
  autoFocus,
  editingMessage,
  draft,
  setMessageState,
  textareaRef,
}: UseTextareaEffectsParams) {
  useEffect(() => {
    const textarea = textareaRef.current;
    if (textarea) {
      textarea.style.height = "auto";
      textarea.style.height = `${Math.min(textarea.scrollHeight, 200)}px`;
    }
  }, [message, textareaRef]);

  useEffect(() => {
    if (autoFocus && textareaRef.current) {
      textareaRef.current.focus();
    }
  }, [autoFocus, textareaRef]);

  useEffect(() => {
    if (editingMessage) {
      setMessageState(editingMessage.content);
      if (textareaRef.current) {
        textareaRef.current.focus();
        textareaRef.current.setSelectionRange(
          editingMessage.content.length,
          editingMessage.content.length,
        );
      }
    } else {
      setMessageState(draft);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editingMessage]);
}

interface UseModelCapabilitiesParams {
  enableAttachments: boolean;
  enableWebSearch: boolean;
  currentModel: Model | null;
  attachments: AttachmentState[];
  chatWebSearchDefault: boolean;
  setWebSearchEnabled: (enabled: boolean) => void;
}

export interface ModelCapabilities {
  effectiveAttachments: AttachmentState[];
  modelSupportsImages: boolean;
  modelSupportsPDFs: boolean;
  modelSupportsWebSearch: boolean;
  modelSupportsToolUse: boolean;
  hasIncompatibleAttachments: boolean;
}

/** Compute model capabilities and sync web search default. */
export function useModelCapabilities({
  enableAttachments,
  enableWebSearch,
  currentModel,
  attachments,
  chatWebSearchDefault,
  setWebSearchEnabled,
}: UseModelCapabilitiesParams): ModelCapabilities {
  const effectiveAttachments = useMemo(
    () => (enableAttachments ? attachments : []),
    [enableAttachments, attachments],
  );
  const modelSupportsImages = enableAttachments && supportsImages(currentModel);
  const modelSupportsPDFs = enableAttachments && supportsPDFs(currentModel);
  const modelSupportsWebSearch = enableWebSearch && supportsTools(currentModel);
  const modelSupportsToolUse = supportsTools(currentModel);

  useEffect(() => {
    if (enableWebSearch) {
      if (!modelSupportsWebSearch) {
        setWebSearchEnabled(false);
      } else {
        setWebSearchEnabled(chatWebSearchDefault);
      }
    }
  }, [chatWebSearchDefault, enableWebSearch, modelSupportsWebSearch, setWebSearchEnabled]);

  const hasIncompatibleAttachments = effectiveAttachments.some((att) => {
    if (att.type === "image" && !modelSupportsImages) return true;
    if (att.type === "pdf" && !modelSupportsPDFs) return true;
    return false;
  });

  return {
    effectiveAttachments,
    modelSupportsImages,
    modelSupportsPDFs,
    modelSupportsWebSearch,
    modelSupportsToolUse,
    hasIncompatibleAttachments,
  };
}
