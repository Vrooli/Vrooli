import { useEffect, useRef } from 'react';
import { useVariant } from '../contexts/VariantContext';

// Generate session ID (persisted in sessionStorage)
function getSessionID(): string {
  let sessionID = sessionStorage.getItem('metrics_session_id');
  if (!sessionID) {
    sessionID = `session_${Date.now()}_${Math.random().toString(36).substring(7)}`;
    sessionStorage.setItem('metrics_session_id', sessionID);
  }
  return sessionID;
}

// Generate visitor ID (persisted in localStorage for cross-session tracking)
function getVisitorID(): string {
  let visitorID = localStorage.getItem('metrics_visitor_id');
  if (!visitorID) {
    visitorID = `visitor_${Date.now()}_${Math.random().toString(36).substring(7)}`;
    localStorage.setItem('metrics_visitor_id', visitorID);
  }
  return visitorID;
}

interface MetricEvent {
  event_type: 'page_view' | 'scroll_depth' | 'click' | 'form_submit' | 'conversion';
  variant_id: number;
  event_data?: Record<string, unknown>;
  session_id: string;
  visitor_id?: string;
  event_id?: string;
}

/**
 * Hook to track analytics events with variant tagging
 * Implements OT-P0-019 (METRIC-TAG): All events include variant_id
 * Implements OT-P0-021 (METRIC-EVENTS): Emits page_view, scroll_depth, click, form_submit, conversion
 */
export function useMetrics() {
  const { variant } = useVariant();
  const sessionID = useRef(getSessionID());
  const visitorID = useRef(getVisitorID());
  const scrollDepthTracked = useRef<Set<number>>(new Set());

  // Track event to API
  const trackEvent = async (
    eventType: MetricEvent['event_type'],
    eventData?: Record<string, unknown>
  ) => {
    if (!variant) {
      console.warn('[useMetrics] No variant selected, skipping event tracking');
      return;
    }

    const event: MetricEvent = {
      event_type: eventType,
      variant_id: variant.id ?? 0,
      session_id: sessionID.current,
      visitor_id: visitorID.current,
      event_data: eventData,
    };

    try {
      const response = await fetch('/api/v1/metrics/track', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(event),
      });

      if (!response.ok) {
        console.error('[useMetrics] Failed to track event:', response.statusText);
      }
    } catch (error) {
      console.error('[useMetrics] Error tracking event:', error);
    }
  };

  // Track page view on mount
  useEffect(() => {
    if (variant) {
      trackEvent('page_view', {
        page: window.location.pathname,
        referrer: document.referrer,
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [variant?.id]);

  // Track scroll depth (bands: 25%, 50%, 75%, 100%)
  useEffect(() => {
    if (!variant) return;

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
  }, [variant?.id]);

  // Track CTA clicks
  const trackCTAClick = (elementId: string, elementData?: Record<string, unknown>) => {
    trackEvent('click', {
      element_id: elementId,
      element_type: 'cta',
      ...elementData,
    });
  };

  // Track form submission
  const trackFormSubmit = (formId: string, formData?: Record<string, unknown>) => {
    trackEvent('form_submit', {
      form_id: formId,
      ...formData,
    });
  };

  // Track conversion (e.g., Stripe checkout success)
  const trackConversion = (conversionData?: Record<string, unknown>) => {
    trackEvent('conversion', conversionData);
  };

  return {
    trackCTAClick,
    trackFormSubmit,
    trackConversion,
    trackEvent, // Generic event tracker
  };
}
