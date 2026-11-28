import { useContext, useEffect, useMemo, useState } from 'react';
import { useHref } from 'react-router-dom';
import type { PublicBranding } from '../../../shared/api/types';
import { LandingVariantContext } from '../../../app/providers/LandingVariantContext';
import { resolvePublicPresentation } from '../presentation/publicIntegration';
import { resolveResources } from '../presentation/resolvedResources';
import { safeHref } from '../presentation/links';
import { loadSiteBranding } from './siteBrandingSource';
import type { Action, ResolvedActions } from '../presentation/types';

/** Everything the site chrome and site-owned pages need to identify the business. */
export interface SiteIdentity {
  /** Public brand shown in the header and footer. */
  brandName: string;
  /** Same-origin logo path, when one is configured. */
  brandLogo?: string;
  tagline?: string;
  copyright: string;
  /** Registered business name, falling back to the public brand. */
  businessName: string;
  contactEmail?: string;
  /** Postal address lines. */
  addressLines: string[];
  website?: string;
  privacyMarkdown?: string;
  termsMarkdown?: string;
  privacyEffectiveDate?: string;
  termsEffectiveDate?: string;
  /** The landing page's header action, when the published presentation supplies one. */
  headerAction?: { action: Action; resolvedActions: ResolvedActions; reason: string };
  /** True while public branding is still loading. */
  loading: boolean;
}

type ChromeSnapshot = Pick<SiteIdentity, 'brandName' | 'brandLogo' | 'tagline' | 'copyright'>;

const CHROME_KEY = 'lpbs:site-chrome';
function readChrome(): ChromeSnapshot | undefined {
  try {
    const raw = window.sessionStorage.getItem(CHROME_KEY);
    const value = raw ? JSON.parse(raw) as Partial<ChromeSnapshot> : undefined;
    return value && typeof value.brandName === 'string' && typeof value.copyright === 'string' ? value as ChromeSnapshot : undefined;
  } catch { return undefined; }
}

function writeChrome(value: ChromeSnapshot) {
  try { window.sessionStorage.setItem(CHROME_KEY, JSON.stringify(value)); } catch { /* storage is a convenience only */ }
}

const trimmed = (value?: string | null) => value?.trim() || undefined;

/**
 * Resolves the business identity for site-owned pages. The published landing
 * presentation owns the visible brand, so site pages match the landing page
 * exactly; Branding settings own the business facts (legal name, contact
 * details, legal documents) and supply the brand when no presentation is loaded.
 */
export function useSiteIdentity(): SiteIdentity {
  const [branding, setBranding] = useState<PublicBranding | null | undefined>(undefined);
  // Optional: sign-in and embedded surfaces render without the public presentation provider.
  const landing = useContext(LandingVariantContext);
  const config = landing?.config;
  const request = landing?.request;
  const base = useHref('/');

  useEffect(() => {
    let live = true;
    void loadSiteBranding().then(value => { if (live) setBranding(value); });
    return () => { live = false; };
  }, []);

  const presentation = useMemo(() => {
    if (!config?.presentation || request?.route !== '/') return undefined;
    try {
      const resolved = resolvePublicPresentation(config.presentation, request, base, config.downloads);
      return { ...resolved, shell: resolveResources(resolved.presentation).shell };
    } catch { return undefined; }
  }, [config, request, base]);

  return useMemo(() => {
    const shell = presentation?.shell;
    const siteName = trimmed(branding?.site_name);
    const logo = [shell?.footer_brand_logo, branding?.logo_icon_url, branding?.logo_url].map(value => trimmed(value)).find(value => value && safeHref(value)?.startsWith('/'));
    const fromPresentation: ChromeSnapshot | undefined = shell ? {
      brandName: shell.footer_brand_name, brandLogo: logo, tagline: shell.footer_tagline, copyright: shell.copyright,
    } : undefined;
    if (fromPresentation) writeChrome(fromPresentation);
    const year = new Date().getFullYear();
    const chrome = fromPresentation ?? readChrome() ?? {
      brandName: siteName ?? '', brandLogo: logo, tagline: trimmed(branding?.tagline),
      copyright: siteName ? `${trimmed(branding?.legal_name) ?? siteName} © ${String(year)}` : '',
    };
    const businessName = trimmed(branding?.legal_name) ?? chrome.brandName;
    return {
      ...chrome,
      businessName,
      contactEmail: trimmed(branding?.support_email),
      addressLines: (branding?.contact_address ?? '').split('\n').map(line => line.trim()).filter(Boolean),
      website: trimmed(branding?.canonical_base_url),
      privacyMarkdown: trimmed(branding?.privacy_policy_markdown),
      termsMarkdown: trimmed(branding?.terms_markdown),
      privacyEffectiveDate: trimmed(branding?.privacy_effective_date),
      termsEffectiveDate: trimmed(branding?.terms_effective_date),
      headerAction: shell?.header_action && presentation ? { action: shell.header_action, resolvedActions: presentation.resolvedActions, reason: shell.unavailable_reason } : undefined,
      loading: branding === undefined,
    };
  }, [branding, presentation]);
}
