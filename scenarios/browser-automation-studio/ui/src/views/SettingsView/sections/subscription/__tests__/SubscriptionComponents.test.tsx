/**
 * Subscription Components Test Suite
 *
 * Tests subscription UI components including:
 * - UpgradePromptSection URL generation
 * - SubscriptionStatusCard "Get Subscription" button
 * - EmailInputSection "Get Subscription" link
 *
 * Requirements validated:
 * - Correct vrooli.com checkout URLs
 * - URL includes plan and email parameters
 * - Get Subscription buttons appear for inactive status
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';

// Mock the entitlement store
const mockEntitlementStore = {
  status: null as null | {
    status: string;
    tier: string;
    is_active: boolean;
  },
  userEmail: '',
  isLoading: false,
  isOffline: false,
  lastFetched: null,
  refreshEntitlement: vi.fn(),
};

vi.mock('@stores/entitlementStore', () => ({
  useEntitlementStore: vi.fn((selector) => {
    if (typeof selector === 'function') {
      return selector(mockEntitlementStore);
    }
    return mockEntitlementStore;
  }),
  TIER_CONFIG: {
    free: { label: 'Free', color: 'text-gray-400', bgColor: 'bg-gray-700/50', borderColor: 'border-gray-600' },
    solo: { label: 'Solo', color: 'text-blue-400', bgColor: 'bg-blue-900/30', borderColor: 'border-blue-600' },
    pro: { label: 'Pro', color: 'text-purple-400', bgColor: 'bg-purple-900/30', borderColor: 'border-purple-600' },
    studio: { label: 'Studio', color: 'text-amber-400', bgColor: 'bg-amber-900/30', borderColor: 'border-amber-600' },
    business: { label: 'Business', color: 'text-emerald-400', bgColor: 'bg-gradient-to-r from-emerald-900/30 to-teal-900/30', borderColor: 'border-emerald-600' },
  },
  STATUS_CONFIG: {
    active: { label: 'Active', color: 'text-green-400', icon: 'check' },
    trialing: { label: 'Trial', color: 'text-blue-400', icon: 'clock' },
    past_due: { label: 'Past Due', color: 'text-amber-400', icon: 'alert' },
    canceled: { label: 'Canceled', color: 'text-red-400', icon: 'x' },
    inactive: { label: 'Inactive', color: 'text-gray-400', icon: 'x' },
  },
}));

vi.mock('@shared/ui', () => ({
  TierBadge: ({ tier, size }: { tier: string; size?: string }) => (
    <span data-testid="tier-badge" data-tier={tier} data-size={size}>
      {tier}
    </span>
  ),
  LoadingSpinner: ({ size }: { size?: number }) => (
    <div data-testid="loading-spinner" data-size={size} />
  ),
}));

vi.mock('date-fns', () => ({
  formatDistanceToNow: () => '5 minutes ago',
}));

// Import components after mocks
import { UpgradePromptSection } from '../UpgradePromptSection';
import { SubscriptionStatusCard } from '../SubscriptionStatusCard';
import { EmailInputSection } from '../EmailInputSection';

describe('UpgradePromptSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockEntitlementStore.status = null;
    mockEntitlementStore.userEmail = '';
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders for free tier users', () => {
    mockEntitlementStore.status = {
      status: 'inactive',
      tier: 'free',
      is_active: false,
    };

    render(<UpgradePromptSection />);

    expect(screen.getByText('Upgrade Your Plan')).toBeInTheDocument();
    expect(screen.getByText('Upgrade Now')).toBeInTheDocument();
  });

  it('does not render for studio tier users', () => {
    mockEntitlementStore.status = {
      status: 'active',
      tier: 'studio',
      is_active: true,
    };

    const { container } = render(<UpgradePromptSection />);

    expect(container.firstChild).toBeNull();
  });

  it('does not render for business tier users', () => {
    mockEntitlementStore.status = {
      status: 'active',
      tier: 'business',
      is_active: true,
    };

    const { container } = render(<UpgradePromptSection />);

    expect(container.firstChild).toBeNull();
  });

  it('generates correct checkout URL with plan parameter', () => {
    mockEntitlementStore.status = {
      status: 'inactive',
      tier: 'free',
      is_active: false,
    };

    render(<UpgradePromptSection />);

    const upgradeLink = screen.getByText('Upgrade Now').closest('a');
    expect(upgradeLink).toHaveAttribute('href', expect.stringContaining('https://vrooli.com/checkout'));
    expect(upgradeLink).toHaveAttribute('href', expect.stringContaining('plan=pro'));
  });

  it('includes email in checkout URL when user email is set', () => {
    mockEntitlementStore.status = {
      status: 'inactive',
      tier: 'free',
      is_active: false,
    };
    mockEntitlementStore.userEmail = 'test@example.com';

    render(<UpgradePromptSection />);

    const upgradeLink = screen.getByText('Upgrade Now').closest('a');
    expect(upgradeLink).toHaveAttribute('href', expect.stringContaining('email=test%40example.com'));
  });

  it('recommends pro tier for free users', () => {
    mockEntitlementStore.status = {
      status: 'inactive',
      tier: 'free',
      is_active: false,
    };

    render(<UpgradePromptSection />);

    const upgradeLink = screen.getByText('Upgrade Now').closest('a');
    expect(upgradeLink).toHaveAttribute('href', expect.stringContaining('plan=pro'));
  });

  it('recommends pro tier for solo users', () => {
    mockEntitlementStore.status = {
      status: 'active',
      tier: 'solo',
      is_active: true,
    };

    render(<UpgradePromptSection />);

    const upgradeLink = screen.getByText('Upgrade Now').closest('a');
    expect(upgradeLink).toHaveAttribute('href', expect.stringContaining('plan=pro'));
  });

  it('recommends studio tier for pro users', () => {
    mockEntitlementStore.status = {
      status: 'active',
      tier: 'pro',
      is_active: true,
    };

    render(<UpgradePromptSection />);

    const upgradeLink = screen.getByText('Upgrade Now').closest('a');
    expect(upgradeLink).toHaveAttribute('href', expect.stringContaining('plan=studio'));
  });

  it('opens upgrade link in new tab', () => {
    mockEntitlementStore.status = {
      status: 'inactive',
      tier: 'free',
      is_active: false,
    };

    render(<UpgradePromptSection />);

    const upgradeLink = screen.getByText('Upgrade Now').closest('a');
    expect(upgradeLink).toHaveAttribute('target', '_blank');
    expect(upgradeLink).toHaveAttribute('rel', 'noopener noreferrer');
  });

  it('shows tier comparison cards', () => {
    mockEntitlementStore.status = {
      status: 'inactive',
      tier: 'free',
      is_active: false,
    };

    render(<UpgradePromptSection />);

    expect(screen.getByText('Solo')).toBeInTheDocument();
    expect(screen.getByText('Pro')).toBeInTheDocument();
    expect(screen.getByText('Studio')).toBeInTheDocument();
    expect(screen.getByText('Popular')).toBeInTheDocument();
  });
});

describe('SubscriptionStatusCard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockEntitlementStore.status = null;
    mockEntitlementStore.userEmail = '';
    mockEntitlementStore.isLoading = false;
    mockEntitlementStore.isOffline = false;
    mockEntitlementStore.lastFetched = new Date();
  });

  it('shows "Get Subscription" button for inactive status', () => {
    mockEntitlementStore.status = {
      status: 'inactive',
      tier: 'free',
      is_active: false,
    };

    render(<SubscriptionStatusCard />);

    expect(screen.getByText('Get Subscription')).toBeInTheDocument();
  });

  it('shows "Get Subscription" button for canceled status', () => {
    mockEntitlementStore.status = {
      status: 'canceled',
      tier: 'free',
      is_active: false,
    };

    render(<SubscriptionStatusCard />);

    expect(screen.getByText('Get Subscription')).toBeInTheDocument();
  });

  it('does not show "Get Subscription" for active status', () => {
    mockEntitlementStore.status = {
      status: 'active',
      tier: 'pro',
      is_active: true,
    };

    render(<SubscriptionStatusCard />);

    expect(screen.queryByText('Get Subscription')).not.toBeInTheDocument();
  });

  it('generates correct checkout URL for Get Subscription button', () => {
    mockEntitlementStore.status = {
      status: 'inactive',
      tier: 'free',
      is_active: false,
    };
    mockEntitlementStore.userEmail = 'user@example.com';

    render(<SubscriptionStatusCard />);

    const getSubLink = screen.getByText('Get Subscription').closest('a');
    expect(getSubLink).toHaveAttribute('href', expect.stringContaining('https://vrooli.com/checkout'));
    expect(getSubLink).toHaveAttribute('href', expect.stringContaining('plan=pro'));
    expect(getSubLink).toHaveAttribute('href', expect.stringContaining('email=user%40example.com'));
  });

  it('shows placeholder when no status', () => {
    mockEntitlementStore.status = null;

    render(<SubscriptionStatusCard />);

    expect(screen.getByText('No subscription status available.')).toBeInTheDocument();
  });

  it('shows tier badge when status exists', () => {
    mockEntitlementStore.status = {
      status: 'active',
      tier: 'pro',
      is_active: true,
    };

    render(<SubscriptionStatusCard />);

    expect(screen.getByTestId('tier-badge')).toBeInTheDocument();
  });

  it('shows last fetched time', () => {
    mockEntitlementStore.status = {
      status: 'active',
      tier: 'pro',
      is_active: true,
    };
    mockEntitlementStore.lastFetched = new Date();

    render(<SubscriptionStatusCard />);

    expect(screen.getByText(/Last updated/)).toBeInTheDocument();
  });
});

describe('EmailInputSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockEntitlementStore.status = null;
    mockEntitlementStore.userEmail = '';
    mockEntitlementStore.isLoading = false;
  });

  it('renders email input', () => {
    render(<EmailInputSection />);

    expect(screen.getByPlaceholderText('you@example.com')).toBeInTheDocument();
  });

  it('shows "Get Subscription" link for inactive status when email is entered', () => {
    mockEntitlementStore.status = {
      status: 'inactive',
      tier: 'free',
      is_active: false,
    };
    mockEntitlementStore.userEmail = 'user@example.com';

    render(<EmailInputSection />);

    expect(screen.getByText('Get Subscription')).toBeInTheDocument();
    expect(screen.getByText('No active subscription found for this email.')).toBeInTheDocument();
  });

  it('shows "Get Subscription" link for canceled status', () => {
    mockEntitlementStore.status = {
      status: 'canceled',
      tier: 'free',
      is_active: false,
    };
    mockEntitlementStore.userEmail = 'user@example.com';

    render(<EmailInputSection />);

    expect(screen.getByText('Get Subscription')).toBeInTheDocument();
  });

  it('does not show "Get Subscription" for active status', () => {
    mockEntitlementStore.status = {
      status: 'active',
      tier: 'pro',
      is_active: true,
    };
    mockEntitlementStore.userEmail = 'user@example.com';

    render(<EmailInputSection />);

    expect(screen.queryByText('Get Subscription')).not.toBeInTheDocument();
    expect(screen.queryByText('No active subscription found')).not.toBeInTheDocument();
  });

  it('generates correct checkout URL in Get Subscription link', () => {
    mockEntitlementStore.status = {
      status: 'inactive',
      tier: 'free',
      is_active: false,
    };
    mockEntitlementStore.userEmail = 'test@example.com';

    render(<EmailInputSection />);

    const getSubLink = screen.getByText('Get Subscription').closest('a');
    expect(getSubLink).toHaveAttribute('href', expect.stringContaining('https://vrooli.com/checkout'));
    expect(getSubLink).toHaveAttribute('href', expect.stringContaining('plan=pro'));
    expect(getSubLink).toHaveAttribute('href', expect.stringContaining('email=test%40example.com'));
  });

  it('does not show inactive message without email stored', () => {
    mockEntitlementStore.status = {
      status: 'inactive',
      tier: 'free',
      is_active: false,
    };
    mockEntitlementStore.userEmail = ''; // No email stored

    render(<EmailInputSection />);

    // The inactive message should only show when there's a stored email
    expect(screen.queryByText('No active subscription found for this email.')).not.toBeInTheDocument();
  });

  it('shows verify subscription button', () => {
    render(<EmailInputSection />);

    expect(screen.getByText('Verify Subscription')).toBeInTheDocument();
  });

  it('shows update email button when email is already set', () => {
    mockEntitlementStore.userEmail = 'existing@example.com';
    mockEntitlementStore.status = {
      status: 'active',
      tier: 'pro',
      is_active: true,
    };

    render(<EmailInputSection />);

    expect(screen.getByText('Update Email')).toBeInTheDocument();
  });

  it('shows clear email button when email is set', () => {
    mockEntitlementStore.userEmail = 'existing@example.com';
    mockEntitlementStore.status = {
      status: 'active',
      tier: 'pro',
      is_active: true,
    };

    render(<EmailInputSection />);

    expect(screen.getByText('Clear Email')).toBeInTheDocument();
  });
});

describe('URL Generation', () => {
  beforeEach(() => {
    mockEntitlementStore.status = {
      status: 'inactive',
      tier: 'free',
      is_active: false,
    };
  });

  it('uses vrooli.com as default landing page URL', () => {
    render(<UpgradePromptSection />);

    const upgradeLink = screen.getByText('Upgrade Now').closest('a');
    expect(upgradeLink?.getAttribute('href')).toMatch(/^https:\/\/vrooli\.com/);
  });

  it('properly encodes special characters in email', () => {
    mockEntitlementStore.userEmail = 'test+tag@example.com';

    render(<UpgradePromptSection />);

    const upgradeLink = screen.getByText('Upgrade Now').closest('a');
    expect(upgradeLink?.getAttribute('href')).toContain('email=test%2Btag%40example.com');
  });
});
