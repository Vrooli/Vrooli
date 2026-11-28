import { createClient } from "@connectrpc/connect";
import { ProjectsService } from "@vrooli/proto-types/content-desk/v1/projects/projects_pb";
import { transport } from "./client";

export const projectsClient = createClient(ProjectsService, transport);
