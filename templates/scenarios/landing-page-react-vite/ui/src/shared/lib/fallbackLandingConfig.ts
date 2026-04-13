import type { LandingConfigResponse, LandingSection, DownloadApp, PricingOverview, VariantAxes, LandingHeaderConfig } from '../api';
import rawFallback from '../../../../.vrooli/variants/fallback.json';
import { normalizeHeaderConfig } from './headerConfig';

interface FallbackPayload {
  variant: LandingConfigResponse['variant'];
  sections?: LandingSection[] | null;
  pricing?: PricingOverview;
  downloads?: DownloadApp[];
  axes?: VariantAxes;
  header?: LandingHeaderConfig;
}

function normalizeSections(sectionsInput?: LandingSection[] | null): LandingSection[] {
  const sections = Array.isArray(sectionsInput) ? sectionsInput : [];

  return sections.map((section, index) => ({
    ...section,
    order: typeof section.order === 'number' && Number.isFinite(section.order) ? section.order : index + 1,
    enabled: section.enabled ?? true,
  }));
}

const FALLBACK_CONFIG: LandingConfigResponse = (() => {
  const payload = rawFallback as FallbackPayload;
  const variant = {
    ...payload.variant,
    axes: payload.variant.axes ?? payload.axes,
  };

  return {
    variant,
    sections: normalizeSections(payload.sections),
    pricing: payload.pricing,
    downloads: payload.downloads ?? [],
    header: normalizeHeaderConfig(payload.header, variant.name),
    fallback: true,
  };
})();

export function getFallbackLandingConfig(): LandingConfigResponse {
  return JSON.parse(JSON.stringify(FALLBACK_CONFIG));
}
