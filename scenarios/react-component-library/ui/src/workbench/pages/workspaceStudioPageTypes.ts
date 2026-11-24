import type { Workspace, WorkspaceDisplay, WorkspacePage } from "../../api/workspaces";
import { useWorkspaceStudio } from "../studio/useWorkspaceStudio";

export type LoadedWorkspaceStudio = ReturnType<typeof useWorkspaceStudio> & {
  workspace: Workspace;
  page: WorkspacePage;
  display: WorkspaceDisplay;
};
