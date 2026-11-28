import { RecordingsService } from '@vrooli/proto-types/browser-automation-studio/v1/recordings/recordings_pb';
import { SessionProfilesService } from '@vrooli/proto-types/browser-automation-studio/v1/session_profiles/session_profiles_pb';
import { VisionNavigationService } from '@vrooli/proto-types/browser-automation-studio/v1/ai/ai_pb';
import { CreateRecordingSessionResponseSchema } from '@vrooli/proto-types/browser-automation-studio/v1/recording/session_pb';
import { apiRoute, env, fixtureAfter, rest, rpc, target, withJourneyResources } from './support';

describe('[REQ:BAS-RH-J12] human control at a local authentication challenge', () => {
  it('pauses honestly, lets a person complete the challenge, and recovers that authorized profile', async () => withJourneyResources(async (resources) => {
    let profileId = '';
    let navigationId = '';
      const created = await rpc(SessionProfilesService.method.create, { name: `J12 ${Date.now()}` });
      profileId = created.profile?.id ?? '';
      expect(profileId).not.toBe('');
      if (profileId) resources.add('delete session profile', () => rpc(SessionProfilesService.method.delete, { id: profileId }));

      const opened = await rest(CreateRecordingSessionResponseSchema, '/recordings/live/session', {
        viewport_width: 640,
        viewport_height: 480,
        initial_url: `${env.fixture}/auth-challenge`,
        session_profile_id: profileId,
      });
      let originalSessionOpen = true;
      resources.add('close original recording session', async () => {
        if (originalSessionOpen) await apiRoute(`/recordings/live/session/${opened.sessionId}/close`, {});
      });
      await fixtureAfter('challenge-visit', 0);

      const started = await rpc(VisionNavigationService.method.startNavigation, {
        sessionId: opened.sessionId,
        prompt: 'Continue this authorized fixture sign-in after the person completes its local verification step.',
        model: 'local_first',
        maxSteps: 4,
        navigatorType: 'playwright',
        clientSource: 'api',
      });
      navigationId = started.navigationId;
      expect(navigationId).not.toBe('');
      resources.add('abort navigation', async () => {
        if (navigationId) {
          await rpc(VisionNavigationService.method.abortNavigation, { navigationId });
          navigationId = '';
        }
      });

      const paused = await rpc(VisionNavigationService.method.getNavigationStatus, { navigationId, waitMillis: 10_000n });
      expect(paused.status).toBe('awaiting_human');
      expect(paused.terminal).toBe(true);
      expect(paused.sessionId).toBe(opened.sessionId);
      expect(paused.steps.some((step) => step.actionType === 'request_human')).toBe(true);

      // Use BAS's ordinary pointer-input surface while the navigator is
      // paused. The fixture challenge is synthetic and makes no outside call.
      await apiRoute(`/recordings/live/${opened.sessionId}/input`, {
        type: 'pointer', action: 'click', button: 'left', ...target('challenge'),
      });
      const observed = await fixtureAfter('challenge', 0);
      expect(observed.challenges).toContainEqual(expect.objectContaining({ completedBy: 'person' }));

      const stopped = await rpc(VisionNavigationService.method.abortNavigation, { navigationId });
      expect(stopped.status).toBe('aborting');
      navigationId = '';

      const closed = await apiRoute<{ profile_persisted: boolean }>(`/recordings/live/session/${opened.sessionId}/close`, {});
      expect(closed.profile_persisted).toBe(true);
      originalSessionOpen = false;

      const saved = await rpc(RecordingsService.method.getStorageState, { profileId });
      expect(saved.cookies).toContainEqual(expect.objectContaining({ name: 'bas_j12_authorized', value: 'local-challenge-complete' }));
      const history = await rpc(RecordingsService.method.getHistory, { profileId });
      expect(history.entries.some((entry) => entry.url === `${env.fixture}/auth-challenge`)).toBe(true);
      const tabs = await rpc(RecordingsService.method.getSessionTabs, { profileId });
      expect(tabs.tabs.some((tab) => tab.url === `${env.fixture}/auth-challenge`)).toBe(true);

      const recovered = await rest(CreateRecordingSessionResponseSchema, '/recordings/live/session', {
        viewport_width: 640,
        viewport_height: 480,
        session_profile_id: profileId,
      });
      let recoveredSessionOpen = true;
      resources.add('close recovered recording session', async () => {
        if (recoveredSessionOpen) await apiRoute(`/recordings/live/session/${recovered.sessionId}/close`, {});
      });
      const recoveredVisit = await fixtureAfter('challenge-visit', 1);
      expect(recoveredVisit.challengeVisits[1].context).toContain('bas_j12_authorized=local-challenge-complete');
  }));
});
