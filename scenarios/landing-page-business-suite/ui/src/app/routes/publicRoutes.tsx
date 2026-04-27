import { ReactNode } from 'react';
import { Route } from 'react-router-dom';
import { ErrorBoundary } from '../../shared/ui/ErrorBoundary';
import { useLandingVariant } from '../providers/useLandingVariant';
import { ComingSoonPage } from '../../surfaces/public-landing/routes/ComingSoonPage';
import { PublicLanding } from '../../surfaces/public-landing/routes/PublicLanding';
import { CheckoutPage } from '../../surfaces/public-landing/routes/CheckoutPage';
import { FeedbackPage } from '../../surfaces/public-landing/routes/FeedbackPage';

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
      <PublicRouteGuard>{children}</PublicRouteGuard>
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
