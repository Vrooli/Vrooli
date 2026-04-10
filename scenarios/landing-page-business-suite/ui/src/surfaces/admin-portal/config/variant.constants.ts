import type { VariantResolution } from '../../../app/providers/LandingVariantProvider';

/**
 * Human-readable labels for variant resolution strategies.
 * Centralized here to avoid duplication across RuntimeSignalStrip, AdminHome, and AdminAnalytics.
 */
export const RESOLUTION_LABELS: Record<VariantResolution, string> = {
  url_param: 'URL parameter',
  local_storage: 'Stored visitor assignment',
  api_select: 'Weighted API selection',
  fallback: 'Fallback payload',
  unknown: 'Unknown strategy',
};

/**
 * Get human-readable label for a variant resolution strategy.
 * Falls back to 'Unknown strategy' for unrecognized values.
 */
export function getResolutionLabel(resolution: VariantResolution): string {
  return RESOLUTION_LABELS[resolution] ?? RESOLUTION_LABELS.unknown;
}
