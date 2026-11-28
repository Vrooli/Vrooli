import { useCallback, type Dispatch, type SetStateAction } from "react";

import type { Workspace } from "../../api/workspaces";
import { jumpCommand, redoCommand, undoCommand, type CommandStack } from "../studio/history";

interface StudioModel {
  workspace: Workspace | null;
  history: CommandStack<Workspace>;
}

/** Owns history navigation and clears node selection after restoring a snapshot. */
export function useWorkspaceHistory(
  setModel: Dispatch<SetStateAction<StudioModel>>,
  setSelectedId: Dispatch<SetStateAction<string | null>>,
) {
  const undo = useCallback(() => {
    setModel((current) => {
      const result = undoCommand(current.history);
      return result ? { workspace: structuredClone(result.state), history: result.stack } : current;
    });
    setSelectedId(null);
  }, [setModel, setSelectedId]);

  const redo = useCallback(() => {
    setModel((current) => {
      const result = redoCommand(current.history);
      return result ? { workspace: structuredClone(result.state), history: result.stack } : current;
    });
    setSelectedId(null);
  }, [setModel, setSelectedId]);

  const jump = useCallback(
    (position: number) => {
      setModel((current) => {
        const result = jumpCommand(current.history, position);
        return result ? { workspace: structuredClone(result.state), history: result.stack } : current;
      });
      setSelectedId(null);
    },
    [setModel, setSelectedId],
  );

  return { undo, redo, jump };
}
