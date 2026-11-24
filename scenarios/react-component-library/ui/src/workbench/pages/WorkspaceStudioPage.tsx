import { WorkspaceStudioPageContent } from "./WorkspaceStudioPageContent";

export function WorkspaceStudioPage({ workspaceId }: { workspaceId: string }) {
  return <WorkspaceStudioPageContent workspaceId={workspaceId} />;
}
