import { createClient } from '@connectrpc/connect';
import { ProjectsService } from '@vrooli/proto-types/browser-automation-studio/v1/projects/project_pb';

import { transport } from './client';

export const projectsClient = createClient(ProjectsService, transport);
