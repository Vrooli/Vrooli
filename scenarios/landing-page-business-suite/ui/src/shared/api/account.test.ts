import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { getSubscriptionInfo, getCreditInfo, getEntitlements } from './account';
import type { SubscriptionInfo, CreditInfo } from './types';
import { ApiError } from './common';
import { createFetchMock, mockResponses, installFetchMock } from '../test-utils/api-mocks';

vi.mock('@bufbuild/protobuf', () => ({
  fromJson: vi.fn((schema, data) => data),
}));

describe('account API', () => {
  let fetchMock: ReturnType<typeof createFetchMock>;

  beforeEach(() => {
    vi.clearAllMocks();
    fetchMock = createFetchMock();
    installFetchMock(fetchMock);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('getSubscriptionInfo', () => {
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

    it('throws on unauthorized', async () => {
      fetchMock.mockResolvedValue(mockResponses.unauthorized());

      await expect(getSubscriptionInfo()).rejects.toBeInstanceOf(ApiError);
    });
  });

  describe('getCreditInfo', () => {
    it('returns credit information', async () => {
      const response = {
        balance: {
          customer_email: 'user@example.com',
          balance_credits: 5000000,
          bundle_key: 'main',
          updated_at: '2024-01-01T00:00:00Z',
        },
        display_credits_label: 'credits',
        display_credits_multiplier: 0.001,
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
        display_credits_label: 'tokens',
        display_credits_multiplier: 1,
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
          balance_credits: 1000,
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(response));

      const result = await getCreditInfo();

      expect(result.bonus_credits).toBe(0);
    });
  });

  describe('getEntitlements', () => {
    it('returns entitlements without user param', async () => {
      const response = {
        status: 'active',
        plan_tier: 'pro',
        features: ['feature_a', 'feature_b'],
      };
      fetchMock.mockResolvedValue(mockResponses.success(response));

      const result = await getEntitlements();

      const callArgs = fetchMock.mock.calls[0];
      expect(callArgs[0]).toContain('/entitlements');
      expect(callArgs[0]).not.toContain('user=');
      expect(result.plan_tier).toBe('pro');
    });

    it('includes user param when provided', async () => {
      fetchMock.mockResolvedValue(
        mockResponses.success({
          status: 'inactive',
          plan_tier: 'free',
          features: [],
        })
      );

      await getEntitlements('user@example.com');

      const callArgs = fetchMock.mock.calls[0];
      expect(callArgs[0]).toContain('user=user%40example.com');
    });

    it('trims whitespace from email', async () => {
      fetchMock.mockResolvedValue(
        mockResponses.success({
          status: 'inactive',
          plan_tier: 'free',
          features: [],
        })
      );

      await getEntitlements('  user@example.com  ');

      const callArgs = fetchMock.mock.calls[0];
      expect(callArgs[0]).toContain('user=user%40example.com');
    });

    it('skips empty user param', async () => {
      fetchMock.mockResolvedValue(
        mockResponses.success({
          status: 'inactive',
          plan_tier: 'free',
          features: [],
        })
      );

      await getEntitlements('   ');

      const callArgs = fetchMock.mock.calls[0];
      expect(callArgs[0]).not.toContain('user=');
    });

    it('throws on server error', async () => {
      fetchMock.mockResolvedValue(mockResponses.serverError());

      await expect(getEntitlements()).rejects.toBeInstanceOf(ApiError);
    });
  });
});
