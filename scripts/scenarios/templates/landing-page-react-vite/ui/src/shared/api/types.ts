export type VariantAxes = Record<string, string>;

export interface VariantSpaceAxisVariant {
  id: string;
  label: string;
  description?: string;
  examples?: Record<string, string>;
  defaultWeight?: number;
  tags?: string[];
  status?: 'active' | 'experimental' | 'deprecated';
  agentHints?: string[];
}

export interface VariantSpaceAxis {
  _note?: string;
  variants: VariantSpaceAxisVariant[];
}

export interface VariantSpaceConstraints {
  _note?: string;
  disallowedCombinations?: Record<string, string>[];
}

export interface VariantSpace {
  _name: string;
  _schemaVersion: number;
  _note?: string;
  _agentGuidelines?: string[];
  axes: Record<string, VariantSpaceAxis>;
  constraints?: VariantSpaceConstraints;
}

export interface Variant {
  id?: number;
  slug: string;
  name: string;
  description?: string;
  weight?: number;
  status?: 'active' | 'archived' | 'deleted' | 'fallback';
  created_at?: string;
  updated_at?: string;
  archived_at?: string;
  axes?: VariantAxes;
}

export interface BundleProduct {
  bundle_key: string;
  name: string;
  stripe_product_id: string;
  credits_per_usd: number;
  display_credits_multiplier: number;
  display_credits_label: string;
}

export interface PlanOption {
  plan_name: string;
  plan_tier: string;
  billing_interval: 'month' | 'year';
  amount_cents: number;
  currency: string;
  intro_enabled: boolean;
  intro_amount_cents?: number;
  intro_periods?: number;
  intro_price_lookup_key?: string;
  stripe_price_id: string;
  monthly_included_credits: number;
  one_time_bonus_credits: number;
  metadata?: Record<string, unknown>;
}

export interface PricingOverview {
  bundle: BundleProduct;
  monthly: PlanOption[];
  yearly: PlanOption[];
  updated_at: string;
}

export interface DownloadAsset {
  id?: number;
  bundle_key: string;
  platform: string;
  artifact_url: string;
  release_version: string;
  release_notes?: string;
  requires_entitlement: boolean;
  metadata?: Record<string, unknown>;
}

export type SectionType =
  | 'hero'
  | 'features'
  | 'pricing'
  | 'cta'
  | 'testimonials'
  | 'faq'
  | 'footer'
  | 'video';

export interface ContentSection {
  id: number;
  variant_id: number;
  section_type: SectionType;
  content: Record<string, unknown>;
  order: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface LandingSection {
  id?: number;
  section_type: string;
  content: Record<string, unknown>;
  order: number;
  enabled?: boolean;
}

export interface LandingConfigResponse {
  variant: {
    id?: number;
    slug: string;
    name: string;
    description?: string;
    axes?: VariantAxes;
  };
  sections: LandingSection[];
  pricing?: PricingOverview;
  downloads: DownloadAsset[];
  fallback: boolean;
}

export interface MetricEvent {
  event_type: 'page_view' | 'scroll_depth' | 'click' | 'form_submit' | 'conversion' | 'download';
  variant_id: number;
  session_id: string;
  visitor_id?: string;
  event_data?: Record<string, unknown>;
}

export interface AnalyticsSummary {
  total_visitors: number;
  total_downloads?: number;
  variant_stats: VariantStats[];
  top_cta?: string;
  top_cta_ctr?: number;
}

export interface VariantStats {
  variant_id: number;
  variant_slug: string;
  variant_name: string;
  views: number;
  cta_clicks: number;
  conversions: number;
  downloads: number;
  conversion_rate: number;
  avg_scroll_depth?: number;
  trend?: 'up' | 'down' | 'stable';
}

export interface SubscriptionInfo {
  status: string;
  subscription_id?: string;
  customer_email?: string;
  plan_tier?: string;
  price_id?: string;
  bundle_key?: string;
  updated_at?: string;
}

export interface CreditInfo {
  customer_email: string;
  balance_credits: number;
  bonus_credits: number;
  display_credits_label: string;
  display_credits_multiplier: number;
}

export interface EntitlementPayload {
  status: string;
  plan_tier?: string;
  price_id?: string;
  features?: string[];
  credits?: CreditInfo;
  subscription?: SubscriptionInfo;
}
