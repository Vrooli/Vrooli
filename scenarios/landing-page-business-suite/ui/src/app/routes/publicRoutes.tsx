/* eslint-disable react-refresh/only-export-components */
import React, { ReactNode, lazy } from 'react';
import { Navigate, Route } from 'react-router-dom';
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
const ContactPage = lazy(() => import('../../surfaces/public-landing/site/ContactPage').then(module => ({ default: module.ContactPage })));
const ThankYouPage = lazy(() => import('../../surfaces/public-landing/site/SitePages').then(module => ({ default: module.ThankYouPage })));
const LegalPage = lazy(() => import('../../surfaces/public-landing/site/SitePages').then(module => ({ default: module.LegalPage })));
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
    <Route path="/checkout" element={<PublicRoute name="Checkout"><CheckoutPage /></PublicRoute>} />
    <Route path="/contact" element={<PublicRoute name="Contact"><ContactPage /></PublicRoute>} />
    <Route path="/feedback" element={<Navigate to="/contact" replace />} />
    <Route path="/thank-you" element={<PublicRoute name="ThankYou"><ThankYouPage /></PublicRoute>} />
    <Route path="/privacy" element={<PublicRoute name="PrivacyPolicy"><LegalPage kind="privacy" /></PublicRoute>} />
    <Route path="/terms" element={<PublicRoute name="Terms"><LegalPage kind="terms" /></PublicRoute>} />
    <Route path="*" element={<PublicRoute name="NotFound"><PublicPresentationState state="not-found" /></PublicRoute>} />
  </>
);
