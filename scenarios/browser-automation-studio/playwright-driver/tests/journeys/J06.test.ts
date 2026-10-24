import { randomUUID } from 'node:crypto';
import { SessionProfilesService } from '@vrooli/proto-types/browser-automation-studio/v1/session_profiles/session_profiles_pb';
import { rpc, withJourneyResources } from './support';

describe('[REQ:BAS-RH-J06] saved profile survives rejected persistence writes', () => {
  it('keeps the acknowledged profile when a later write is rejected', async () => withJourneyResources(async (resources) => {
    const created = await rpc(SessionProfilesService.method.create, { name: `J06 ${randomUUID()}` });
    const id = created.profile?.id ?? '';
    expect(id).not.toBe('');
    if (id) resources.add('delete session profile', () => rpc(SessionProfilesService.method.delete, { id }));
      const saved = await rpc(SessionProfilesService.method.update, {
        id, name: 'J06 recoverable profile', browserProfile: { locale: 'en-US' },
      });
      expect(saved.profile).toMatchObject({ id, name: 'J06 recoverable profile' });

      // A failed mutation must not be acknowledged or replace the prior save.
      await expect(rpc(SessionProfilesService.method.update, {
        id: `missing-${randomUUID()}`, name: 'must not be saved',
      })).rejects.toThrow();
      const listed = await rpc(SessionProfilesService.method.list, {});
      expect(listed.profiles).toContainEqual(expect.objectContaining({ id, name: 'J06 recoverable profile' }));
      expect(listed.profiles).not.toContainEqual(expect.objectContaining({ id, name: 'must not be saved' }));
  }));
});
