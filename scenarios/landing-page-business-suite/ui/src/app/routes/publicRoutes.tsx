/* eslint-disable react-refresh/only-export-components */
import React, { ReactNode, lazy } from 'react';
import { Route } from 'react-router-dom';
import { ErrorBoundary } from '../../shared/ui/ErrorBoundary';
import { useLandingVariant } from '../providers/useLandingVariant';
import { onProfilerRender } from '../../lib/profiler';

const ComingSoonPage = lazy(() =>
  import('../../surfaces/public-landing/routes/ComingSoonPage').then((module) => ({
    default: module.ComingSoonPage,
  }))
);
const PublicLanding = lazy(() =>
  import('../../surfaces/public-landing/routes/PublicLanding').then((module) => ({
    default: module.PublicLanding,
  }))
);
const CheckoutPage = lazy(() =>
  import('../../surfaces/public-landing/routes/CheckoutPage').then((module) => ({
    default: module.CheckoutPage,
  }))
);
const FeedbackPage = lazy(() =>
  import('../../surfaces/public-landing/routes/FeedbackPage').then((module) => ({
    default: module.FeedbackPage,
  }))
);

export function PublicRouteGuard({ children }: { children: ReactNode }) {
  const { config, loading } = useLandingVariant();

  if (loading) {
    return (
      <main
        aria-busy="true"
        aria-label="Preparing Silent Founder OS"
        data-testid="landing-experience-surface"
        data-experience-surface="public-landing"
        data-experience-state="loading"
        className="min-h-full overflow-hidden bg-bg-base text-white"
      >
        <div className="border-b border-white/10 bg-surface-deep/80 px-6 py-5">
          <div className="mx-auto flex max-w-6xl items-center justify-between">
            <span className="text-sm font-semibold tracking-wide">Silent Founder OS</span>
            <span className="h-9 w-24 animate-pulse rounded-full bg-white/10" aria-hidden="true" />
          </div>
        </div>
        <div className="mx-auto grid max-w-6xl gap-12 px-6 py-20 lg:grid-cols-[1.05fr,0.95fr] lg:items-center">
          <div className="space-y-6">
            <span className="inline-block h-7 w-48 animate-pulse rounded-full bg-accent/20" aria-hidden="true" />
            <div className="space-y-3" aria-hidden="true">
              <div className="h-12 max-w-xl animate-pulse rounded-lg bg-white/10" />
              <div className="h-12 w-4/5 animate-pulse rounded-lg bg-white/10" />
              <div className="h-5 max-w-lg animate-pulse rounded bg-white/5" />
            </div>
            <p className="text-sm text-slate-300">Preparing your workspace…</p>
          </div>
          <div className="h-72 animate-pulse rounded-3xl border border-white/10 bg-surface-deep" aria-hidden="true" />
        </div>
      </main>
    );
  }

  if (config?.branding?.coming_soon_enabled) {
    return (
      <div data-testid="landing-experience-surface" data-experience-surface="public-landing" data-experience-state="ready">
        <ComingSoonPage branding={config.branding} />
      </div>
    );
  }

  return (
    <div
      data-testid="landing-experience-surface"
      data-experience-surface="public-landing"
      data-experience-state={config ? 'ready' : 'error'}
    >
      {children}
    </div>
  );
}

function PublicRoute({ name, children }: { name: string; children: ReactNode }) {
  return (
    <ErrorBoundary level="route" name={name}>
      <React.Profiler id={name} onRender={onProfilerRender}>
        <PublicRouteGuard>{children}</PublicRouteGuard>
      </React.Profiler>
    </ErrorBoundary>
  );
}

export const publicRoutes = (
  <>
    <Route path="/" element={<PublicRoute name="PublicLanding"><PublicLanding /></PublicRoute>} />
    <Route path="/health" element={<PublicRoute name="Health"><PublicLanding /></PublicRoute>} />
    <Route path="/checkout" element={<PublicRoute name="Checkout"><CheckoutPage /></PublicRoute>} />
    <Route path="/feedback" element={<PublicRoute name="Feedback"><FeedbackPage /></PublicRoute>} />
  </>
);
