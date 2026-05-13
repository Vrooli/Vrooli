import { createClient } from "@connectrpc/connect";
import { WorkspaceService } from "@vrooli/proto-types/web-console/v1/workspace/workspace_pb";

import { transport } from "./client";

// workspaceClient is the Connect-Web client for WorkspaceService.
// UI code imports this directly; the legacy fetch helpers in lib/api.ts
// are shims that delegate here and normalize the camelCase proto shape
// to the existing snake_case wire types.
export const workspaceClient = createClient(WorkspaceService, transport);
