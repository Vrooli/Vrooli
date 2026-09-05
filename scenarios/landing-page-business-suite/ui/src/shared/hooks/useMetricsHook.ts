import { useCallback, useContext, useEffect, useRef } from 'react';
import { useLandingVariant } from '../../app/providers/useLandingVariant';
import { MetricsModeContext } from './MetricsModeContext';
import { trackMetric, type MetricEvent as APIMetricEvent } from '../api';

const SESSION_STORAGE_KEY = 'metrics_session_id';
const VISITOR_STORAGE_KEY = 'metrics_visitor_id';
const CAMPAIGN_STORAGE_KEY = 'metrics_campaign';

let fallbackSessionId: string | null = null;
let fallbackVisitorId: string | null = null;
let sessionWarningLogged = false;
let visitorWarningLogged = false;
const activePageViews = new Map<string, number>();
const trackedScrollDepth = new Map<string, Set<number>>();
const activeScrollListeners = new Map<string, { count: number; handler: () => void }>();

function generateId(prefix: string) {
	return `${prefix}_${Date.now().toString()}_${Math.random().toString(36).substring(2, 9)}`;
}

function logStorageWarning(kind: 'session' | 'local', error: unknown) {
  if (kind === 'session') {
    if (sessionWarningLogged) return;
    sessionWarningLogged = true;
  } else {
    if (visitorWarningLogged) return;
    visitorWarningLogged = true;
  }
  console.warn(`[useMetrics] Access to ${kind}Storage unavailable:`, error);
}

function getStorage(kind: 'session' | 'local') {
  if (typeof window === 'undefined') {
    return undefined;
  }

  try {
    return kind === 'session' ? window.sessionStorage : window.localStorage;
  } catch (error) {
    logStorageWarning(kind, error);
    return undefined;
  }
}

// Generate session ID (persisted in sessionStorage)
function getSessionID(): string {
  const fallback = () => {
    if (!fallbackSessionId) {
      fallbackSessionId = generateId('session');
    }
    return fallbackSessionId;
  };

  const storage = getStorage('session');
  if (!storage) {
    return fallback();
  }

  try {
    let sessionID = storage.getItem(SESSION_STORAGE_KEY);
    if (!sessionID) {
      sessionID = generateId('session');
      storage.setItem(SESSION_STORAGE_KEY, sessionID);
    }
    fallbackSessionId = sessionID;
    return sessionID;
  } catch (error) {
    logStorageWarning('session', error);
    return fallback();
  }
}

// Generate visitor ID (persisted in localStorage for cross-session tracking)
function getVisitorID(): string {
  const fallback = () => {
    if (!fallbackVisitorId) {
      fallbackVisitorId = generateId('visitor');
    }
    return fallbackVisitorId;
  };

  const storage = getStorage('local');
  if (!storage) {
    return fallback();
  }

  try {
    let visitorID = storage.getItem(VISITOR_STORAGE_KEY);
    if (!visitorID) {
      visitorID = generateId('visitor');
      storage.setItem(VISITOR_STORAGE_KEY, visitorID);
    }
    fallbackVisitorId = visitorID;
    return visitorID;
  } catch (error) {
    logStorageWarning('local', error);
    return fallback();
  }
}

function getPageMetricKey(variantSlug: string) {
  const path = typeof window === 'undefined' ? '/' : `${window.location.pathname}${window.location.search}`;
  return `${variantSlug}:${path}`;
}

type CampaignAttribution = { utm_source: string; utm_medium: string; utm_campaign: string; landing_path: string; referrer: string };

function getCampaignAttribution(): CampaignAttribution {
  const empty = { utm_source: '', utm_medium: '', utm_campaign: '', landing_path: typeof window === 'undefined' ? '/' : window.location.pathname, referrer: '' };
  const storage = getStorage('session');
  if (!storage) return empty;
  try {
    const existing = storage.getItem(CAMPAIGN_STORAGE_KEY);
    if (existing) return JSON.parse(existing) as CampaignAttribution;
    const params = new URLSearchParams(window.location.search);
    const value = { utm_source: params.get('utm_source') ?? '', utm_medium: params.get('utm_medium') ?? '', utm_campaign: params.get('utm_campaign') ?? '', landing_path: window.location.pathname, referrer: document.referrer };
    storage.setItem(CAMPAIGN_STORAGE_KEY, JSON.stringify(value));
    return value;
  } catch { return empty; }
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
  const { variant } = useLandingVariant();
  const metricsMode = useContext(MetricsModeContext);
  const previewMode = metricsMode === 'preview';
  const sessionID = useRef(getSessionID());
  const visitorID = useRef(getVisitorID());

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

    const attribution = getCampaignAttribution();
    const event: MetricEventPayload = {
      event_type: eventType,
      variant_slug: variant.slug,
      session_id: sessionID.current,
      visitor_id: visitorID.current,
      event_data: eventData,
      ...attribution,
      referrer: eventType === 'page_view' ? attribution.referrer : '',
    };

    try {
      await trackMetric(event);
    } catch (error) {
      console.error('[useMetrics] Error tracking event:', error);
    }
  }, [previewMode, variant]);

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

  // Track form submission
  const trackFormSubmit = useCallback((formId: string, formData?: Record<string, unknown>) => {
    if (previewMode) return;
    void trackEvent('form_submit', {
      form_id: formId,
      ...formData,
    });
  }, [previewMode, trackEvent]);

  // Track conversion (e.g., Stripe checkout success)
  const trackConversion = useCallback((conversionData?: Record<string, unknown>) => {
    if (previewMode) return;
    void trackEvent('conversion', conversionData);
  }, [previewMode, trackEvent]);

  const trackDownload = useCallback((downloadData?: Record<string, unknown>) => {
    if (previewMode) return;
    void trackEvent('download', downloadData);
  }, [previewMode, trackEvent]);

  return {
    trackCTAClick,
    trackFormSubmit,
    trackConversion,
    trackDownload,
    trackEvent, // Generic event tracker
  };
}
