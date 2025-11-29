import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

// Type definitions
export interface Variant {
  id: number;
  slug: string;
  name: string;
  description?: string;
  weight: number;
  status: 'active' | 'archived' | 'deleted';
  created_at: string;
  updated_at: string;
  archived_at?: string;
}

/**
 * Section types available for landing pages.
 * This type is auto-derived from the section registry.
 *
 * To add a new section, update components/sections/registry.tsx
 * No need to manually update this file!
 */
import type { SectionType } from '../components/sections/registry';
export type { SectionType };

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

export interface MetricEvent {
  event_type: 'page_view' | 'scroll_depth' | 'click' | 'form_submit' | 'conversion';
  variant_id: number;
  session_id: string;
  visitor_id?: string;
  event_data?: Record<string, unknown>;
}

export interface AnalyticsSummary {
  total_visitors: number;
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
  conversion_rate: number;
  avg_scroll_depth?: number;
  trend?: 'up' | 'down' | 'stable';
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

// Variants
export async function listVariants() {
  return apiCall<{ variants: Variant[] }>('/variants');
}

export async function getVariant(slug: string) {
  return apiCall<Variant>(`/variants/${slug}`);
}

export async function createVariant(data: { name: string; slug: string; description?: string; weight?: number }) {
  return apiCall<Variant>('/variants', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateVariant(slug: string, data: Partial<Variant>) {
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

// Content Sections
export async function getSections(variantId: number) {
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
