/* eslint-disable react-refresh/only-export-components */
import React, { ReactNode, lazy } from 'react';
import { Route } from 'react-router-dom';
import { ErrorBoundary } from '../../shared/ui/ErrorBoundary';
import { PublicPresentationState } from '../../surfaces/public-landing/presentation/PublicPresentationState';
import { onProfilerRender } from '../../lib/profiler';

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
const DownloadPage = lazy(() => import('../../surfaces/public-landing/presentation/DownloadPage').then(module => ({ default: module.DownloadPage })));

export function PublicRouteGuard({ children }: { children: ReactNode }) {
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
    <Route path="/apps/:slug" element={<PublicRoute name="AppDetail"><PublicLanding /></PublicRoute>} />
    <Route path="/apps/:slug/download" element={<PublicRoute name="AppDownload"><DownloadPage /></PublicRoute>} />
    <Route path="*" element={<PublicPresentationState state="not-found" />} />
    <Route path="/checkout" element={<PublicRoute name="Checkout"><CheckoutPage /></PublicRoute>} />
    <Route path="/feedback" element={<PublicRoute name="Feedback"><FeedbackPage /></PublicRoute>} />
  </>
);
