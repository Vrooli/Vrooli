import { buildActivityNodeId, buildBacklogNodeId } from "../../surfaces/graph/lib/node-id-parser";
import type { AgentSessionArtifact } from "../../types";

export function nodeIdForSessionArtifact(artifact: Pick<AgentSessionArtifact, "artifactType" | "entityRef">): string | null {
  const ref = artifact.entityRef?.trim();
  if (!ref) return null;

  switch (artifact.artifactType) {
    case "backlog_item": {
      const slashIndex = ref.indexOf("/");
      if (slashIndex <= 0 || slashIndex === ref.length - 1) return null;
      return buildBacklogNodeId(ref.slice(0, slashIndex), ref.slice(slashIndex + 1));
    }
    case "capture":
      return `capture/${ref}`;
    case "agent_activity":
      return buildActivityNodeId(ref);
    default:
      return null;
  }
}
