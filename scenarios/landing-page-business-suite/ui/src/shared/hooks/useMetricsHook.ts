import { useCallback, useContext, useEffect, useRef } from 'react';
import { useLandingVariant } from '../../app/providers/useLandingVariant';
import { MetricsModeContext } from './MetricsModeContext';
import { trackMetric, type MetricEvent as APIMetricEvent } from '../api';
import { getAttributionContext } from '../lib/attribution';

const activePageViews = new Map<string, number>();
const trackedScrollDepth = new Map<string, Set<number>>();
const activeScrollListeners = new Map<string, { count: number; handler: () => void }>();

function getPageMetricKey(variantSlug: string) {
  const path = typeof window === 'undefined' ? '/' : `${window.location.pathname}${window.location.search}`;
  return `${variantSlug}:${path}`;
}

type MetricEventPayload = APIMetricEvent & {
  event_id?: string;
};

/**
 * Hook to track analytics events with variant tagging
 * Implements OT-P0-019 (METRIC-TAG): All events include variant_id
 * Implements OT-P0-021 (METRIC-EVENTS): Emits page_view, scroll_depth, click, form_submit, conversion
 */
export function useMetrics() {
  const { variant, visitorId: assignedVisitor } = useLandingVariant();
  const metricsMode = useContext(MetricsModeContext);
  const previewMode = metricsMode === 'preview';
  const attributionRef = useRef<ReturnType<typeof getAttributionContext>>();
  if (!previewMode && variant?.slug) attributionRef.current = getAttributionContext(variant.slug, assignedVisitor);

  // Track event to API
  const trackEvent = useCallback(async (
    eventType: APIMetricEvent['event_type'],
    eventData?: Record<string, unknown>
  ) => {
    if (previewMode) {
      return;
    }
    if (!variant?.slug) {
      console.warn('[useMetrics] No variant selected, skipping event tracking');
      return;
    }

    const attribution = attributionRef.current ?? getAttributionContext(variant.slug, assignedVisitor);
    const event: MetricEventPayload = {
      event_type: eventType,
      variant_slug: variant.slug,
      session_id: attribution.session_id,
      visitor_id: attribution.visitor_id,
      event_data: eventData,
      ...attribution,
      referrer: eventType === 'page_view' ? attribution.referrer : '',
    };

    try {
      await trackMetric(event);
    } catch (error) {
      console.error('[useMetrics] Error tracking event:', error);
    }
  }, [assignedVisitor, previewMode, variant]);

  // Track page view on mount
  useEffect(() => {
    if (previewMode || !variant?.slug) {
      return;
    }
    const pageKey = getPageMetricKey(variant.slug);
    const currentCount = activePageViews.get(pageKey) ?? 0;
    activePageViews.set(pageKey, currentCount + 1);
    if (currentCount === 0) {
      void trackEvent('page_view', {
        page: window.location.pathname,
        referrer: document.referrer,
      });
    }
    return () => {
      const nextCount = (activePageViews.get(pageKey) ?? 1) - 1;
      if (nextCount <= 0) {
        activePageViews.delete(pageKey);
      } else {
        activePageViews.set(pageKey, nextCount);
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [variant?.slug, previewMode]);

  // Track scroll depth (bands: 25%, 50%, 75%, 100%)
  useEffect(() => {
    if (previewMode || !variant?.slug) return;

    const pageKey = getPageMetricKey(variant.slug);
    const existing = activeScrollListeners.get(pageKey);
    if (existing) {
      existing.count += 1;
      return () => {
        const current = activeScrollListeners.get(pageKey);
        if (!current) return;
        current.count -= 1;
        if (current.count <= 0) {
          window.removeEventListener('scroll', current.handler);
          activeScrollListeners.delete(pageKey);
          trackedScrollDepth.delete(pageKey);
        }
      };
    }

    const handleScroll = () => {
      const scrollPercentage = (window.scrollY + window.innerHeight) / document.documentElement.scrollHeight * 100;
      const bands = [25, 50, 75, 100];
      let trackedBands = trackedScrollDepth.get(pageKey);
      if (!trackedBands) {
        trackedBands = new Set();
        trackedScrollDepth.set(pageKey, trackedBands);
      }

      for (const band of bands) {
        if (scrollPercentage >= band && !trackedBands.has(band)) {
          trackedBands.add(band);
          void trackEvent('scroll_depth', { depth: band });
        }
      }
    };

    window.addEventListener('scroll', handleScroll, { passive: true });
    activeScrollListeners.set(pageKey, { count: 1, handler: handleScroll });
    return () => {
      const current = activeScrollListeners.get(pageKey);
      if (!current) return;
      current.count -= 1;
      if (current.count <= 0) {
        window.removeEventListener('scroll', current.handler);
        activeScrollListeners.delete(pageKey);
        trackedScrollDepth.delete(pageKey);
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [variant?.slug, previewMode]);

  // Track CTA clicks
  const trackCTAClick = useCallback((elementId: string, elementData?: Record<string, unknown>) => {
    if (previewMode) return;
    void trackEvent('click', {
      element_id: elementId,
      element_type: 'cta',
      ...elementData,
    });
  }, [previewMode, trackEvent]);

  const trackDownload = useCallback((downloadData?: Record<string, unknown>) => {
    if (previewMode) return;
    void trackEvent('download', downloadData);
  }, [previewMode, trackEvent]);

  return {
    trackCTAClick,
    trackDownload,
    trackEvent, // Generic event tracker
  };
}
