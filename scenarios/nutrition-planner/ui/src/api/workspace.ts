import { createClient } from "@connectrpc/connect";
import { WorkspaceService, type Workspace } from "@vrooli/proto-types/nutrition-planner/v1/workspace/workspace_pb";
import { transport } from "./client";

const client = createClient(WorkspaceService, transport);

export async function ensureWorkspace(): Promise<Workspace> {
  const listed = await client.listWorkspaces({});
  const existing = listed.workspaces[0];
  if (existing) return existing;
  const created = await client.createWorkspace({ name: "My Daily workspace", idempotencyKey: crypto.randomUUID() });
  if (!created.workspace) throw new Error("The API returned no workspace.");
  return created.workspace;
}
