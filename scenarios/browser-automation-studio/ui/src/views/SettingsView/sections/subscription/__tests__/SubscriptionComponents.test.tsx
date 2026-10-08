import { describe, expect, it, vi, beforeEach } from 'vitest';
import { fireEvent, render, screen } from '@/test-utils';

const state = {
  status: null as { tier: string; status: 'active' | 'inactive'; monthly_remaining: number } | null,
  userEmail: '',
  refreshEntitlement: vi.fn(),
};
const auth = { signedIn: false, error: null as string | null, signIn: vi.fn(), signOut: vi.fn() };

vi.mock('@stores/entitlementStore', () => ({
  useEntitlementStore: (selector?: (value: typeof state) => unknown) => selector ? selector(state) : state,
}));
vi.mock('@stores/authStore', () => ({
  useAuthStore: (selector?: (value: typeof auth) => unknown) => selector ? selector(auth) : auth,
  useIsAuthenticated: () => auth.signedIn,
}));
vi.mock('@components/MonetizationAccount', () => ({
  AuthSection: ({ signedIn, onSignIn, onSignOut }: { signedIn: boolean; onSignIn: () => void; onSignOut: () => void }) => (
    <button type="button" onClick={signedIn ? onSignOut : onSignIn}>{signedIn ? 'Sign out' : 'Sign in'}</button>
  ),
  SubscriptionStatusCard: ({ plan, status, credits }: { plan: string; status: string; credits: number }) => (
    <div data-testid="shared-status">{plan}:{status}:{credits}</div>
  ),
  UpgradePrompt: ({ feature, requiredPlan, href }: { feature: string; requiredPlan: string; href: string }) => (
    <a href={href}>{feature} requires {requiredPlan}</a>
  ),
}));

import { AuthSection } from '../AuthSection';
import { SubscriptionStatusCard } from '../SubscriptionStatusCard';
import { UpgradePromptSection } from '../UpgradePromptSection';

describe('shared monetization account adapters', () => {
  beforeEach(() => {
    state.status = null;
    state.userEmail = '';
    auth.signedIn = false;
    auth.error = null;
    vi.clearAllMocks();
  });

  it('delegates sign-in to the shared RCL account primitive', () => {
    render(<AuthSection />);
    fireEvent.click(screen.getByRole('button', { name: 'Sign in' }));
    expect(auth.signIn).toHaveBeenCalledOnce();
  });

  it('delegates lease status and refresh to shared RCL primitives', () => {
    state.status = { tier: 'pro', status: 'active', monthly_remaining: 95 };
    render(<SubscriptionStatusCard />);
    expect(screen.getByTestId('shared-status')).toHaveTextContent('pro:active:95');
    fireEvent.click(screen.getByRole('button', { name: 'Refresh subscription status' }));
    expect(state.refreshEntitlement).toHaveBeenCalledOnce();
  });

  it('uses the shared upgrade prompt and hides it for the highest plans', () => {
    state.status = { tier: 'free', status: 'inactive', monthly_remaining: 0 };
    state.userEmail = 'buyer@example.com';
    render(<UpgradePromptSection />);
    expect(screen.getByRole('link')).toHaveAttribute('href', expect.stringContaining('email=buyer%40example.com'));

    state.status = { tier: 'business', status: 'active', monthly_remaining: 100 };
    const { container } = render(<UpgradePromptSection />);
    expect(container.firstChild).toBeNull();
  });
});
