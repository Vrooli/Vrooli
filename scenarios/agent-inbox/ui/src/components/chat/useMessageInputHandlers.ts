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
  const {
    filteredSlashCommands,
    handleMessageChangeSlash,
    setSlashPopupOpen,
    setSlashSelectedIndex,
    slashPopupOpen,
    slashSelectedIndex,
  } = slashCommands;
  const { handleSlashCommandSelect } = templateActions;
  const { handleSubmit } = sendLogic;

  const handleWebSearchToggle = useCallback((enabled: boolean) => {
    setWebSearchEnabled(enabled);
  }, [setWebSearchEnabled]);

  const handleMessageChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      const value = e.target.value;
      setMessage(value);
      handleMessageChangeSlash(value, e.target.selectionStart);
    },
    [handleMessageChangeSlash, setMessage],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (slashPopupOpen) {
        if (e.key === "ArrowDown") {
          e.preventDefault();
          setSlashSelectedIndex((prev) =>
            prev < filteredSlashCommands.length - 1
              ? prev + 1
              : 0,
          );
          return;
        }
        if (e.key === "ArrowUp") {
          e.preventDefault();
          setSlashSelectedIndex((prev) =>
            prev > 0
              ? prev - 1
              : filteredSlashCommands.length - 1,
          );
          return;
        }
        if (e.key === "Enter") {
          e.preventDefault();
          const cmd =
            filteredSlashCommands[
              slashSelectedIndex
            ];
          if (cmd) {
            if (cmd.type === "search") setWebSearchEnabled(true);
            handleSlashCommandSelect(cmd);
            setSlashPopupOpen(false);
          }
          return;
        }
        if (e.key === "Escape") {
          e.preventDefault();
          setSlashPopupOpen(false);
          return;
        }
      }

      if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
        e.preventDefault();
        handleSubmit();
      }
      if (e.key === "Escape" && isEditMode && onCancelEdit) {
        e.preventDefault();
        setMessageState(draft);
        onCancelEdit();
      }
    },
    [
      handleSubmit,
      isEditMode,
      onCancelEdit,
      slashPopupOpen,
      filteredSlashCommands,
      slashSelectedIndex,
      handleSlashCommandSelect,
      setSlashPopupOpen,
      setSlashSelectedIndex,
      setMessageState,
      draft,
      setWebSearchEnabled,
    ],
  );

  return {
    handleWebSearchToggle,
    handleMessageChange,
    handleKeyDown,
  };
}
