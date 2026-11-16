import { describe, expect, it, vi, beforeEach } from 'vitest';
import { fireEvent, render, screen } from '@/test-utils';

const state = {
  status: null as { tier: string; status: 'active' | 'inactive'; monthly_remaining: number; ai_credits_limit: number; ai_credits_remaining: number } | null,
  userEmail: '',
  isLoading: false,
  error: null as string | null,
  getUserEmail: vi.fn(async () => ''),
  fetchStatus: vi.fn(async () => undefined),
  refreshEntitlement: vi.fn(),
  setUserEmail: vi.fn(),
};
const auth = { isAuthenticated: false, isLoading: false, user: null as { email: string } | null, error: null as string | null, signIn: vi.fn(), signOut: vi.fn() };

vi.mock('@stores/entitlementStore', () => ({
  useEntitlementStore: (selector?: (value: typeof state) => unknown) => selector ? selector(state) : state,
}));
vi.mock('@stores/authStore', () => ({
  useAuthStore: () => auth,
}));
vi.mock('@vrooli/react-component-library/MonetizationAccount/2.0.2', () => ({
  MonetizationAccountPanel: ({ configured, plan, statusLabel, creditsLabel, onSignIn, onSignOut, onRefresh, onManageAccount, onComparePlans }: {
    configured: boolean; plan: string; statusLabel: string; creditsLabel: string;
    onSignIn: () => void; onSignOut: () => void; onRefresh: () => void;
    onManageAccount: () => void; onComparePlans: () => void;
  }) => (
    <section data-testid="account-panel" data-configured={configured}>
      <span>{plan}:{statusLabel}:{creditsLabel}</span>
      <button type="button" onClick={onSignIn}>Sign in</button>
      <button type="button" onClick={onSignOut}>Sign out</button>
      <button type="button" onClick={onRefresh}>Refresh</button>
      <button type="button" onClick={onManageAccount}>Manage billing</button>
      <button type="button" onClick={onComparePlans}>Compare plans</button>
    </section>
  ),
}));
vi.mock('../UnifiedUsageSection', () => ({ UnifiedUsageSection: () => null }));
vi.mock('../UsageHistorySection', () => ({ UsageHistorySection: () => null }));
vi.mock('../OperationLogModal', () => ({ OperationLogModal: () => null }));
vi.mock('../FeatureAccessList', () => ({ FeatureAccessList: () => null }));
vi.mock('@shared/ui', () => ({ LoadingSpinner: () => null }));
vi.mock('@shared/upgradeDestination', () => ({ LANDING_PAGE_URL: 'https://vrooli.com', resolveUpgradeDestination: (path: string) => new URL(path, 'https://vrooli.com').toString() }));
vi.mock('@components/MonetizationAccount', () => ({
  UpgradePrompt: ({ feature, requiredPlan, href }: { feature: string; requiredPlan: string; href: string }) => (
    <a href={href}>{feature} requires {requiredPlan}</a>
  ),
}));

import { SubscriptionTab } from '../SubscriptionTab';
import { UpgradePromptSection } from '../UpgradePromptSection';

describe('shared monetization account adapters', () => {
  beforeEach(() => {
    state.status = null;
    state.userEmail = '';
    auth.isAuthenticated = false;
    auth.isLoading = false;
    auth.user = null;
    auth.error = null;
    vi.clearAllMocks();
  });

  it('uses the shared account panel while preserving BAS auth and hosted billing actions', () => {
    state.status = { tier: 'pro', status: 'active', monthly_remaining: 95, ai_credits_limit: 100, ai_credits_remaining: 75 };
    auth.isAuthenticated = true;
    const open = vi.spyOn(window, 'open').mockImplementation(() => null);
    render(<SubscriptionTab />);
    expect(screen.getByTestId('account-panel')).toHaveAttribute('data-configured', 'true');
    expect(screen.getByText('pro:Active:75 remaining')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));
    fireEvent.click(screen.getByRole('button', { name: 'Sign out' }));
    fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));
    expect(auth.signIn).toHaveBeenCalledOnce();
    expect(auth.signOut).toHaveBeenCalledOnce();
    expect(state.refreshEntitlement).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole('button', { name: 'Manage billing' }));
    fireEvent.click(screen.getByRole('button', { name: 'Compare plans' }));
    expect(open).toHaveBeenNthCalledWith(1, 'https://vrooli.com/account', '_blank', 'noopener,noreferrer');
    expect(open).toHaveBeenNthCalledWith(2, 'https://vrooli.com/pricing', '_blank', 'noopener,noreferrer');
    open.mockRestore();
  });

  it('uses the shared upgrade prompt and hides it for the highest plans', () => {
    state.status = { tier: 'free', status: 'inactive', monthly_remaining: 0, ai_credits_limit: 100, ai_credits_remaining: 0 };
    state.userEmail = 'buyer@example.com';
    render(<UpgradePromptSection />);
    expect(screen.getByRole('link')).toHaveAttribute('href', expect.stringContaining('email=buyer%40example.com'));

    state.status = { tier: 'business', status: 'active', monthly_remaining: 100, ai_credits_limit: 100, ai_credits_remaining: 100 };
    const { container } = render(<UpgradePromptSection />);
    expect(container.firstChild).toBeNull();
  });
});
