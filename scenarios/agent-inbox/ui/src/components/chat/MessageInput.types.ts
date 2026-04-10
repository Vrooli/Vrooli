import type { ForcedTool } from "./AttachmentButton";
import type { Model, Message, UploadResponse } from "../../lib/api";
import type { SkillPayload, Template } from "@/lib/types/templates";

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

export const MESSAGE_INPUT_SUGGESTIONS_EXPANDED_KEY =
  "agent-inbox:message-input-suggestions-expanded";

export function getSuggestionsExpandedDefault(): boolean {
  if (typeof window === "undefined") {
    return false;
  }
  return (
    localStorage.getItem(MESSAGE_INPUT_SUGGESTIONS_EXPANDED_KEY) === "true"
  );
}

export function setSuggestionsExpandedStorage(expanded: boolean): void {
  if (typeof window !== "undefined") {
    localStorage.setItem(
      MESSAGE_INPUT_SUGGESTIONS_EXPANDED_KEY,
      String(expanded),
    );
  }
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface MessagePayload {
  content: string;
  attachmentIds: string[];
  webSearchEnabled: boolean;
  forcedTool?: ForcedTool;
  skillIds?: string[];
  skills?: SkillPayload[]; // Full skill payloads for tool context injection
  suggestedToolIds?: string[]; // Tools suggested by template
}

export interface MessageInputProps {
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
  /** Callback when a template with suggested tools is activated */
  onTemplateActivated?: (
    templateId: string,
    toolIds: string[],
  ) => Promise<void>;
  /** Currently active template ID (persisted state, for UI indicator) */
  activeTemplateId?: string | null;
  /** Callback to deactivate the active template */
  onTemplateDeactivate?: () => void;
  /** Block sending without disabling the textarea (e.g. agent is busy) */
  disableSend?: boolean;
  /** Tooltip reason when disableSend is true */
  disableSendReason?: string;
  /** Custom upload function for attachments (e.g., agent mode uses a different endpoint) */
  customUploadFn?: (file: File) => Promise<UploadResponse>;
}

// Re-export ForcedTool so consumers of this module don't need to reach into AttachmentButton
export type { ForcedTool } from "./AttachmentButton";
export type { Template } from "@/lib/types/templates";
