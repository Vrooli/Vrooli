import { useState, useCallback, useMemo, useEffect } from "react";
import type { SlashCommand } from "@/lib/types/templates";

export interface SlashCommandState {
  slashPopupOpen: boolean;
  slashQuery: string;
  slashSelectedIndex: number;
  slashPopupPosition: { bottom: number; left: number };
  filteredSlashCommands: SlashCommand[];
}

export interface SlashCommandActions {
  setSlashPopupOpen: (open: boolean) => void;
  setSlashSelectedIndex: React.Dispatch<React.SetStateAction<number>>;
  handleMessageChangeSlash: (value: string, cursorPosition: number) => void;
}

interface UseSlashCommandsOptions {
  filterCommands: (query: string) => SlashCommand[];
}

export function useSlashCommands({ filterCommands }: UseSlashCommandsOptions) {
  const [slashPopupOpen, setSlashPopupOpen] = useState(false);
  const [slashQuery, setSlashQuery] = useState("");
  const [slashSelectedIndex, setSlashSelectedIndex] = useState(0);
  const [slashPopupPosition, setSlashPopupPosition] = useState({
    bottom: 60,
    left: 0,
  });

  const filteredSlashCommands = useMemo(
    () => filterCommands(slashQuery),
    [filterCommands, slashQuery],
  );

  // Auto-close slash popup when no results and user has typed something
  useEffect(() => {
    if (
      slashPopupOpen &&
      slashQuery.length > 0 &&
      filteredSlashCommands.length === 0
    ) {
      setSlashPopupOpen(false);
    }
  }, [slashPopupOpen, slashQuery, filteredSlashCommands.length]);

  /** Call this from the textarea onChange to detect slash commands. */
  const handleMessageChangeSlash = useCallback(
    (value: string, cursorPosition: number) => {
      const textBeforeCursor = value.slice(0, cursorPosition);
      const lastNewlineIndex = textBeforeCursor.lastIndexOf("\n");
      const lineStart = lastNewlineIndex + 1;
      const lineBeforeCursor = textBeforeCursor.slice(lineStart);

      // Match "/" at start of line followed by optional word characters
      const slashMatch = lineBeforeCursor.match(/^\/(\S*)$/);

      if (slashMatch) {
        const query = slashMatch[1] ?? "";
        setSlashQuery(query);
        setSlashPopupOpen(true);
        setSlashSelectedIndex(0);
        setSlashPopupPosition({ bottom: 60, left: 8 });
      } else {
        setSlashPopupOpen(false);
      }
    },
    [],
  );

  return {
    slashPopupOpen,
    slashQuery,
    slashSelectedIndex,
    slashPopupPosition,
    filteredSlashCommands,
    setSlashPopupOpen,
    setSlashSelectedIndex,
    handleMessageChangeSlash,
  };
}
