/**
 * useMessageInputHandlers - Event handlers for the MessageInput component.
 *
 * Extracted from useMessageInput to keep each module under 300 lines.
 */
import { useCallback } from "react";
import type { useSlashCommands } from "./useSlashCommands";
import type { useTemplateActions } from "./useTemplateActions";
import type { useSendMessage } from "./useSendMessage";

interface UseMessageInputHandlersParams {
  message: string;
  setMessage: (value: string) => void;
  setMessageState: (value: string) => void;
  draft: string;
  isEditMode: boolean;
  onCancelEdit?: () => void;
  setWebSearchEnabled: (enabled: boolean) => void;
  slashCommands: ReturnType<typeof useSlashCommands>;
  templateActions: ReturnType<typeof useTemplateActions>;
  sendLogic: ReturnType<typeof useSendMessage>;
}

export function useMessageInputHandlers({
  setMessage,
  setMessageState,
  draft,
  isEditMode,
  onCancelEdit,
  setWebSearchEnabled,
  slashCommands,
  templateActions,
  sendLogic,
}: UseMessageInputHandlersParams) {
  const handleWebSearchToggle = useCallback((enabled: boolean) => {
    setWebSearchEnabled(enabled);
  }, [setWebSearchEnabled]);

  const handleMessageChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      const value = e.target.value;
      setMessage(value);
      slashCommands.handleMessageChangeSlash(value, e.target.selectionStart);
    },
    [setMessage, slashCommands.handleMessageChangeSlash],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (slashCommands.slashPopupOpen) {
        if (e.key === "ArrowDown") {
          e.preventDefault();
          slashCommands.setSlashSelectedIndex((prev) =>
            prev < slashCommands.filteredSlashCommands.length - 1
              ? prev + 1
              : 0,
          );
          return;
        }
        if (e.key === "ArrowUp") {
          e.preventDefault();
          slashCommands.setSlashSelectedIndex((prev) =>
            prev > 0
              ? prev - 1
              : slashCommands.filteredSlashCommands.length - 1,
          );
          return;
        }
        if (e.key === "Enter") {
          e.preventDefault();
          const cmd =
            slashCommands.filteredSlashCommands[
              slashCommands.slashSelectedIndex
            ];
          if (cmd) {
            if (cmd.type === "search") setWebSearchEnabled(true);
            templateActions.handleSlashCommandSelect(cmd);
            slashCommands.setSlashPopupOpen(false);
          }
          return;
        }
        if (e.key === "Escape") {
          e.preventDefault();
          slashCommands.setSlashPopupOpen(false);
          return;
        }
      }

      if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
        e.preventDefault();
        sendLogic.handleSubmit();
      }
      if (e.key === "Escape" && isEditMode && onCancelEdit) {
        e.preventDefault();
        setMessageState(draft);
        onCancelEdit();
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [
      sendLogic.handleSubmit,
      isEditMode,
      onCancelEdit,
      slashCommands.slashPopupOpen,
      slashCommands.filteredSlashCommands,
      slashCommands.slashSelectedIndex,
      templateActions.handleSlashCommandSelect,
      slashCommands.setSlashPopupOpen,
      slashCommands.setSlashSelectedIndex,
    ],
  );

  return {
    handleWebSearchToggle,
    handleMessageChange,
    handleKeyDown,
  };
}
