import { useCallback } from "react";
import type { ForcedTool } from "./AttachmentButton";
import type { MessagePayload } from "./MessageInput.types";
import type { AttachmentState } from "../../hooks/useAttachments";

interface UseSendMessageOptions {
  message: string;
  setMessageState: (value: string) => void;
  clearDraft: () => void;
  effectiveAttachments: AttachmentState[];
  enableAttachments: boolean;
  enableWebSearch: boolean;
  isUploading: boolean;
  hasErrors: boolean;
  hasIncompatibleAttachments: boolean;
  allUploaded: boolean;
  getUploadedIds: () => string[];
  clearAttachments: () => void;
  webSearchEnabled: boolean;
  setWebSearchEnabled: (enabled: boolean) => void;
  chatWebSearchDefault: boolean;
  forcedTool: ForcedTool | null;
  setForcedTool: (tool: ForcedTool | null) => void;
  loading: boolean;
  isEditMode: boolean;
  onSend: (payload: MessagePayload) => void;
  onSubmitEdit?: (payload: MessagePayload) => void;
  activeTemplate: { template: { suggestedToolIds?: string[] } } | null;
  getFilledTemplateContent: () => string;
  isTemplateValid: () => boolean;
  getTemplateMissingFields: () => string[];
  selectedSkillIds: string[];
  buildSkillPayloads: (ids: string[]) => MessagePayload["skills"];
  resetTemplatesAndSkills: () => void;
  textareaRef: React.RefObject<HTMLTextAreaElement | null>;
  disableSend?: boolean;
  disableSendReason?: string;
}

export function useSendMessage({
  message,
  setMessageState,
  clearDraft,
  effectiveAttachments,
  enableAttachments,
  enableWebSearch,
  isUploading,
  hasErrors,
  hasIncompatibleAttachments,
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
}: UseSendMessageOptions) {
  const handleSubmit = useCallback(() => {
    const trimmedMessage = message.trim();
    const finalContent = activeTemplate
      ? getFilledTemplateContent()
      : trimmedMessage;

    const hasContent = finalContent.trim() || effectiveAttachments.length > 0;

    if (!hasContent || loading) return;
    if (activeTemplate && !isTemplateValid()) return;

    if (enableAttachments) {
      if (isUploading || hasErrors || hasIncompatibleAttachments) return;
      if (effectiveAttachments.length > 0 && !allUploaded) return;
    }

    const payload: MessagePayload = {
      content: finalContent.trim(),
      attachmentIds: enableAttachments ? getUploadedIds() : [],
      webSearchEnabled: enableWebSearch ? webSearchEnabled : false,
      forcedTool: forcedTool ?? undefined,
      skillIds: selectedSkillIds.length > 0 ? selectedSkillIds : undefined,
      skills:
        selectedSkillIds.length > 0
          ? buildSkillPayloads(selectedSkillIds)
          : undefined,
      suggestedToolIds: activeTemplate?.template.suggestedToolIds,
    };

    if (isEditMode && onSubmitEdit) {
      onSubmitEdit(payload);
    } else {
      onSend(payload);
    }

    setMessageState("");
    clearDraft();
    if (enableAttachments) clearAttachments();
    if (enableWebSearch) setWebSearchEnabled(chatWebSearchDefault);
    setForcedTool(null);
    resetTemplatesAndSkills();

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
    clearDraft,
    chatWebSearchDefault,
    enableAttachments,
    enableWebSearch,
    isEditMode,
    onSubmitEdit,
    activeTemplate,
    getFilledTemplateContent,
    isTemplateValid,
    selectedSkillIds,
    buildSkillPayloads,
    resetTemplatesAndSkills,
    setMessageState,
    setWebSearchEnabled,
    setForcedTool,
    textareaRef,
  ]);

  // Determine if send button should be disabled
  const finalContent = activeTemplate ? getFilledTemplateContent() : message;
  const hasContent = finalContent.trim() || effectiveAttachments.length > 0;
  const canSend = (() => {
    if (!hasContent || loading || disableSend) return false;
    if (activeTemplate && !isTemplateValid()) return false;
    if (enableAttachments) {
      if (isUploading || hasErrors || hasIncompatibleAttachments) return false;
      if (effectiveAttachments.length > 0 && !allUploaded) return false;
    }
    return true;
  })();

  // Build send button tooltip
  const modKey = (() => {
    if (typeof navigator === "undefined") return "Ctrl";
    const nav = navigator as unknown as { platform?: string };
    return nav.platform?.includes("Mac") ? "\u2318" : "Ctrl";
  })();
  let sendTooltip = isEditMode
    ? `Save edit (${modKey}+Enter)`
    : `Send message (${modKey}+Enter)`;
  if (loading) {
    sendTooltip = "AI is responding...";
  } else if (disableSend) {
    sendTooltip = disableSendReason || "Sending is temporarily disabled";
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

  return { handleSubmit, canSend, sendTooltip, modKey };
}
