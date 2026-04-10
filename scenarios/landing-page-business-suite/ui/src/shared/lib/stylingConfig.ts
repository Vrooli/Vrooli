import rawStylingConfig from '../../../../.vrooli/styling.json';

export type StylingConfig = typeof rawStylingConfig;

export interface VariantStylingGuidance {
  promise?: string;
  primary_cta?: string;
  secondary_cta?: string;
  notes?: string[];
}

export interface VariantStylingGuidanceWithKey extends VariantStylingGuidance {
  key: string;
}

const stylingConfig: StylingConfig = rawStylingConfig;

function variantGuidanceMap(): Record<string, VariantStylingGuidance> {
  const entries = stylingConfig.variant_guidance ?? {};
  return entries as unknown as Record<string, VariantStylingGuidance>;
}

export function getStylingConfig(): StylingConfig {
  return stylingConfig;
}

export function getVariantStylingGuidance(slug?: string): VariantStylingGuidanceWithKey {
  const map = variantGuidanceMap();
  const normalizedKey = slug && map[slug] ? slug : 'control';
  return {
    key: normalizedKey,
    ...map[normalizedKey],
  };
}
