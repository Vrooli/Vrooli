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
  header_config?: LandingHeaderConfig;
}

export interface BundleProduct {
  bundle_key: string;
  name: string;
  stripe_product_id: string;
  credits_per_usd: number;
  display_credits_multiplier: number;
  display_credits_label: string;
  environment?: string;
  metadata?: Record<string, unknown>;
}

export interface PlanDisplayMetadata {
  subtitle?: string;
  badge?: string;
  cta_label?: string;
  highlight?: boolean;
  features?: string[];
  [key: string]: unknown;
}

export interface PlanOption {
  plan_name: string;
  plan_tier: string;
  billing_interval: 'month' | 'year' | 'one_time';
  amount_cents: number;
  currency: string;
  intro_enabled: boolean;
  intro_amount_cents?: number;
  intro_periods?: number;
  intro_price_lookup_key?: string;
  stripe_price_id: string;
  monthly_included_credits: number;
  one_time_bonus_credits: number;
  plan_rank?: number;
  bonus_type?: string;
  kind?: string;
  is_variable_amount?: boolean;
  display_enabled: boolean;
  bundle_key?: string;
  display_weight: number;
  metadata?: PlanDisplayMetadata;
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
  app_key: string;
  platform: string;
  artifact_url: string;
  release_version: string;
  release_notes?: string;
  requires_entitlement: boolean;
  metadata?: Record<string, unknown>;
}

export interface DownloadStorefront {
  store: string;
  label: string;
  url: string;
  badge?: string;
}

export interface DownloadApp {
  bundle_key: string;
  app_key: string;
  name: string;
  tagline?: string;
  description?: string;
  install_overview?: string;
  install_steps?: string[];
  storefronts?: DownloadStorefront[];
  metadata?: Record<string, unknown>;
  display_order?: number;
  platforms: DownloadAsset[];
}

export type SectionType =
  | 'hero'
  | 'features'
  | 'pricing'
  | 'cta'
  | 'testimonials'
  | 'faq'
  | 'footer'
  | 'video'
  | 'downloads';

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

export type HeaderBrandingMode = 'none' | 'logo' | 'name' | 'logo_and_name';
export type HeaderNavLinkType = 'section' | 'downloads' | 'custom' | 'menu';
export type HeaderCTAMode = 'inherit_hero' | 'downloads' | 'custom' | 'hidden';

export interface LandingHeaderConfig {
  branding: HeaderBrandingConfig;
  nav: HeaderNavConfig;
  ctas: HeaderCTAGroup;
  behavior: HeaderBehaviorConfig;
}

export interface HeaderBrandingConfig {
  mode: HeaderBrandingMode;
  label?: string;
  subtitle?: string;
  mobile_preference?: 'auto' | 'logo' | 'name' | 'stacked';
  logo_url?: string | null;
  logo_icon_url?: string | null;
}

export interface HeaderNavConfig {
  links: LandingHeaderNavLink[];
}

export interface LandingHeaderNavLink {
  id: string;
  type: HeaderNavLinkType;
  label: string;
  section_type?: string;
  section_id?: number;
  anchor?: string;
  href?: string;
  visible_on?: HeaderVisibilityConfig;
  children?: LandingHeaderNavLink[];
}

export interface HeaderVisibilityConfig {
  desktop?: boolean;
  mobile?: boolean;
}

export interface HeaderCTAGroup {
  primary: HeaderCTAConfig;
  secondary: HeaderCTAConfig;
}

export interface HeaderCTAConfig {
  mode: HeaderCTAMode;
  label?: string;
  href?: string;
  variant?: 'solid' | 'ghost';
}

export interface HeaderBehaviorConfig {
  sticky: boolean;
  hide_on_scroll: boolean;
}

// Public branding info included in landing config
export interface LandingBranding {
  site_name: string;
  tagline?: string | null;
  logo_url?: string | null;
  logo_icon_url?: string | null;
  favicon_url?: string | null;
  theme_primary_color?: string | null;
  theme_background_color?: string | null;
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
  downloads: DownloadApp[];
  header: LandingHeaderConfig;
  branding?: LandingBranding;
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

export interface BundleCatalogEntry {
  bundle: BundleProduct;
  prices: PlanOption[];
}

// Site-wide branding configuration
export interface SiteBranding {
  id: number;
  site_name: string;
  tagline?: string | null;
  logo_url?: string | null;
  logo_icon_url?: string | null;
  favicon_url?: string | null;
  apple_touch_icon_url?: string | null;
  default_title?: string | null;
  default_description?: string | null;
  default_og_image_url?: string | null;
  theme_primary_color?: string | null;
  theme_background_color?: string | null;
  canonical_base_url?: string | null;
  google_site_verification?: string | null;
  robots_txt?: string | null;
  created_at?: string;
  updated_at?: string;
}

export interface SiteBrandingUpdate {
  site_name?: string;
  tagline?: string;
  logo_url?: string;
  logo_icon_url?: string;
  favicon_url?: string;
  apple_touch_icon_url?: string;
  default_title?: string;
  default_description?: string;
  default_og_image_url?: string;
  theme_primary_color?: string;
  theme_background_color?: string;
  canonical_base_url?: string;
  google_site_verification?: string;
  robots_txt?: string;
}

export interface PublicBranding {
  site_name: string;
  tagline?: string | null;
  logo_url?: string | null;
  logo_icon_url?: string | null;
  favicon_url?: string | null;
  theme_primary_color?: string | null;
  theme_background_color?: string | null;
}

// Uploaded asset
export interface Asset {
  id: number;
  filename: string;
  original_filename: string;
  mime_type: string;
  size_bytes: number;
  storage_path: string;
  thumbnail_path?: string | null;
  alt_text?: string | null;
  category: string;
  uploaded_by?: string | null;
  created_at: string;
  url: string;
  derivatives?: Record<string, string>;
}

export type AssetCategory = 'logo' | 'favicon' | 'og_image' | 'general';

// Per-variant SEO configuration
export interface VariantSEOConfig {
  title?: string;
  description?: string;
  og_title?: string;
  og_description?: string;
  og_image_url?: string;
  twitter_card?: 'summary' | 'summary_large_image';
  canonical_path?: string;
  noindex?: boolean;
  structured_data?: Record<string, unknown>;
}
