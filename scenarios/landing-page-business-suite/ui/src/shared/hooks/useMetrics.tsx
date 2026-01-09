import { createContext, useCallback, useContext, useEffect, useRef, type ReactNode } from 'react';
import { useLandingVariant } from '../../app/providers/LandingVariantProvider';
import { trackMetric, type MetricEvent as APIMetricEvent } from '../api';

const SESSION_STORAGE_KEY = 'metrics_session_id';
const VISITOR_STORAGE_KEY = 'metrics_visitor_id';

let fallbackSessionId: string | null = null;
let fallbackVisitorId: string | null = null;
let sessionWarningLogged = false;
let visitorWarningLogged = false;

type MetricsMode = 'live' | 'preview';

const MetricsModeContext = createContext<MetricsMode>('live');

export function MetricsModeProvider({
  mode = 'live',
  children,
}: {
  mode?: MetricsMode;
  children: ReactNode;
}) {
  return <MetricsModeContext.Provider value={mode}>{children}</MetricsModeContext.Provider>;
}

function generateId(prefix: string) {
  return `${prefix}_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`;
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
  const scrollDepthTracked = useRef<Set<number>>(new Set());

  // Track event to API
  const trackEvent = useCallback(async (
    eventType: APIMetricEvent['event_type'],
    eventData?: Record<string, unknown>
  ) => {
    if (previewMode) {
      return;
    }
    if (!variant) {
      console.warn('[useMetrics] No variant selected, skipping event tracking');
      return;
    }

    const event: MetricEventPayload = {
      event_type: eventType,
      variant_id: variant.id ?? 0,
      session_id: sessionID.current,
      visitor_id: visitorID.current,
      event_data: eventData,
    };

    try {
      await trackMetric(event);
    } catch (error) {
      console.error('[useMetrics] Error tracking event:', error);
    }
  }, [previewMode, variant]);

  // Track page view on mount
  useEffect(() => {
    if (previewMode || !variant) {
      return;
    }
    if (variant) {
      trackEvent('page_view', {
        page: window.location.pathname,
        referrer: document.referrer,
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [variant?.id, previewMode]);

  // Track scroll depth (bands: 25%, 50%, 75%, 100%)
  useEffect(() => {
    if (previewMode || !variant) return;

    const handleScroll = () => {
      const scrollPercentage = (window.scrollY + window.innerHeight) / document.documentElement.scrollHeight * 100;
      const bands = [25, 50, 75, 100];

      for (const band of bands) {
        if (scrollPercentage >= band && !scrollDepthTracked.current.has(band)) {
          scrollDepthTracked.current.add(band);
          trackEvent('scroll_depth', { depth: band });
        }
      }
    };

    window.addEventListener('scroll', handleScroll, { passive: true });
    return () => window.removeEventListener('scroll', handleScroll);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [variant?.id, previewMode]);

  // Track CTA clicks
  const trackCTAClick = (elementId: string, elementData?: Record<string, unknown>) => {
    if (previewMode) return;
    trackEvent('click', {
      element_id: elementId,
      element_type: 'cta',
      ...elementData,
    });
  };

  // Track form submission
  const trackFormSubmit = (formId: string, formData?: Record<string, unknown>) => {
    if (previewMode) return;
    trackEvent('form_submit', {
      form_id: formId,
      ...formData,
    });
  };

  // Track conversion (e.g., Stripe checkout success)
  const trackConversion = (conversionData?: Record<string, unknown>) => {
    if (previewMode) return;
    trackEvent('conversion', conversionData);
  };

  const trackDownload = (downloadData?: Record<string, unknown>) => {
    if (previewMode) return;
    trackEvent('download', downloadData);
  };

  return {
    trackCTAClick,
    trackFormSubmit,
    trackConversion,
    trackDownload,
    trackEvent, // Generic event tracker
  };
}
