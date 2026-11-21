import { createClient } from '@connectrpc/connect';
import { SessionProfilesService } from '@vrooli/proto-types/browser-automation-studio/v1/session_profiles/session_profiles_pb';

import { transport } from './client';

export const sessionProfilesClient = createClient(SessionProfilesService, transport);
