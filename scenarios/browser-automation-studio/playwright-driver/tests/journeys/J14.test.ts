import { RecordingsService } from '@vrooli/proto-types/browser-automation-studio/v1/recordings/recordings_pb';
import { SessionProfilesService } from '@vrooli/proto-types/browser-automation-studio/v1/session_profiles/session_profiles_pb';
import { CreateRecordingSessionResponseSchema } from '@vrooli/proto-types/browser-automation-studio/v1/recording/session_pb';
import { apiRoute, env, fixtureAfter, rest, rpc, withJourneyResources } from './support';

describe('[REQ:BAS-RH-J14] encrypted profile recovery', () => {
  it('recovers protected cookies, local storage, history and tabs from the last committed profile', async () => withJourneyResources(async (resources) => {
    let profileId = '';
      const created = await rpc(SessionProfilesService.method.create, { name: `J14 ${Date.now()}` });
      profileId = created.profile?.id ?? '';
      expect(profileId).not.toBe('');
      if (profileId) resources.add('delete session profile', () => rpc(SessionProfilesService.method.delete, { id: profileId }));

      const original = await rest(CreateRecordingSessionResponseSchema, '/recordings/live/session', {
        viewport_width: 640,
        viewport_height: 480,
        initial_url: `${env.fixture}/encrypted-profile`,
        session_profile_id: profileId,
      });
      let originalSessionOpen = true;
      resources.add('close original recording session', async () => {
        if (originalSessionOpen) await apiRoute(`/recordings/live/session/${original.sessionId}/close`, {});
      });
      const firstVisit = await fixtureAfter('profile-observation', 0);
      expect(firstVisit.profileObservations[0].localStorage).toBe('saved-local-state');

      const closed = await apiRoute<{ profile_persisted: boolean }>(`/recordings/live/session/${original.sessionId}/close`, {});
      expect(closed.profile_persisted).toBe(true);
      originalSessionOpen = false;

      const saved = await rpc(RecordingsService.method.getStorageState, { profileId });
      expect(saved.cookies).toContainEqual(expect.objectContaining({ name: 'bas_j14_secret', valueMasked: true, value: '[HIDDEN]' }));
      expect(saved.stats?.cookieCount).toBeGreaterThan(0);
      expect(saved.origins.some((origin) => origin.localStorage.some((item) => item.name === 'bas_j14_local' && item.value === 'saved-local-state'))).toBe(true);
      const profileList = await rpc(SessionProfilesService.method.list, {});
      expect(profileList.profiles).toContainEqual(expect.objectContaining({ id: profileId, hasStorageState: true }));

      const history = await rpc(RecordingsService.method.getHistory, { profileId });
      expect(history.entries.some((entry) => entry.url === `${env.fixture}/encrypted-profile`)).toBe(true);
      const tabs = await rpc(RecordingsService.method.getSessionTabs, { profileId });
      expect(tabs.tabs.some((tab) => tab.url === `${env.fixture}/encrypted-profile` && tab.isActive)).toBe(true);

      const recovered = await rest(CreateRecordingSessionResponseSchema, '/recordings/live/session', {
        viewport_width: 640,
        viewport_height: 480,
        session_profile_id: profileId,
      });
      resources.add('close recovered recording session', () => apiRoute(`/recordings/live/session/${recovered.sessionId}/close`, {}));
      const recoveryVisit = await fixtureAfter('profile-observation', 1);
      expect(recoveryVisit.profileObservations[1].context).toContain('bas_j14_secret=sealed-profile-value');
      expect(recoveryVisit.profileObservations[1].localStorage).toBe('saved-local-state');
  }));
});
