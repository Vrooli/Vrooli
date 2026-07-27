import {
  getBranding,
  updateBranding,
  clearBrandingField,
  type SiteBranding,
  type Asset,
} from '../../../shared/api';
import { isFormDirty } from '../../../shared/lib/formUtils';

/**
 * Form state for branding configuration
 */
export interface BrandingFormState {
  site_name: string;
  tagline: string;
  logo_url: string;
  logo_icon_url: string;
  favicon_url: string;
  apple_touch_icon_url: string;
  default_title: string;
  default_description: string;
  default_og_image_url: string;
  theme_primary_color: string;
  theme_background_color: string;
  canonical_base_url: string;
  google_site_verification: string;
  robots_txt: string;
  support_chat_url: string;
  support_email: string;
  smtp_host: string;
  smtp_port: string;
  smtp_username: string;
  smtp_password: string;
  smtp_from: string;
  coming_soon_enabled: boolean;
  coming_soon_message: string;
}

/**
 * Branding health check results
 */
export interface BrandingHealthChecks {
  identity: boolean;
  favicon: boolean;
  seo: boolean;
  ogImage: boolean;
}

export interface BrandingHealth {
  checks: BrandingHealthChecks;
  configured: number;
  total: number;
  percentage: number;
}

/**
 * Logo derivative selection result
 */
export interface LogoDerivatives {
  logo_url: string;
  logo_icon_url: string;
  favicon_url: string;
  apple_touch_icon_url: string;
}

/**
 * Favicon derivative selection result
 */
export interface FaviconDerivatives {
  favicon_url: string;
  apple_touch_icon_url: string;
}

/**
 * OG image derivative selection result
 */
export interface OgDerivatives {
  default_og_image_url: string;
}

/**
 * Default empty form state
 */
export const DEFAULT_BRANDING_FORM: BrandingFormState = {
  site_name: '',
  tagline: '',
  logo_url: '',
  logo_icon_url: '',
  favicon_url: '',
  apple_touch_icon_url: '',
  default_title: '',
  default_description: '',
  default_og_image_url: '',
  theme_primary_color: '',
  theme_background_color: '',
  canonical_base_url: '',
  google_site_verification: '',
  robots_txt: '',
  support_chat_url: '',
  support_email: '',
  smtp_host: '',
  smtp_port: '587',
  smtp_username: '',
  smtp_password: '',
  smtp_from: '',
  coming_soon_enabled: false,
  coming_soon_message: '',
};

/**
 * Convert API SiteBranding to form state
 */
export function brandingToForm(branding: SiteBranding): BrandingFormState {
  return {
    site_name: branding.site_name,
    tagline: branding.tagline ?? '',
    logo_url: branding.logo_url ?? '',
    logo_icon_url: branding.logo_icon_url ?? '',
    favicon_url: branding.favicon_url ?? '',
    apple_touch_icon_url: branding.apple_touch_icon_url ?? '',
    default_title: branding.default_title ?? '',
    default_description: branding.default_description ?? '',
    default_og_image_url: branding.default_og_image_url ?? '',
    theme_primary_color: branding.theme_primary_color ?? '',
    theme_background_color: branding.theme_background_color ?? '',
    canonical_base_url: branding.canonical_base_url ?? '',
    google_site_verification: branding.google_site_verification ?? '',
    robots_txt: branding.robots_txt ?? '',
    support_chat_url: branding.support_chat_url ?? '',
    support_email: branding.support_email ?? '',
    smtp_host: branding.smtp_host ?? '',
    smtp_port: branding.smtp_port?.toString() ?? '587',
    smtp_username: branding.smtp_username ?? '',
    smtp_password: branding.smtp_password ?? '',
    smtp_from: branding.smtp_from ?? '',
    coming_soon_enabled: branding.coming_soon_enabled ?? false,
    coming_soon_message: branding.coming_soon_message ?? '',
  };
}

/**
 * Convert form state to API update payload
 * Only includes fields that have changed from the original
 */
export function formToBrandingPayload(
  form: BrandingFormState,
  original: BrandingFormState
): Record<string, string | number | boolean> {
  const payload: Record<string, string | number | boolean> = {};

  (Object.keys(form) as (keyof BrandingFormState)[]).forEach((key) => {
    const current = form[key];
    const originalValue = original[key];

    // Handle boolean fields (coming_soon_enabled)
    if (key === 'coming_soon_enabled') {
      const currentBool = Boolean(current);
      const originalBool = Boolean(originalValue);
      if (currentBool !== originalBool) {
        payload[key] = currentBool;
      }
      return;
    }

    // Handle string fields - ensure we have strings
    const currentStr = String(current).trim();
    const originalStr = String(originalValue).trim();
    if (currentStr !== originalStr && currentStr.length > 0) {
      // Convert smtp_port to number
      if (key === 'smtp_port') {
        payload[key] = parseInt(currentStr, 10);
      } else {
        payload[key] = currentStr;
      }
    }
  });

  return payload;
}

/**
 * Check if branding form has any changes from original
 */
export function isBrandingDirty(
  form: BrandingFormState,
  original: BrandingFormState
): boolean {
  return isFormDirty(form, original);
}

/**
 * Compute branding setup health/completeness
 */
export function computeBrandingHealth(form: BrandingFormState): BrandingHealth {
  const checks: BrandingHealthChecks = {
    identity: Boolean(form.site_name && form.logo_url),
    favicon: Boolean(form.favicon_url),
    seo: Boolean(form.default_title && form.default_description),
    ogImage: Boolean(form.default_og_image_url),
  };

  const configured = Object.values(checks).filter(Boolean).length;
  const total = Object.keys(checks).length;
  const percentage = Math.round((configured / total) * 100);

  return { checks, configured, total, percentage };
}

/**
 * Select appropriate logo derivatives from an uploaded asset
 * Returns URLs for logo, icon, favicon, and apple touch icon
 */
export function selectLogoDerivatives(
  asset: Asset,
  currentForm: BrandingFormState
): LogoDerivatives {
  const primaryLogo =
    asset.derivatives?.logo_512 ||
    asset.derivatives?.logo_256 ||
    asset.url ||
    currentForm.logo_url;

  const iconLogo =
    asset.derivatives?.logo_icon ||
    asset.derivatives?.logo_256 ||
    asset.derivatives?.logo_128 ||
    currentForm.logo_icon_url ||
    primaryLogo;

  const favicon =
    asset.derivatives?.favicon_32 ||
    asset.derivatives?.favicon_64 ||
    asset.derivatives?.favicon ||
    currentForm.favicon_url;

  const touch =
    asset.derivatives?.apple_touch_180 ||
    currentForm.apple_touch_icon_url ||
    favicon;

  return {
    logo_url: primaryLogo,
    logo_icon_url: iconLogo,
    favicon_url: favicon,
    apple_touch_icon_url: touch,
  };
}

/**
 * Select appropriate favicon derivatives from an uploaded asset
 * Returns URLs for favicon and apple touch icon
 */
export function selectFaviconDerivatives(
  asset: Asset,
  currentForm: BrandingFormState
): FaviconDerivatives {
  const favicon =
    asset.derivatives?.favicon ||
    asset.derivatives?.favicon_32 ||
    asset.derivatives?.favicon_64 ||
    asset.url ||
    currentForm.favicon_url;

  const touch =
    asset.derivatives?.apple_touch_180 ||
    currentForm.apple_touch_icon_url ||
    favicon;

  return {
    favicon_url: favicon,
    apple_touch_icon_url: touch,
  };
}

/**
 * Select appropriate OG image derivative from an uploaded asset
 */
export function selectOgDerivatives(asset: Asset): OgDerivatives {
  const og = asset.derivatives?.og_image_1200x630 || asset.url;

  return {
    default_og_image_url: typeof og === 'string' ? og : '',
  };
}

/**
 * Load branding from API
 */
export async function loadBranding(): Promise<SiteBranding> {
  return getBranding();
}

/**
 * Save branding changes to API
 * Returns the updated branding data
 */
export async function saveBranding(
  form: BrandingFormState,
  original: BrandingFormState
): Promise<SiteBranding> {
  const payload = formToBrandingPayload(form, original);

  if (Object.keys(payload).length === 0) {
    throw new Error('No changes to save');
  }

  return updateBranding(payload);
}

/**
 * Clear a specific branding field via API
 */
export async function clearField(field: keyof BrandingFormState): Promise<SiteBranding> {
  return clearBrandingField(field);
}

/**
 * Format field name for display (converts snake_case to human-readable)
 */
export function formatFieldName(field: string): string {
  return field.replace(/_/g, ' ');
}
