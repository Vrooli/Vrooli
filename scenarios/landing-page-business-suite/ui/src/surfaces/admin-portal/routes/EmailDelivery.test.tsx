import { describe, expect, it, vi, beforeEach } from 'vitest';
import { renderWithProviders as render } from '@vrooli/api-base/testing';
import { screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { EmailDelivery } from './EmailDelivery';
import * as emailAPI from '../../../shared/api/emailReadiness';

vi.mock('../../../shared/api/emailReadiness', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api/emailReadiness')>('../../../shared/api/emailReadiness');
  return { ...actual, getEmailReadiness: vi.fn(), getSignInDeliveryAdminReport: vi.fn(), sendDeliveryProbe: vi.fn() };
});

describe('EmailDelivery admin surface', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(emailAPI.getEmailReadiness).mockResolvedValue({
      Domain: 'example.test',
      Providers: [{ Provider: 'Mailgun', Status: 'fail', Detail: 'SPF mechanism is missing', Record: 'TXT example.test = v=spf1 include:mailgun.org' }],
      DMARC: { Provider: 'DMARC', Status: 'pass', Detail: 'p=reject' },
      CheckedAt: new Date().toISOString(),
    });
    vi.mocked(emailAPI.getSignInDeliveryAdminReport).mockResolvedValue({
      window_hours: 24,
      outbox: { pending: 2 },
      providers: [{ id: 'mailgun', transport: 'mailgun_smtp', enabled: true, cost_rank: 10, recommended: true, credential: { ok: false, detail: 'credential rejected' }, dns: { ok: false, detail: 'publish TXT record' }, state: 'needs_setup', remedy: 'Rotate the Mailgun credential.' }],
      routing: [{ provider_id: 'mailgun', cost_rank: 10, eligible: false, skip_reason: 'credential is not verified' }],
      chosen_provider: '',
      quotas: [{ provider_id: 'mailgun', window: 'daily', used: 8, ceiling: 10, started_at: new Date().toISOString(), ends_at: new Date().toISOString() }],
      queue_health: { by_priority: [{ priority: 80, depth: 2, oldest_age_seconds: 12 }], oldest_wait_seconds: 12, median_acceptance_seconds_24h: 4 },
      history: [{ id: 'message-1', purpose: 'signin', recipient: 'person@example.net', requested_at: new Date().toISOString(), provider_id: 'mailgun', status: 'failed', last_error: 'credential rejected', attempts: [{ attempt: 1, provider_id: 'mailgun', outcome: 'permanent', diagnostic: '535', started_at: new Date().toISOString() }] }],
    });
  });

  it('renders provider remedy, routing, quota, queue, and searchable history without credentials', async () => {
    render(<BrowserRouter><EmailDelivery /></BrowserRouter>);

    expect(await screen.findByText('Email delivery')).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText(/Rotate the Mailgun credential/)).toBeInTheDocument());
    expect(screen.getByText('mailgun · rank 10')).toBeInTheDocument();
    expect(screen.getByText('mailgun · daily: 8/10')).toBeInTheDocument();
    expect(screen.getByText(/Oldest waiting message: 12s/)).toBeInTheDocument();
    expect(screen.getByText(/person@example.net/)).toBeInTheDocument();
    expect(screen.getByLabelText('SMTP password')).toHaveValue('');
  });
});
