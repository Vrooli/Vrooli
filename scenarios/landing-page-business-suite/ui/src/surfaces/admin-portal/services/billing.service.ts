// DOC: docs/reference/api/payments.md - Stripe integration API
// DOC: docs/reference/STRIPE_WEBHOOKS.md - Webhook configuration
// DOC: docs/guides/ADMIN_GUIDE.md#stripe-setup - Admin Stripe setup
import {
  getStripeSettings,
  updateStripeSettings,
  getBundleCatalog,
  updateBundlePrice,
  verifyStripePrice,
  type StripeSettingsResponse,
  type BundleCatalogEntry,
} from '../../../shared/api';
import { parseFeaturesText, type PriceFormState } from './pricing.service';

/**
 * Form state for Stripe configuration
 */
export interface StripeFormState {
  publishableKey: string;
  secretKey: string;
  webhookSecret: string;
  dashboardUrl: string;
}

/**
 * Status check for Stripe price verification
 */
export interface PriceVerificationResult {
  status: 'idle' | 'checking' | 'ok' | 'error';
  message?: string;
}

/**
 * Default empty Stripe form state
 */
export const DEFAULT_STRIPE_FORM: StripeFormState = {
  publishableKey: '',
  secretKey: '',
  webhookSecret: '',
  dashboardUrl: '',
};

/**
 * Load Stripe settings from API
 */
export async function loadStripeSettings(): Promise<StripeSettingsResponse> {
  return getStripeSettings();
}

/**
 * Save Stripe settings to API
 * Only sends non-empty fields
 */
export async function saveStripeSettings(form: StripeFormState): Promise<StripeSettingsResponse> {
  const payload: Record<string, string> = {};
  (Object.keys(form) as (keyof StripeFormState)[]).forEach((key) => {
    const value = form[key].trim();
    if (value.length > 0) {
      const apiKey = key === 'dashboardUrl' ? 'dashboard_url' : key.replace(/[A-Z]/g, (match) => `_${match.toLowerCase()}`);
      payload[apiKey] = value;
    }
  });

  if (Object.keys(payload).length === 0) {
    throw new Error('Enter at least one field before saving.');
  }

  return updateStripeSettings(payload);
}

/**
 * Check if any Stripe settings have values entered
 */
export function hasStripeFormValues(form: StripeFormState): boolean {
  return Object.values(form).some((value) => value.trim().length > 0);
}

/**
 * Load bundle catalog from API
 */
export async function loadBundleCatalog(): Promise<{ bundles: BundleCatalogEntry[] }> {
  return getBundleCatalog();
}

/**
 * Save price form values to API
 */
export async function savePriceForm(
  bundleKey: string,
  priceId: string,
  formState: PriceFormState
): Promise<void> {
  const features = parseFeaturesText(formState.values.featuresText);
  const stripePriceId = formState.values.stripePriceId.trim();

  await updateBundlePrice(bundleKey, priceId, {
    stripe_price_id: stripePriceId === '' ? '' : stripePriceId,
    plan_name: formState.values.planName.trim() || undefined,
    display_weight: formState.values.displayWeight,
    display_enabled: formState.values.displayEnabled,
    subtitle: formState.values.subtitle.trim() || undefined,
    badge: formState.values.badge.trim() || undefined,
    cta_label: formState.values.ctaLabel.trim() || undefined,
    highlight: formState.values.highlight,
    features,
  });
}

/**
 * Verify a Stripe price ID or lookup key
 */
export async function verifyPriceId(priceIdOrKey: string): Promise<PriceVerificationResult> {
  const value = priceIdOrKey.trim();
  if (!value) {
    return {
      status: 'error',
      message: 'Enter a Stripe price ID or lookup key',
    };
  }

  try {
    const info = await verifyStripePrice(value);
    const parts = [
      info.id ? `ID ${info.id}` : null,
      info.lookup_key ? `lookup ${info.lookup_key}` : null,
      info.interval ? info.interval : null,
      info.currency ? info.currency.toUpperCase() : null,
      info.active === false ? 'inactive' : null,
    ].filter(Boolean);

    return {
      status: 'ok',
      message: parts.length > 0 ? parts.join(' · ') : 'Verified',
    };
  } catch (error) {
    return {
      status: 'error',
      message: error instanceof Error ? error.message : 'Verification failed',
    };
  }
}

/**
 * Check if all three Stripe keys are configured (publishable, secret, webhook).
 * Use this to determine if Stripe is fully ready for production use.
 */
export function isStripeFullyConfigured(settings: StripeSettingsResponse | null): boolean {
  if (!settings) return false;
  return settings.publishable_key_set && settings.secret_key_set && settings.webhook_secret_set;
}

/**
 * Check if at least one Stripe key is configured.
 * Use this to determine if the user has started Stripe setup.
 */
export function isStripePartiallyConfigured(settings: StripeSettingsResponse | null): boolean {
  if (!settings) return false;
  return settings.publishable_key_set || settings.secret_key_set || settings.webhook_secret_set;
}

/**
 * Build status badges for Stripe configuration.
 * Uses the *_set flags as the single source of truth for whether each key is configured.
 * Note: publishable_key_preview is only populated when publishable_key_set is true,
 * so we don't need to check both.
 */
export function buildStripeStatusBadges(settings: StripeSettingsResponse | null): Array<{ label: string; ok: boolean }> {
  if (!settings) {
    return [];
  }

  return [
    { label: 'Publishable Key', ok: settings.publishable_key_set },
    { label: 'Restricted Key', ok: settings.secret_key_set },
    { label: 'Webhook Secret', ok: settings.webhook_secret_set },
  ];
}
