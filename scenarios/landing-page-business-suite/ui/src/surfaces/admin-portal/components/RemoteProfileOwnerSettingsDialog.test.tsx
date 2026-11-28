import { renderWithProviders as render } from '@vrooli/api-base/testing';
import { describe, expect, it, vi } from 'vitest';
import { RemoteProfileOwnerSettingsDialog } from './RemoteProfileOwnerSettingsDialog';
import { expectActionBarOrder } from '../../../test/overlayActionBarAssertions';

vi.mock('../../../shared/api', () => ({
  listRemoteAPIKeysAdmin: vi.fn().mockResolvedValue({ keys: [] }),
  getRemoteStripeSettingsAdmin: vi.fn().mockResolvedValue({ dashboard_url: '', publishable_key_set: false, secret_key_set: false, webhook_secret_set: false }),
}));

describe('RemoteProfileOwnerSettingsDialog action contract', () => {
  it('marks Close primary and keeps Refresh secondary', async () => {
    render(<RemoteProfileOwnerSettingsDialog profile={{
      id: 4, tag: 'qa-target', label: 'QA target', api_base: 'https://qa.example.test', status: 'active', has_session: true, created_at: '2026-11-19T00:00:00Z', updated_at: '2026-11-19T00:00:00Z',
    }} open onOpenChange={vi.fn()} />);

    expectActionBarOrder('close', 'refresh');
  });
});
