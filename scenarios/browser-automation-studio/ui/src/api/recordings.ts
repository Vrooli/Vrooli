import { createClient } from '@connectrpc/connect';
import { RecordingsService } from '@vrooli/proto-types/browser-automation-studio/v1/recordings/recordings_pb';

import { transport } from './client';

/**
 * Connect-Web client for the BAS RecordingsService.
 *
 * Covers every sub-resource hanging off /recordings/sessions/{profileId}/...:
 *   - Storage state (cookies + localStorage)
 *   - Service workers (live, via playwright-driver)
 *   - Browser history (persisted, with playwright-driver re-navigation)
 *   - Saved tabs (for session restoration)
 *
 * Session-profile CRUD itself lives on the sibling SessionProfilesService.
 */
export const recordingsClient = createClient(RecordingsService, transport);

export { RecordingsService };
