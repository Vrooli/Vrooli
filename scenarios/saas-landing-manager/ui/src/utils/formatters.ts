/**
 * Formatting utilities for the SaaS Landing Manager UI
 */

export interface ScenarioMetadata {
  characteristics?: string[];
  analysis_version?: string;
  [key: string]: unknown;
}

export function formatCurrency(value: string): string {
  if (!value) {
    return 'N/A';
  }

  const numeric = Number.parseFloat(value.replace(/[^0-9.]/g, ''));
  if (Number.isNaN(numeric)) {
    return value;
  }

  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 0,
  }).format(numeric);
}

export function formatDate(timestamp: string): string {
  if (!timestamp) {
    return 'Unknown';
  }

  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) {
    return timestamp;
  }

  return date.toLocaleString();
}

export function formatRelativeTime(timestamp: string): string {
  if (!timestamp) {
    return 'N/A';
  }

  const target = new Date(timestamp).getTime();
  if (Number.isNaN(target)) {
    return timestamp;
  }

  const diffMs = Date.now() - target;
  const diffMinutes = Math.round(diffMs / 60000);
  if (diffMinutes < 1) {
    return 'just now';
  }
  if (diffMinutes < 60) {
    return `${diffMinutes}m ago`;
  }
  const diffHours = Math.round(diffMinutes / 60);
  if (diffHours < 24) {
    return `${diffHours}h ago`;
  }
  const diffDays = Math.round(diffHours / 24);
  if (diffDays < 7) {
    return `${diffDays}d ago`;
  }
  const diffWeeks = Math.round(diffDays / 7);
  if (diffWeeks < 5) {
    return `${diffWeeks}w ago`;
  }
  const diffMonths = Math.round(diffDays / 30);
  if (diffMonths < 12) {
    return `${diffMonths}mo ago`;
  }
  const diffYears = Math.round(diffDays / 365);
  return `${diffYears}y ago`;
}

export function extractCharacteristics(metadata?: ScenarioMetadata): string[] {
  if (!metadata?.characteristics) {
    return [];
  }
  return metadata.characteristics.filter((entry): entry is string => typeof entry === 'string');
}

export const SaasTypeLabels: Record<string, string> = {
  b2b_tool: 'B2B Tool',
  b2c_app: 'B2C App',
  api_service: 'API Service',
  marketplace: 'Marketplace',
};

export const ConfidenceDescriptions: Record<string, string> = {
  High: 'Strong signal and recent scan',
  Medium: 'Moderate confidence, consider review',
  Low: 'Limited data, investigate manually',
};

export function confidenceBand(score: number): string {
  if (score >= 1.5) {
    return 'High';
  }
  if (score >= 1.0) {
    return 'Medium';
  }
  return 'Low';
}

export interface SaaSScenario {
  id: string;
  scenario_name: string;
  display_name: string;
  description: string;
  saas_type: string;
  industry: string;
  revenue_potential: string;
  has_landing_page: boolean;
  landing_page_url: string;
  last_scan: string;
  confidence_score: number;
  metadata: ScenarioMetadata;
}

export function scenarioLabel(scenario: SaaSScenario): string {
  return scenario.display_name || scenario.scenario_name || scenario.id;
}
