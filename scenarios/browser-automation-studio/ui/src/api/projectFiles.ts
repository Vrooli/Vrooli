import { createClient } from '@connectrpc/connect';
import { ProjectFilesService } from '@vrooli/proto-types/browser-automation-studio/v1/project_files/project_files_pb';

import { transport } from './client';

export const projectFilesClient = createClient(ProjectFilesService, transport);
