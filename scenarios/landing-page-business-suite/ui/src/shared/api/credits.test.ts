import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import {
  listAPIKeys,
  createAPIKey,
  deleteAPIKey,
  testAPIKey,
  toggleAPIKey,
  getAllTierLimits,
  getTierLimits,
  updateTierLimit,
  createTierLimit,
  deleteTierLimit,
  getAppLimits,
  getUsageSummary,
  getAdminUsageSummary,
  formatCredits,
  formatDollars,
  dollarsToInternalUnits,
  TIER_OPTIONS,
  PROVIDER_OPTIONS,
  type APIKey,
  type TierLimit,
  type UsageSummary,
} from './credits';
import { ApiError } from './common';
import { createFetchMock, mockResponses, installFetchMock, getFetchCall, parseJsonBody } from '../test-utils/api-mocks';

describe('credits API', () => {
  let fetchMock: ReturnType<typeof createFetchMock>;

  beforeEach(() => {
    vi.clearAllMocks();
    fetchMock = createFetchMock();
    installFetchMock(fetchMock);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('API Key Management', () => {
    describe('listAPIKeys', () => {
      it('returns list of API keys', async () => {
        const keys: APIKey[] = [
          {
            id: '1',
            provider: 'anthropic',
            key_hint: 'sk-ant-...abc',
            is_active: true,
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
          },
        ];
        fetchMock.mockResolvedValue(mockResponses.success({ keys }));

        const result = await listAPIKeys();

        expect(result.keys).toHaveLength(1);
        expect(result.keys[0]?.provider).toBe('anthropic');
      });

      it('returns empty array when no keys', async () => {
        fetchMock.mockResolvedValue(mockResponses.success({ keys: [] }));

        const result = await listAPIKeys();

        expect(result.keys).toHaveLength(0);
      });
    });

    describe('createAPIKey', () => {
      it('sends POST request with provider and key', async () => {
        const newKey: APIKey = {
          id: '2',
          provider: 'openai',
          key_hint: 'sk-...xyz',
          is_active: true,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        };
        fetchMock.mockResolvedValue(mockResponses.success(newKey));

        await createAPIKey({ provider: 'openai', key: 'sk-full-key-value' });

        const [, options] = getFetchCall(fetchMock);
        expect(options.method).toBe('POST');
        expect(parseJsonBody(options.body)).toEqual({
          provider: 'openai',
          key: 'sk-full-key-value',
        });
      });

      it('returns created key', async () => {
        const newKey: APIKey = {
          id: '2',
          provider: 'openai',
          key_hint: 'sk-...xyz',
          is_active: true,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        };
        fetchMock.mockResolvedValue(mockResponses.success(newKey));

        const result = await createAPIKey({ provider: 'openai', key: 'sk-full-key' });

        expect(result.id).toBe('2');
        expect(result.provider).toBe('openai');
      });
    });

    describe('deleteAPIKey', () => {
      it('sends DELETE request with provider in query string', async () => {
        fetchMock.mockResolvedValue(mockResponses.empty());

        await deleteAPIKey('anthropic');

        const [url, options] = getFetchCall(fetchMock);
        expect(options.method).toBe('DELETE');
        expect(url).toContain('provider=anthropic');
      });

      it('URL encodes provider', async () => {
        fetchMock.mockResolvedValue(mockResponses.empty());

        await deleteAPIKey('provider/with/special+chars');

        const [url] = getFetchCall(fetchMock);
        expect(url).toContain(encodeURIComponent('provider/with/special+chars'));
      });

      it('throws ApiError on failure', async () => {
        fetchMock.mockResolvedValue(mockResponses.notFound('API key not found'));

        await expect(deleteAPIKey('nonexistent')).rejects.toBeInstanceOf(ApiError);
      });
    });

    describe('testAPIKey', () => {
      it('sends POST request to test endpoint', async () => {
        const result = {
          success: true,
          message: 'API key is valid',
          provider: 'anthropic',
        };
        fetchMock.mockResolvedValue(mockResponses.success(result));

        await testAPIKey('anthropic');

        const [url, options] = getFetchCall(fetchMock);
        expect(options.method).toBe('POST');
        expect(url).toContain('/api/v1/admin/api-keys/test');
        expect(url).toContain('provider=anthropic');
      });

      it('returns test result', async () => {
        const testResult = {
          success: true,
          message: 'API key is valid',
          provider: 'anthropic',
        };
        fetchMock.mockResolvedValue(mockResponses.success(testResult));

        const result = await testAPIKey('anthropic');

        expect(result.success).toBe(true);
        expect(result.message).toBe('API key is valid');
      });

      it('returns failure result for invalid key', async () => {
        const testResult = {
          success: false,
          message: 'Invalid API key',
          provider: 'anthropic',
        };
        fetchMock.mockResolvedValue(mockResponses.success(testResult));

        const result = await testAPIKey('anthropic');

        expect(result.success).toBe(false);
      });
    });

    describe('toggleAPIKey', () => {
      it('sends POST request with provider and active state', async () => {
        fetchMock.mockResolvedValue(mockResponses.empty());

        await toggleAPIKey('anthropic', false);

        const [, options] = getFetchCall(fetchMock);
        expect(options.method).toBe('POST');
        expect(parseJsonBody(options.body)).toEqual({
          provider: 'anthropic',
          active: false,
        });
      });
    });
  });

  describe('Tier Limits Management', () => {
    describe('getAllTierLimits', () => {
      it('returns limits organized by tier', async () => {
        const limits: Record<string, TierLimit[]> = {
          free: [
            {
              id: '1',
              tier_id: 'free',
              limit_type: 'cost_based',
              limit_key: 'ai_credits',
              limit_value: 1000000,
              cost_multiplier: 1000000,
              reset_period: 'monthly',
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-01T00:00:00Z',
            },
          ],
          pro: [],
        };
        fetchMock.mockResolvedValue(mockResponses.success({ limits }));

        const result = await getAllTierLimits();

        expect(result.limits.free).toHaveLength(1);
        expect(result.limits.pro).toHaveLength(0);
      });
    });

    describe('getTierLimits', () => {
      it('returns limits for specific tier', async () => {
        const response = {
          tier_id: 'pro',
          limits: [
            {
              id: '2',
              tier_id: 'pro',
              limit_type: 'cost_based',
              limit_key: 'ai_credits',
              limit_value: 10000000,
              cost_multiplier: 1000000,
              reset_period: 'monthly',
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-01T00:00:00Z',
            },
          ],
        };
        fetchMock.mockResolvedValue(mockResponses.success(response));

        const result = await getTierLimits('pro');

        expect(result.tier_id).toBe('pro');
        expect(result.limits).toHaveLength(1);
      });

      it('URL encodes tier ID', async () => {
        fetchMock.mockResolvedValue(mockResponses.success({ tier_id: 'special', limits: [] }));

        await getTierLimits('tier/with/slashes');

        const [url] = getFetchCall(fetchMock);
        expect(url).toContain(encodeURIComponent('tier/with/slashes'));
      });
    });

    describe('updateTierLimit', () => {
      it('sends PUT request with update payload', async () => {
        const updatedLimit: TierLimit = {
          id: '1',
          tier_id: 'pro',
          limit_type: 'cost_based',
          limit_key: 'ai_credits',
          limit_value: 20000000,
          cost_multiplier: 1000000,
          reset_period: 'monthly',
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-02T00:00:00Z',
        };
        fetchMock.mockResolvedValue(mockResponses.success(updatedLimit));

        await updateTierLimit('pro', 'ai_credits', { limit_value: 20000000 });

        const [, options] = getFetchCall(fetchMock);
        expect(options.method).toBe('PUT');
        expect(parseJsonBody(options.body)).toEqual({
          limit_key: 'ai_credits',
          app_bundle_key: undefined,
          update: { limit_value: 20000000 },
        });
      });

      it('includes app_bundle_key when provided', async () => {
        const updatedLimit: TierLimit = {
          id: '1',
          tier_id: 'pro',
          limit_type: 'app_specific',
          limit_key: 'api_calls',
          limit_value: 1000,
          cost_multiplier: 1,
          app_bundle_key: 'my_app',
          reset_period: 'daily',
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-02T00:00:00Z',
        };
        fetchMock.mockResolvedValue(mockResponses.success(updatedLimit));

        await updateTierLimit('pro', 'api_calls', { limit_value: 2000 }, 'my_app');

        const [, options] = getFetchCall(fetchMock);
        expect(parseJsonBody(options.body).app_bundle_key).toBe('my_app');
      });
    });

    describe('createTierLimit', () => {
      it('sends POST request with limit data', async () => {
        const newLimit: TierLimit = {
          id: '3',
          tier_id: 'business',
          limit_type: 'cost_based',
          limit_key: 'ai_credits',
          limit_value: 50000000,
          cost_multiplier: 1000000,
          reset_period: 'monthly',
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        };
        fetchMock.mockResolvedValue(mockResponses.success(newLimit));

        await createTierLimit({
          tier_id: 'business',
          limit_type: 'cost_based',
          limit_key: 'ai_credits',
          limit_value: 50000000,
        });

        const [, options] = getFetchCall(fetchMock);
        expect(options.method).toBe('POST');
      });
    });

    describe('deleteTierLimit', () => {
      it('sends DELETE request with limit identifiers', async () => {
        fetchMock.mockResolvedValue(mockResponses.empty());

        await deleteTierLimit('pro', 'ai_credits');

        const [, options] = getFetchCall(fetchMock);
        expect(options.method).toBe('DELETE');
        expect(parseJsonBody(options.body)).toEqual({
          tier_id: 'pro',
          limit_key: 'ai_credits',
          app_bundle_key: undefined,
        });
      });

      it('includes app_bundle_key when provided', async () => {
        fetchMock.mockResolvedValue(mockResponses.empty());

        await deleteTierLimit('pro', 'api_calls', 'my_app');

        const [, options] = getFetchCall(fetchMock);
        expect(parseJsonBody(options.body).app_bundle_key).toBe('my_app');
      });
    });
  });

  describe('App Limits', () => {
    describe('getAppLimits', () => {
      it('returns limits for specific app', async () => {
        const response = {
          app_bundle_key: 'my_app',
          limits: {
            free: [
              {
                id: '1',
                tier_id: 'free',
                limit_type: 'app_specific',
                limit_key: 'api_calls',
                limit_value: 100,
                cost_multiplier: 1,
                app_bundle_key: 'my_app',
                reset_period: 'daily',
                created_at: '2024-01-01T00:00:00Z',
                updated_at: '2024-01-01T00:00:00Z',
              },
            ],
          },
        };
        fetchMock.mockResolvedValue(mockResponses.success(response));

        const result = await getAppLimits('my_app');

        expect(result.app_bundle_key).toBe('my_app');
        expect(result.limits.free).toHaveLength(1);
      });
    });
  });

  describe('Usage Management', () => {
    describe('getUsageSummary', () => {
      it('returns usage summary without params', async () => {
        const summary: UsageSummary = {
          user_identity: 'user@example.com',
          billing_period: '2024-01',
          tier: 'pro',
          limits: { ai_credits: 10000000 },
          usage: { ai_credits: 5000000 },
          remaining: { ai_credits: 5000000 },
          display_credits: { ai_credits: 5000 },
          reset_date: '2024-02-01T00:00:00Z',
        };
        fetchMock.mockResolvedValue(mockResponses.success(summary));

        const result = await getUsageSummary();

        expect(result.user_identity).toBe('user@example.com');
        expect(result.usage.ai_credits).toBe(5000000);
      });

      it('includes user param when provided', async () => {
        fetchMock.mockResolvedValue(
          mockResponses.success({
            user_identity: 'other@example.com',
            billing_period: '2024-01',
            limits: {},
            usage: {},
            remaining: {},
            display_credits: {},
            reset_date: '2024-02-01T00:00:00Z',
          })
        );

        await getUsageSummary('other@example.com');

        const [url] = getFetchCall(fetchMock);
        expect(url).toContain('user=other%40example.com');
      });

      it('includes tier param when provided', async () => {
        fetchMock.mockResolvedValue(
          mockResponses.success({
            user_identity: '',
            billing_period: '2024-01',
            limits: {},
            usage: {},
            remaining: {},
            display_credits: {},
            reset_date: '2024-02-01T00:00:00Z',
          })
        );

        await getUsageSummary(undefined, 'pro');

        const [url] = getFetchCall(fetchMock);
        expect(url).toContain('tier=pro');
      });
    });

    describe('getAdminUsageSummary', () => {
      it('returns admin usage summary', async () => {
        const summary = {
          billing_period: '2024-01',
          records: [],
          user_totals: { 'user@example.com': 5000000 },
          app_totals: { my_app: 2000000 },
          total_users: 10,
          total_records: 50,
        };
        fetchMock.mockResolvedValue(mockResponses.success(summary));

        const result = await getAdminUsageSummary();

        expect(result.total_users).toBe(10);
        expect(result.total_records).toBe(50);
      });

      it('includes period param when provided', async () => {
        fetchMock.mockResolvedValue(
          mockResponses.success({
            billing_period: '2024-06',
            records: [],
            user_totals: {},
            app_totals: {},
            total_users: 0,
            total_records: 0,
          })
        );

        await getAdminUsageSummary('2024-06');

        const [url] = getFetchCall(fetchMock);
        expect(url).toContain('period=2024-06');
      });
    });
  });

  describe('Helper Functions', () => {
    describe('formatCredits', () => {
      it('returns "Unlimited" for negative values', () => {
        expect(formatCredits(-1)).toBe('Unlimited');
      });

      it('returns "0" for zero', () => {
        expect(formatCredits(0)).toBe('0');
      });

      it('formats small values', () => {
        expect(formatCredits(100000)).toBe('1');
        expect(formatCredits(500000)).toBe('5');
      });

      it('formats large values with k suffix', () => {
        expect(formatCredits(100000000)).toBe('1.0k');
        expect(formatCredits(500000000)).toBe('5.0k');
      });
    });

    describe('formatDollars', () => {
      it('returns "Unlimited" for negative values', () => {
        expect(formatDollars(-1)).toBe('Unlimited');
      });

      it('returns "$0" for zero', () => {
        expect(formatDollars(0)).toBe('$0');
      });

      it('formats dollar amounts', () => {
        expect(formatDollars(100000000)).toBe('$1.00');
        expect(formatDollars(999000000)).toBe('$9.99');
      });

      it('formats large amounts with k suffix', () => {
        expect(formatDollars(100000000000)).toBe('$1.0k');
      });

      it('respects custom cost multiplier', () => {
        expect(formatDollars(1000, 100)).toBe('$0.10');
      });
    });

    describe('dollarsToInternalUnits', () => {
      it('converts dollars to internal units', () => {
        expect(dollarsToInternalUnits(1)).toBe(100000000);
        expect(dollarsToInternalUnits(9.99)).toBe(999000000);
      });

      it('respects custom cost multiplier', () => {
        expect(dollarsToInternalUnits(1, 100)).toBe(10000);
      });

      it('rounds to nearest integer', () => {
        expect(dollarsToInternalUnits(0.001)).toBe(100000);
      });
    });
  });

  describe('Constants', () => {
    describe('TIER_OPTIONS', () => {
      it('has expected tiers', () => {
        const values = TIER_OPTIONS.map((t) => t.value);
        expect(values).toContain('free');
        expect(values).toContain('pro');
        expect(values).toContain('business');
      });

      it('each option has value and label', () => {
        TIER_OPTIONS.forEach((option) => {
          expect(option.value).toBeTruthy();
          expect(option.label).toBeTruthy();
        });
      });
    });

    describe('PROVIDER_OPTIONS', () => {
      it('has expected providers', () => {
        const values = PROVIDER_OPTIONS.map((p) => p.value);
        expect(values).toContain('openrouter');
        expect(values).toContain('openai');
        expect(values).toContain('anthropic');
      });

      it('each option has value, label, and description', () => {
        PROVIDER_OPTIONS.forEach((option) => {
          expect(option.value).toBeTruthy();
          expect(option.label).toBeTruthy();
          expect(option.description).toBeTruthy();
        });
      });
    });
  });
});
