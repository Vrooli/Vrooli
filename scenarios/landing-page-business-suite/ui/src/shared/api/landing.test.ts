import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { getLandingConfig, getPlans } from './landing';
import { ApiError } from './common';
import { createFetchMock, mockResponses, installFetchMock } from '../test-utils/api-mocks';

vi.mock('@bufbuild/protobuf', () => ({
  fromJson: vi.fn((schema, data) => data),
}));

describe('landing API', () => {
  let fetchMock: ReturnType<typeof createFetchMock>;

  beforeEach(() => {
    vi.clearAllMocks();
    fetchMock = createFetchMock();
    installFetchMock(fetchMock);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('getLandingConfig', () => {
    it('returns landing configuration', async () => {
      const config = {
        title: 'Welcome',
        subtitle: 'Best landing page',
        sections: [{ type: 'hero', enabled: true }],
      };
      fetchMock.mockResolvedValue(mockResponses.success(config));

      const result = await getLandingConfig();

      expect(result).toEqual(config);
    });

    it('calls endpoint without variant param when not provided', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({}));

      await getLandingConfig();

      const callArgs = fetchMock.mock.calls[0];
      expect(callArgs[0]).toContain('/landing-config');
      expect(callArgs[0]).not.toContain('variant=');
    });

    it('includes variant param when provided', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({}));

      await getLandingConfig('dark-theme');

      const callArgs = fetchMock.mock.calls[0];
      expect(callArgs[0]).toContain('variant=dark-theme');
    });

    it('URL encodes variant slug', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({}));

      await getLandingConfig('special/variant');

      const callArgs = fetchMock.mock.calls[0];
      expect(callArgs[0]).toContain(encodeURIComponent('special/variant'));
    });

    it('throws on server error', async () => {
      fetchMock.mockResolvedValue(mockResponses.serverError());

      await expect(getLandingConfig()).rejects.toBeInstanceOf(ApiError);
    });
  });

  describe('getPlans', () => {
    it('returns pricing overview with normalized plans', async () => {
      const protoResponse = {
        pricing: {
          bundle: {
            bundleKey: 'main',
            name: 'Main Bundle',
            stripeProductId: 'prod_123',
            creditsPerUsd: 1000000,
            displayCreditsMultiplier: 0.001,
            displayCreditsLabel: 'credits',
            environment: 'production',
          },
          monthly: [
            {
              planName: 'Pro',
              planTier: 'pro',
              amountCents: 9900,
              currency: 'usd',
              introEnabled: false,
              stripePriceId: 'price_pro_monthly',
              monthlyIncludedCredits: 10000000,
              oneTimeBonusCredits: 0,
              displayEnabled: true,
              displayWeight: 50,
            },
          ],
          yearly: [
            {
              planName: 'Pro Annual',
              planTier: 'pro',
              amountCents: 99000,
              currency: 'usd',
              introEnabled: false,
              stripePriceId: 'price_pro_yearly',
              monthlyIncludedCredits: 10000000,
              oneTimeBonusCredits: 5000000,
              displayEnabled: true,
              displayWeight: 50,
            },
          ],
          updatedAt: '2024-01-01T00:00:00Z',
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getPlans();

      expect(result.bundle.bundle_key).toBe('main');
      expect(result.bundle.name).toBe('Main Bundle');
      expect(result.monthly).toHaveLength(1);
      expect(result.monthly[0].plan_name).toBe('Pro');
      expect(result.yearly).toHaveLength(1);
      expect(result.yearly[0].plan_name).toBe('Pro Annual');
    });

    it('returns plans in monthly and yearly arrays', async () => {
      const protoResponse = {
        pricing: {
          bundle: {
            bundleKey: 'main',
            name: 'Main',
            stripeProductId: 'prod_123',
            creditsPerUsd: 1000000,
            displayCreditsMultiplier: 0.001,
            displayCreditsLabel: 'credits',
          },
          monthly: [{ planName: 'Monthly' }],
          yearly: [{ planName: 'Yearly' }],
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getPlans();

      expect(result.monthly).toHaveLength(1);
      expect(result.yearly).toHaveLength(1);
      expect(result.monthly[0].plan_name).toBe('Monthly');
      expect(result.yearly[0].plan_name).toBe('Yearly');
    });

    it('handles timestamp with toJsonString method', async () => {
      const protoResponse = {
        pricing: {
          bundle: { bundleKey: 'main', name: 'Main', stripeProductId: 'prod_123' },
          monthly: [],
          yearly: [],
          updatedAt: {
            toJsonString: () => '2024-06-15T10:30:00Z',
          },
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getPlans();

      expect(result.updated_at).toBe('2024-06-15T10:30:00Z');
    });

    it('handles timestamp with seconds/nanos', async () => {
      const protoResponse = {
        pricing: {
          bundle: { bundleKey: 'main', name: 'Main', stripeProductId: 'prod_123' },
          monthly: [],
          yearly: [],
          updatedAt: {
            seconds: 1718444100,
            nanos: 500000000,
          },
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getPlans();

      expect(result.updated_at).toBeDefined();
      expect(new Date(result.updated_at).getTime()).toBeGreaterThan(0);
    });

    it('handles string timestamp', async () => {
      const protoResponse = {
        pricing: {
          bundle: { bundleKey: 'main', name: 'Main', stripeProductId: 'prod_123' },
          monthly: [],
          yearly: [],
          updatedAt: '2024-01-01T00:00:00Z',
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getPlans();

      expect(result.updated_at).toBe('2024-01-01T00:00:00Z');
    });

    it('uses current time when updatedAt is missing', async () => {
      const beforeCall = new Date().toISOString();
      const protoResponse = {
        pricing: {
          bundle: { bundleKey: 'main', name: 'Main', stripeProductId: 'prod_123' },
          monthly: [],
          yearly: [],
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getPlans();

      const afterCall = new Date().toISOString();
      expect(result.updated_at >= beforeCall).toBe(true);
      expect(result.updated_at <= afterCall).toBe(true);
    });

    it('returns plan_name from normalized plan', async () => {
      const protoResponse = {
        pricing: {
          bundle: { bundleKey: 'main', name: 'Main', stripeProductId: 'prod_123' },
          monthly: [
            { planName: 'Plan1' },
            { planName: 'Plan2' },
            { planName: 'Plan3' },
          ],
          yearly: [],
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getPlans();

      expect(result.monthly).toHaveLength(3);
      expect(result.monthly[0].plan_name).toBe('Plan1');
      expect(result.monthly[1].plan_name).toBe('Plan2');
      expect(result.monthly[2].plan_name).toBe('Plan3');
    });

    it('handles intro pricing data', async () => {
      const protoResponse = {
        pricing: {
          bundle: { bundleKey: 'main', name: 'Main', stripeProductId: 'prod_123' },
          monthly: [
            {
              planName: 'Pro',
              introEnabled: true,
              introAmountCents: 4900,
              introPeriods: 3,
              introPriceLookupKey: 'pro_intro',
            },
          ],
          yearly: [],
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getPlans();

      expect(result.monthly[0].intro_enabled).toBe(true);
      expect(result.monthly[0].intro_amount_cents).toBe(4900);
      expect(result.monthly[0].intro_periods).toBe(3);
      expect(result.monthly[0].intro_price_lookup_key).toBe('pro_intro');
    });

    it('defaults numeric fields to 0 when missing', async () => {
      const protoResponse = {
        pricing: {
          bundle: { bundleKey: 'main', name: 'Main', stripeProductId: 'prod_123' },
          monthly: [{ planName: 'Basic' }],
          yearly: [],
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getPlans();

      expect(result.monthly[0].amount_cents).toBe(0);
      expect(result.monthly[0].monthly_included_credits).toBe(0);
      expect(result.monthly[0].one_time_bonus_credits).toBe(0);
      expect(result.monthly[0].display_weight).toBe(0);
    });
  });
});
