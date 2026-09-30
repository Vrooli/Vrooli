import { useCallback } from "react";
import { useNavigate } from "react-router-dom";

import { setWorkspaceIntent, type WorkspaceIntent } from "../workspace/workspaceIntent";

/**
 * Stage a Workspace intent (mode + op + optional starting image) and navigate
 * to `/workspace`. The single path Home tiles, the universal entry, and the
 * sample buttons all use to drop the user into the Workspace pre-set to a task.
 */
export function useOpenInWorkspace(): (intent: WorkspaceIntent) => void {
  const navigate = useNavigate();
  return useCallback(
    (intent: WorkspaceIntent) => {
      setWorkspaceIntent(intent);
      navigate("/workspace");
    },
    [navigate],
  );
}
