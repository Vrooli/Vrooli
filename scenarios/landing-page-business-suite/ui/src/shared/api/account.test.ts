import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { getSubscriptionInfo, getCreditInfo, getEntitlements } from './account';
import type { SubscriptionInfo, CreditInfo } from './types';
import { createFetchMock, mockResponses, installFetchMock } from '../test-utils/api-mocks';

const accountClient = vi.hoisted(() => ({ getMySubscription: vi.fn(), getMyCredits: vi.fn(), getEntitlements: vi.fn() }));
vi.mock('@connectrpc/connect', () => ({ createClient: vi.fn(() => accountClient) }));

describe('account API', () => {
  let fetchMock: ReturnType<typeof createFetchMock>;

  beforeEach(() => {
    vi.clearAllMocks();
    fetchMock = createFetchMock();
    installFetchMock(fetchMock);
    const response = async () => {
      const value = await fetchMock('/landing_page_business_suite.v1.AccountService');
      if (!value || typeof value.json !== 'function') throw new Error('expected Connect account response');
      return value.json();
    };
    accountClient.getMySubscription.mockImplementation(response);
    accountClient.getMyCredits.mockImplementation(response);
    accountClient.getEntitlements.mockImplementation(response);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('getSubscriptionInfo', () => {
    it.each([
      [1, 'active'], [2, 'trialing'], [3, 'past_due'], [4, 'canceled'], [0, 'inactive'], [99, 'inactive'],
    ])('maps generated subscription state %s to the public %s status', async (state, expectedStatus) => {
      fetchMock.mockResolvedValue(mockResponses.success({ status: { state } }));
      await expect(getSubscriptionInfo()).resolves.toMatchObject({ status: expectedStatus });
    });

    it('returns subscription info fields', async () => {
      const protoResponse = {
        status: {
          subscriptionId: 'sub_123',
          userIdentity: 'user@example.com',
          planTier: 'pro',
          stripePriceId: 'price_pro_monthly',
          bundleKey: 'main',
          cachedAt: {
            toJsonString: () => '2024-01-01T00:00:00Z',
          },
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getSubscriptionInfo();

      expect(result.subscription_id).toBe('sub_123');
      expect(result.customer_email).toBe('user@example.com');
      expect(result.plan_tier).toBe('pro');
      expect(result.price_id).toBe('price_pro_monthly');
    });

    it('returns a valid status string', async () => {
      fetchMock.mockResolvedValue(
        mockResponses.success({
          status: {},
        })
      );

      const result = await getSubscriptionInfo();

      expect(typeof result.status).toBe('string');
      expect(['active', 'inactive', 'trialing', 'past_due', 'canceled']).toContain(result.status);
    });

    it('handles missing cachedAt', async () => {
      const protoResponse = {
        status: {
          subscriptionId: 'sub_123',
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getSubscriptionInfo();

      expect(result.updated_at).toBeUndefined();
    });

    it('handles missing status object', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({}));

      const result = await getSubscriptionInfo();

      expect(typeof result.status).toBe('string');
    });

  });

  describe('getCreditInfo', () => {
    it('returns credit information', async () => {
      const response = {
        balance: {
          customerEmail: 'user@example.com',
          balanceCredits: 5000000,
          bundleKey: 'main',
        },
        displayCreditsLabel: 'credits',
        displayCreditsMultiplier: 0.001,
      };
      fetchMock.mockResolvedValue(mockResponses.success(response));

      const result = await getCreditInfo();

      expect(result.customer_email).toBe('user@example.com');
      expect(result.balance_credits).toBe(5000000);
      expect(result.display_credits_label).toBe('credits');
      expect(result.display_credits_multiplier).toBe(0.001);
    });

    it('handles missing balance', async () => {
      const response = {
        displayCreditsLabel: 'tokens',
        displayCreditsMultiplier: 1,
      };
      fetchMock.mockResolvedValue(mockResponses.success(response));

      const result = await getCreditInfo();

      expect(result.customer_email).toBe('');
      expect(result.balance_credits).toBe(0);
    });

    it('handles empty response', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({}));

      const result = await getCreditInfo();

      expect(result.customer_email).toBe('');
      expect(result.balance_credits).toBe(0);
      expect(result.bonus_credits).toBe(0);
      expect(result.display_credits_label).toBe('credits');
      expect(result.display_credits_multiplier).toBe(1);
    });

    it('defaults bonus_credits to 0', async () => {
      const response = {
        balance: {
          balanceCredits: 1000,
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(response));

      const result = await getCreditInfo();

      expect(result.bonus_credits).toBe(0);
    });
  });

  describe('getEntitlements', () => {
    it('uses the generated Connect account procedure', async () => {
      const response = {
        status: 'active',
        planTier: 'pro',
        features: ['feature_a', 'feature_b'],
      };
      fetchMock.mockResolvedValue(mockResponses.success(response));

      const result = await getEntitlements();

      expect(accountClient.getEntitlements).toHaveBeenCalledWith({});
      expect(result.plan_tier).toBe('pro');
    });

  });
});
