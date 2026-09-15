import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { getLandingConfig, getPlans, parseLandingConfigJson, recordPresentationExposure } from './landing';
import signalFixture from '../../surfaces/public-landing/presentation/fixtures/signal.json';
import { ResolvedProductPresentationSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';
import { publicConfig } from '../../surfaces/public-landing/presentation/publicTestFixtures';
import { LandingConfigResponseSchema, PresentationAssignmentSource } from '@vrooli/proto-types/landing-page-business-suite/v1/config_pb';
import { assertDefined, createFetchMock, installFetchMock, mockResponses } from '../test-utils/api-mocks';
import { create, toJson, fromJsonString, toJsonString } from '@bufbuild/protobuf';
import { GetPricingResponseSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/pricing_pb';
import { BillingInterval, IntroPricingType, PlanKind } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/commerce_pb';

const { pricingClient, landingConfigClient } = vi.hoisted(() => ({
  pricingClient: { getPricing: vi.fn() },
  landingConfigClient: { getLandingConfig: vi.fn(), recordPresentationExposure: vi.fn() },
}));

vi.mock('@connectrpc/connect', () => ({
  createClient: vi.fn((service: { typeName?: string }) => service.typeName?.endsWith('.LandingConfigService') ? landingConfigClient : pricingClient),
}));
vi.mock('@bufbuild/protobuf', async (importOriginal) => ({ ...(await importOriginal<typeof import('@bufbuild/protobuf')>()), toJson: vi.fn((_schema, message): unknown => message as unknown) }));

describe('landing API', () => {
  it('uses the generated explicit exposure request and preserves owner deduplication', async () => {
    const proof = { visitorId: 'visitor', variantSlug: 'control', revision: 'revision', route: '/', locale: 'en', blockDigest: 'digest', weightFingerprint: 'weights', source: PresentationAssignmentSource.WEIGHTED_VISITOR };
    landingConfigClient.recordPresentationExposure.mockResolvedValue({ recorded: false });
    expect(await recordPresentationExposure(proof)).toEqual({ recorded: false });
    expect(landingConfigClient.recordPresentationExposure).toHaveBeenCalledWith({ $typeName: 'landing_page_business_suite.v1.RecordPresentationExposureRequest', ...proof }, { timeoutMs: 10000 });
    expect(landingConfigClient.getLandingConfig).not.toHaveBeenCalled();
  });
  let fetchMock: ReturnType<typeof createFetchMock>;

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(toJson).mockReset();
    fetchMock = createFetchMock();
    installFetchMock(fetchMock);
    pricingClient.getPricing.mockImplementation(async () => {
      const response = await fetchMock('/landing_page_business_suite.v1.PricingService/GetPricing');
      assertDefined(response, 'Connect pricing response');
      assertDefined(response.json, 'Connect pricing response JSON reader');
      return response.json();
    });
    landingConfigClient.getLandingConfig.mockResolvedValue(create(LandingConfigResponseSchema, { presentation: publicConfig().presentation }));
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('getLandingConfig', () => {
    beforeEach(async () => {
      const actual = await vi.importActual<typeof import('@bufbuild/protobuf')>('@bufbuild/protobuf');
      vi.mocked(toJson).mockImplementation(actual.toJson);
    });

    it('requires the canonical typed presentation and retains the four public owner fields', async () => {
      const presentation = publicConfig().presentation;
      landingConfigClient.getLandingConfig.mockResolvedValue(create(LandingConfigResponseSchema, { presentation, fallback: true }));
      const result = await getLandingConfig();
      expect(result).toEqual({ presentation, pricing: undefined, downloads: [], fallback: true });
      expect(LandingConfigResponseSchema.fields.map(field => [field.name, field.number])).toEqual([
        ['pricing', 3], ['downloads', 4], ['fallback', 7], ['presentation', 10],
      ]);
    });

    it('preserves the complete generated page/display/fixtures/assets/actions/diagnostics projection by identity', async () => {
      const presentation = fromJsonString(ResolvedProductPresentationSchema, JSON.stringify(signalFixture.presentation));
      presentation.actions = [{ $typeName: 'vrooli.landing_page_business_suite.v1.shared.ResolvedPresentationAction',
        key: 'owner-key', status: 'unavailable', href: '', reason: 'Owner unavailable', appKey: '', planRef: '' }];
      const wire = create(LandingConfigResponseSchema, { presentation });
      landingConfigClient.getLandingConfig.mockResolvedValue(wire);
      const result = await getLandingConfig();
      expect(result.presentation).toBe(presentation);
      expect(parseLandingConfigJson(toJsonString(LandingConfigResponseSchema, wire)).presentation).toEqual(presentation);
    });

    it.each(['network', 'bootstrap'] as const)('fails closed without presentation at the %s boundary', async boundary => {
      const wire = create(LandingConfigResponseSchema, { downloads: [], fallback: true });
      if (boundary === 'network') {
        landingConfigClient.getLandingConfig.mockResolvedValue(wire);
        await expect(getLandingConfig()).rejects.toThrow('Invalid LandingConfigResponse response');
      } else expect(() => parseLandingConfigJson(toJsonString(LandingConfigResponseSchema, wire))).toThrow('Invalid LandingConfigResponse response');
    });

    it('normalizes the same generated owner pricing for public joins and standalone commerce', async () => {
      const wire = fromJsonString(LandingConfigResponseSchema, JSON.stringify({
        presentation: toJson(ResolvedProductPresentationSchema, publicConfig().presentation),
        pricing: {
          bundle: { bundle_key: 'suite', name: 'Owner Suite', stripe_product_id: 'prod', credits_per_usd: '1000000', display_credits_multiplier: 0.001, display_credits_label: 'credits' },
          monthly: [{ plan_name: 'Pro', plan_tier: 'pro', billing_interval: 'BILLING_INTERVAL_MONTH', amount_cents: '7900', currency: 'eur',
            intro_enabled: true, intro_type: 'INTRO_PRICING_TYPE_FLAT_AMOUNT', intro_amount_cents: '500', intro_periods: 2,
            stripe_price_id: 'price_pro', display_enabled: true, kind: 'PLAN_KIND_SUBSCRIPTION', metadata: { features: { list_value: { values: [{ string_value: 'Owner feature' }] } } } }],
          credit_topups: [{ plan_name: 'Credits', plan_tier: 'credits', billing_interval: 'BILLING_INTERVAL_ONE_TIME', amount_cents: '1000', kind: 'PLAN_KIND_CREDITS_TOPUP', stripe_price_id: 'price_credits', display_enabled: true }],
          updated_at: '2026-01-01T00:00:00Z',
        },
        downloads: [{
          bundle_key: 'suite', app_key: 'example', name: 'Owner app', metadata: { web_url: 'https://launch.example/app', enabled: true },
          install_steps: ['Install'], storefronts: [{ store: 'direct', label: 'Owner store', url: 'https://store.example/app', badge: 'Open' }], display_order: 3,
          platforms: [{ id: '9', bundle_key: 'suite', app_key: 'example', platform: 'linux', artifact_url: '/authorized/app',
            artifact_source: 'managed', artifact_id: '22', release_version: '1.2.3', release_notes: 'Owner release',
            checksum: 'sha', requires_entitlement: true, artifact_filename: 'app.AppImage', artifact_size_bytes: '4096', artifact_count: 1,
            metadata: { channel: 'stable' } }],
        }],
      }));
      landingConfigClient.getLandingConfig.mockResolvedValue(wire);
      pricingClient.getPricing.mockResolvedValue(create(GetPricingResponseSchema, { pricing: wire.pricing }));
      const result = await getLandingConfig();
      expect(result.pricing).toEqual(await getPlans());
      expect(result.pricing?.monthly[0]).toMatchObject({ billing_interval: 'month', amount_cents: 7900, currency: 'eur', intro_amount_cents: 500, intro_periods: 2, kind: 'subscription', metadata: { features: ['Owner feature'] } });
      expect(result.pricing?.credit_topups[0]).toMatchObject({ stripe_price_id: 'price_credits', kind: 'credits_topup', billing_interval: 'one_time' });
      expect(result.downloads[0]).toMatchObject({ bundle_key: 'suite', app_key: 'example', metadata: { web_url: 'https://launch.example/app', enabled: true }, storefronts: [{ badge: 'Open' }], display_order: 3 });
      expect(result.downloads[0]?.platforms[0]).toMatchObject({ id: 9, artifact_id: 22, artifact_source: 'managed', artifact_url: '/authorized/app', release_version: '1.2.3', requires_entitlement: true, artifact_size_bytes: 4096, artifact_count: 1, metadata: { channel: 'stable' } });
    });

    it('preserves typed nested pricing metadata and unsafe int64 precision through the real generated codec', async () => {
      const wire = fromJsonString(LandingConfigResponseSchema, JSON.stringify({
        presentation: toJson(ResolvedProductPresentationSchema, publicConfig().presentation),
        pricing: { bundle: { bundle_key: 'suite', name: 'Owner suite', stripe_product_id: 'prod' }, monthly: [{
          plan_name: 'Owner plan', plan_tier: 'configured', billing_interval: 'BILLING_INTERVAL_MONTH', amount_cents: '2500', currency: 'usd', stripe_price_id: 'owner-price',
          metadata: {
            credits: { int_value: '42' }, exact_counter: { int_value: '9007199254740993' },
            details: { object_value: { fields: { enabled: { bool_value: false }, empty: { null_value: 0 },
              rows: { list_value: { values: [{ string_value: 'Configured row' }, { int_value: '7' }, { double_value: 1.25 }] } } } } },
          },
        }], updated_at: '2026-01-01T00:00:00Z' },
      }));
      landingConfigClient.getLandingConfig.mockResolvedValue(wire);
      pricingClient.getPricing.mockResolvedValue(create(GetPricingResponseSchema, { pricing: wire.pricing }));
      const result = await getLandingConfig();
      expect(result.pricing).toEqual(await getPlans());
      expect(result.pricing?.updated_at).toBe('2026-01-01T00:00:00Z');
      expect(result.pricing?.monthly[0]?.metadata).toEqual({ credits: 42, exact_counter: '9007199254740993', details: { enabled: false, empty: null, rows: ['Configured row', 7, 1.25] } });
    });

    it('keeps a sparse owner download row sparse instead of fabricating artifact metadata or entitlement', async () => {
      const wire = fromJsonString(LandingConfigResponseSchema, JSON.stringify({
        presentation: toJson(ResolvedProductPresentationSchema, publicConfig().presentation),
        downloads: [{ bundle_key: 'suite', app_key: 'example', name: 'Owner app',
          storefronts: [{ store: 'direct', label: 'Owner store', url: 'https://store.example/app' }],
          platforms: [{ id: '23', bundle_key: 'suite', app_key: 'example', platform: 'linux', release_version: '1.2.3' }],
        }],
      }));
      landingConfigClient.getLandingConfig.mockResolvedValue(wire);
      const result = await getLandingConfig();
      const app = result.downloads[0]; const asset = app?.platforms[0];
      expect(app?.storefronts?.[0]?.badge).toBeUndefined();
      expect(asset).toMatchObject({ id: 23, artifact_url: '', release_version: '1.2.3', requires_entitlement: false, metadata: {} });
      expect(asset?.artifact_source).toBeUndefined(); expect(asset?.release_notes).toBeUndefined();
      expect(asset?.checksum).toBeUndefined(); expect(asset?.artifact_filename).toBeUndefined();
      expect(asset?.artifact_size_bytes).toBeUndefined(); expect(asset?.artifact_count).toBeUndefined();
    });

    it.each(['variant', 'sections', 'header', 'branding', 'coupon_mappings', 'intro_offers', 'private_extra'])('rejects retired/unknown bootstrap field %s through the generated descriptor', field => {
      const raw = toJson(LandingConfigResponseSchema, create(LandingConfigResponseSchema, { presentation: publicConfig().presentation }));
      if (!raw || typeof raw !== 'object' || Array.isArray(raw)) throw new Error('Expected fixture object');
      expect(() => parseLandingConfigJson(JSON.stringify({ ...raw, [field]: {} }))).toThrow();
    });

    it('keeps default request identity empty and forwards explicit route, locale and abort signal', async () => {
      await getLandingConfig();
      expect(landingConfigClient.getLandingConfig).toHaveBeenCalledWith(expect.objectContaining({ variantSlug: '', visitorId: '', route: '/', locale: '' }), expect.anything());
      const signal = new AbortController().signal;
      await getLandingConfig('review', 'visitor', { route: '/apps/example', locale: 'fr', signal });
      expect(landingConfigClient.getLandingConfig).toHaveBeenLastCalledWith(expect.objectContaining({ variantSlug: 'review', visitorId: 'visitor', route: '/apps/example', locale: 'fr' }), { signal });
    });

    it('propagates an owner failure without inventing config', async () => {
      landingConfigClient.getLandingConfig.mockRejectedValue(new Error('Connect unavailable'));
      await expect(getLandingConfig()).rejects.toThrow('Connect unavailable');
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

      expect(pricingClient.getPricing).toHaveBeenCalledWith({});
      expect(result.bundle.bundle_key).toBe('main');
      expect(result.bundle.name).toBe('Main Bundle');
      expect(result.monthly).toHaveLength(1);
      expect(result.monthly[0]?.plan_name).toBe('Pro');
      expect(result.yearly).toHaveLength(1);
      expect(result.yearly[0]?.plan_name).toBe('Pro Annual');
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
      expect(result.monthly[0]?.plan_name).toBe('Monthly');
      expect(result.yearly[0]?.plan_name).toBe('Yearly');
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
      expect(result.monthly[0]?.plan_name).toBe('Plan1');
      expect(result.monthly[1]?.plan_name).toBe('Plan2');
      expect(result.monthly[2]?.plan_name).toBe('Plan3');
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

      expect(result.monthly[0]?.intro_enabled).toBe(true);
      expect(result.monthly[0]?.intro_amount_cents).toBe(4900);
      expect(result.monthly[0]?.intro_periods).toBe(3);
      expect(result.monthly[0]?.intro_price_lookup_key).toBe('pro_intro');
    });

    it('defaults numeric fields to 0 when missing', async () => {
      const protoResponse = {
        pricing: {
          bundle: { bundleKey: 'main', name: 'Main', stripeProductId: 'prod_123' },
          monthly: [{ planName: 'Basic' }, {}],
          yearly: [],
        },
      };
      fetchMock.mockResolvedValue(mockResponses.success(protoResponse));

      const result = await getPlans();

      expect(result.monthly[0]?.amount_cents).toBe(0);
      expect(result.monthly[0]?.monthly_included_credits).toBe(0);
      expect(result.monthly[0]?.one_time_bonus_credits).toBe(0);
      expect(result.monthly[0]?.display_weight).toBe(0);
    });

    it('maps plan kinds, intervals, intros, and metadata from generated proto values', async () => {
      const protoResponse = create(GetPricingResponseSchema, {
        pricing: {
          bundle: {
            bundleKey: 'main',
            name: 'Main',
            stripeProductId: 'prod_123',
            metadata: { source: { kind: { case: 'stringValue', value: 'seeded' } }, plain: { kind: { case: 'nullValue', value: 0 } } },
          },
          monthly: [
            {
              planName: 'Top up', planTier: 'credits', amountCents: 500n, currency: 'usd',
              kind: PlanKind.CREDITS_TOPUP, planRank: 1,
              billingInterval: BillingInterval.ONE_TIME,
              introType: IntroPricingType.PERCENTAGE,
              introAmountCents: 20n, introPeriods: 2, metadata: { label: { kind: { case: 'stringValue', value: 'popular' } } },
            },
          ],
          yearly: [{
            planName: 'Support', planTier: 'support', amountCents: 100n, currency: 'usd',
            kind: PlanKind.SUPPORTER_CONTRIBUTION,
            billingInterval: BillingInterval.YEAR,
            introType: IntroPricingType.FLAT_AMOUNT,
          }],
        },
      });
      const actual = await vi.importActual<typeof import('@bufbuild/protobuf')>('@bufbuild/protobuf');
      vi.mocked(toJson).mockImplementationOnce(actual.toJson);
      pricingClient.getPricing.mockResolvedValue(protoResponse);

      const result = await getPlans();

      expect(result.monthly).toEqual([expect.objectContaining({
        kind: 'credits_topup', billing_interval: 'one_time', intro_type: 'percentage',
        intro_amount_cents: 20, intro_periods: 2, metadata: { label: 'popular' },
      })]);
      expect(result.yearly).toEqual([expect.objectContaining({
        kind: 'supporter_contribution', billing_interval: 'year', intro_type: 'flat_amount',
      })]);
      expect(result.bundle.metadata).toEqual({ source: 'seeded', plain: null });
    });

    it('rejects malformed numeric plans instead of exposing invalid prices to checkout', async () => {
      const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
      const error = vi.spyOn(console, 'error').mockImplementation(() => undefined);
      fetchMock.mockResolvedValue(mockResponses.success({
        pricing: {
          bundle: { bundleKey: 'main', name: 'Main', stripeProductId: 'prod_123', creditsPerUsd: 'not-a-number' },
          monthly: [{ planName: 'Broken', planTier: 'pro', amountCents: 'not-a-number', currency: 'usd' }],
          yearly: [],
        },
      }));

      await expect(getPlans()).rejects.toThrow('Invalid pricing amount');
      warn.mockRestore();
      error.mockRestore();
    });

    it('rejects a successful response with missing pricing instead of inventing an offer', async () => {
      fetchMock.mockResolvedValue(mockResponses.success({}));

      await expect(getPlans()).rejects.toThrow('Invalid PricingOverview response');
    });
  });
});
