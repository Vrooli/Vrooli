import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

// Type definitions
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
  bundle_key: string;
  platform: string;
  artifact_url: string;
  release_version: string;
  release_notes?: string;
  requires_entitlement: boolean;
  metadata?: Record<string, unknown>;
}

/**
 * Section types available for landing pages.
 * When adding a new section, update this union and:
 * 1. Create component in components/sections/{Name}Section.tsx
 * 2. Add switch case in pages/PublicHome.tsx
 * 3. Add CHECK constraint in initialization/postgres/schema.sql
 * 4. Create schema in .vrooli/schemas/sections/{id}.schema.json
 */
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

// Helper for API calls
async function apiCall<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const url = buildApiUrl(endpoint, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
    credentials: 'include',
    ...options,
  });

  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(`API call failed (${res.status}): ${errorText}`);
  }

  return res.json() as Promise<T>;
}

// Health check
export async function fetchHealth() {
  return apiCall<{ status: string; service: string; timestamp: string }>('/health');
}

export async function getLandingConfig(variantSlug?: string) {
  const params = new URLSearchParams();
  if (variantSlug) {
    params.set('variant', variantSlug);
  }
  const query = params.toString() ? `?${params.toString()}` : '';
  return apiCall<LandingConfigResponse>(`/landing-config${query}`);
}

export async function getPlans() {
  return apiCall<PricingOverview>('/plans');
}

export async function getSubscriptionInfo() {
  return apiCall<SubscriptionInfo>('/me/subscription');
}

export async function getCreditInfo() {
  return apiCall<CreditInfo>('/me/credits');
}

export async function getEntitlements(userEmail?: string) {
  const params = new URLSearchParams();
  if (userEmail?.trim()) {
    params.set('user', userEmail.trim());
  }
  const query = params.toString() ? `?${params.toString()}` : '';
  return apiCall<EntitlementPayload>(`/entitlements${query}`);
}

export async function requestDownload(platform: string, user?: string) {
  const params = new URLSearchParams({ platform });
  if (user) {
    params.set('user', user);
  }
  return apiCall<DownloadAsset>(`/downloads?${params.toString()}`);
}

// Authentication
export async function adminLogin(email: string, password: string) {
  return apiCall<{ success: boolean; message: string }>('/admin/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
}

export async function adminLogout() {
  return apiCall<{ success: boolean }>('/admin/logout', {
    method: 'POST',
  });
}

export async function checkAdminSession() {
  return apiCall<{ authenticated: boolean; email?: string }>('/admin/session');
}

// Variants - Public (no auth required)
export async function getPublicVariant(slug: string) {
  return apiCall<Variant>(`/public/variants/${slug}`);
}

// Variants - Admin (requires auth)
export async function listVariants() {
  return apiCall<{ variants: Variant[] }>('/variants');
}

export async function getVariant(slug: string) {
  return apiCall<Variant>(`/variants/${slug}`);
}

export interface VariantCreatePayload {
  name: string;
  slug: string;
  description?: string;
  weight?: number;
  axes: VariantAxes;
}

export interface VariantUpdatePayload {
  name?: string;
  description?: string;
  weight?: number;
  axes?: VariantAxes;
}

export async function createVariant(data: VariantCreatePayload) {
  return apiCall<Variant>('/variants', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateVariant(slug: string, data: VariantUpdatePayload) {
  return apiCall<Variant>(`/variants/${slug}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
  });
}

export async function archiveVariant(slug: string) {
  return apiCall<{ success: boolean }>(`/variants/${slug}/archive`, {
    method: 'POST',
  });
}

export async function deleteVariant(slug: string) {
  return apiCall<{ success: boolean }>(`/variants/${slug}`, {
    method: 'DELETE',
  });
}

export async function selectVariant(variantSlug?: string) {
  const query = variantSlug ? `?variant_slug=${variantSlug}` : '';
  return apiCall<Variant>(`/variants/select${query}`);
}

export async function getVariantSpace() {
  return apiCall<VariantSpace>('/variant-space');
}

// Content Sections - Public (no auth required for landing page display)
export async function getSections(variantId: number) {
  return apiCall<{ sections: ContentSection[] }>(`/public/variants/${variantId}/sections`);
}

// Content Sections - Admin (requires auth for editing)
export async function getAdminSections(variantId: number) {
  return apiCall<{ sections: ContentSection[] }>(`/variants/${variantId}/sections`);
}

export async function getSection(sectionId: number) {
  return apiCall<ContentSection>(`/sections/${sectionId}`);
}

export async function updateSection(sectionId: number, content: Record<string, unknown>) {
  return apiCall<{ success: boolean; message: string }>(`/sections/${sectionId}`, {
    method: 'PATCH',
    body: JSON.stringify({ content }),
  });
}

export async function createSection(section: Omit<ContentSection, 'id' | 'created_at' | 'updated_at'>) {
  return apiCall<ContentSection>('/sections', {
    method: 'POST',
    body: JSON.stringify(section),
  });
}

export async function deleteSection(sectionId: number) {
  return apiCall<{ success: boolean }>(`/sections/${sectionId}`, {
    method: 'DELETE',
  });
}

// Metrics
export async function trackMetric(event: MetricEvent) {
  return apiCall<{ success: boolean }>('/metrics/track', {
    method: 'POST',
    body: JSON.stringify(event),
  });
}

export async function getMetricsSummary(startDate?: string, endDate?: string) {
  const params = new URLSearchParams();
  if (startDate) params.set('start_date', startDate);
  if (endDate) params.set('end_date', endDate);
  const query = params.toString() ? `?${params.toString()}` : '';
  return apiCall<AnalyticsSummary>(`/metrics/summary${query}`);
}

export async function getVariantMetrics(variantSlug?: string, startDate?: string, endDate?: string) {
  const params = new URLSearchParams();
  if (variantSlug) params.set('variant', variantSlug);
  if (startDate) params.set('start_date', startDate);
  if (endDate) params.set('end_date', endDate);
  const query = params.toString() ? `?${params.toString()}` : '';
  return apiCall<{ start_date: string; end_date: string; stats: VariantStats[] }>(`/metrics/variants${query}`);
}

// Agent Customization
export async function triggerAgentCustomization(scenarioId: string, brief: string, assets: string[], preview: boolean = true) {
  return apiCall<{ job_id: string; status: string; agent_id: string }>('/customize', {
    method: 'POST',
    body: JSON.stringify({ scenario_id: scenarioId, brief, assets, preview }),
  });
}
