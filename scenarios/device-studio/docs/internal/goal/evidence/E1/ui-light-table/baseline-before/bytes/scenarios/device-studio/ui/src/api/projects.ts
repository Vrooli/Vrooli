import { ConnectError } from "@connectrpc/connect";
import { createClient } from "@connectrpc/connect";
import {
  ProjectsService,
  ProjectConflictSchema,
  type Project,
} from "@vrooli/proto-types/device-studio/v1/projects/projects_pb";
import { transport } from "./client";

export const projectsClient = createClient(ProjectsService, transport);

export function projectConflict(error: unknown) {
  if (!(error instanceof ConnectError)) return undefined;
  return error.findDetails(ProjectConflictSchema)[0];
}

export type { Project };
