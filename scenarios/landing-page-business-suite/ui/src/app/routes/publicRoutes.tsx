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

function PublicRouteGuard({ children }: { children: ReactNode }) {
  const { config, loading } = useLandingVariant();

  if (loading) {
    return (
      <div className="min-h-screen bg-bg-base flex items-center justify-center">
        <div className="animate-pulse text-slate-400">Loading...</div>
      </div>
    );
  }

  if (config?.branding?.coming_soon_enabled) {
    return <ComingSoonPage branding={config.branding} />;
  }

  return <>{children}</>;
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
