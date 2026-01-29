// DOC: docs/concepts/ARCHITECTURE.md - UI architecture and component structure
// DOC: docs/QUICKSTART.md - Getting started guide
// DOC: docs/guides/ADMIN_GUIDE.md - Admin portal usage
import { ReactNode } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AdminAuthProvider } from './app/providers/AdminAuthProvider';
import { UserAuthProvider } from './app/providers/UserAuthProvider';
import { LandingVariantProvider, useLandingVariant } from './app/providers/LandingVariantProvider';
import { ErrorBoundary } from './shared/ui/ErrorBoundary';
import { ToastProvider } from './shared/ui/Toast';
import { ProtectedRoute } from './surfaces/admin-portal/components/ProtectedRoute';
import { AdminLogin } from './surfaces/admin-portal/routes/AdminLogin';
import { AdminHome } from './surfaces/admin-portal/routes/AdminHome';
import { AdminAnalytics } from './surfaces/admin-portal/routes/AdminAnalytics';
import { Customization } from './surfaces/admin-portal/routes/Customization';
import { VariantEditor } from './surfaces/admin-portal/routes/VariantEditor';
import { SectionEditor } from './surfaces/admin-portal/routes/SectionEditor';
import { AgentCustomization } from './surfaces/admin-portal/routes/AgentCustomization';
import { BillingSettings } from './surfaces/admin-portal/routes/BillingSettings';
import { DownloadSettings } from './surfaces/admin-portal/routes/DownloadSettings';
import { BrandingSettings } from './surfaces/admin-portal/routes/BrandingSettings';
import { DocsViewer } from './surfaces/admin-portal/routes/DocsViewer';
import { FeedbackManagement } from './surfaces/admin-portal/routes/FeedbackManagement';
import { WaitlistManagement } from './surfaces/admin-portal/routes/WaitlistManagement';
import { PublicLanding } from './surfaces/public-landing/routes/PublicLanding';
import { CheckoutPage } from './surfaces/public-landing/routes/CheckoutPage';
import { FeedbackPage } from './surfaces/public-landing/routes/FeedbackPage';
import { ProfileSettings } from './surfaces/admin-portal/routes/ProfileSettings';
import { ComingSoonPage } from './surfaces/public-landing/routes/ComingSoonPage';
import { APIKeysSettings } from './surfaces/admin-portal/routes/APIKeysSettings';
import { TierLimitsSettings } from './surfaces/admin-portal/routes/TierLimitsSettings';
import { AppLimitsSettings } from './surfaces/admin-portal/routes/AppLimitsSettings';
import { UsageDashboard } from './surfaces/admin-portal/routes/UsageDashboard';
import { AppsManagement } from './surfaces/admin-portal/routes/AppsManagement';
import { TiersManagement } from './surfaces/admin-portal/routes/TiersManagement';
import { CouponsManagement } from './surfaces/admin-portal/routes/CouponsManagement';
import { UserAccounts } from './surfaces/admin-portal/routes/UserAccounts';
import { LandingDashboard } from './surfaces/admin-portal/routes/LandingDashboard';
import { BillingDashboard } from './surfaces/admin-portal/routes/BillingDashboard';
import { UsersDashboard } from './surfaces/admin-portal/routes/UsersDashboard';
import { UserLogin, VerifyMagicLink } from './surfaces/user-auth';

/**
 * PublicRouteGuard checks if coming soon mode is enabled and shows the
 * ComingSoonPage instead of the normal public content when enabled.
 */
function PublicRouteGuard({ children }: { children: ReactNode }) {
  const { config, loading } = useLandingVariant();

  if (loading) {
    // Show a minimal loading state while config loads
    return (
      <div className="min-h-screen bg-bg-base flex items-center justify-center">
        <div className="animate-pulse text-slate-400">Loading...</div>
      </div>
    );
  }

  // Check if coming soon mode is enabled
  if (config?.branding?.coming_soon_enabled) {
    return <ComingSoonPage branding={config.branding} />;
  }

  return <>{children}</>;
}

interface RouteShellProps {
  name: string;
  children: ReactNode;
}

function PublicRoute({ name, children }: RouteShellProps) {
  return (
    <ErrorBoundary level="route" name={name}>
      <PublicRouteGuard>{children}</PublicRouteGuard>
    </ErrorBoundary>
  );
}

function AdminRoute({ name, children }: RouteShellProps) {
  return (
    <ProtectedRoute>
      <ErrorBoundary level="route" name={name}>
        {children}
      </ErrorBoundary>
    </ProtectedRoute>
  );
}

function AppRoute({ name, children }: RouteShellProps) {
  return (
    <ErrorBoundary level="route" name={name}>
      {children}
    </ErrorBoundary>
  );
}

export default function App() {
  return (
    <ErrorBoundary level="app" name="App">
      <BrowserRouter>
        <ToastProvider>
          <AdminAuthProvider>
            <UserAuthProvider>
              <LandingVariantProvider>
                <Routes>
                  {/* Public routes - guarded by coming soon mode */}
                  <Route
                    path="/"
                    element={(
                      <PublicRoute name="PublicLanding">
                        <PublicLanding />
                      </PublicRoute>
                    )}
                  />
                  <Route
                    path="/health"
                    element={(
                      <PublicRoute name="Health">
                        <PublicLanding />
                      </PublicRoute>
                    )}
                  />
                  <Route
                    path="/checkout"
                    element={(
                      <PublicRoute name="Checkout">
                        <CheckoutPage />
                      </PublicRoute>
                    )}
                  />
                  <Route
                    path="/feedback"
                    element={(
                      <PublicRoute name="Feedback">
                        <FeedbackPage />
                      </PublicRoute>
                    )}
                  />

                  {/* Admin login (unprotected) */}
                  <Route
                    path="/admin/login"
                    element={(
                      <AppRoute name="AdminLogin">
                        <AdminLogin />
                      </AppRoute>
                    )}
                  />

                  {/* User auth routes (unprotected, not gated by coming soon) */}
                  <Route
                    path="/auth/login"
                    element={(
                      <AppRoute name="UserLogin">
                        <UserLogin />
                      </AppRoute>
                    )}
                  />
                  <Route
                    path="/auth/verify"
                    element={(
                      <AppRoute name="VerifyMagicLink">
                        <VerifyMagicLink />
                      </AppRoute>
                    )}
                  />

                  {/* Protected admin routes */}
                  <Route
                    path="/admin"
                    element={(
                      <AdminRoute name="AdminHome">
                        <AdminHome />
                      </AdminRoute>
                    )}
                  />

                  {/* Section Dashboard Pages */}
                  <Route
                    path="/admin/landing"
                    element={(
                      <AdminRoute name="LandingDashboard">
                        <LandingDashboard />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/billing-home"
                    element={(
                      <AdminRoute name="BillingDashboard">
                        <BillingDashboard />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/users"
                    element={(
                      <AdminRoute name="UsersDashboard">
                        <UsersDashboard />
                      </AdminRoute>
                    )}
                  />

                  <Route
                    path="/admin/analytics"
                    element={(
                      <AdminRoute name="AdminAnalytics">
                        <AdminAnalytics />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/analytics/:variantSlug"
                    element={(
                      <AdminRoute name="AdminAnalyticsVariant">
                        <AdminAnalytics />
                      </AdminRoute>
                    )}
                  />

                  <Route
                    path="/admin/customization"
                    element={(
                      <AdminRoute name="Customization">
                        <Customization />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/billing"
                    element={(
                      <AdminRoute name="BillingSettings">
                        <BillingSettings />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/downloads"
                    element={(
                      <AdminRoute name="DownloadSettings">
                        <DownloadSettings />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/branding"
                    element={(
                      <AdminRoute name="BrandingSettings">
                        <BrandingSettings />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/profile"
                    element={(
                      <AdminRoute name="ProfileSettings">
                        <ProfileSettings />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/docs"
                    element={(
                      <AdminRoute name="DocsViewer">
                        <DocsViewer />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/feedback"
                    element={(
                      <AdminRoute name="FeedbackManagement">
                        <FeedbackManagement />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/waitlist"
                    element={(
                      <AdminRoute name="WaitlistManagement">
                        <WaitlistManagement />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/customization/agent"
                    element={(
                      <AdminRoute name="AgentCustomization">
                        <AgentCustomization />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/customization/variants/:slug"
                    element={(
                      <AdminRoute name="VariantEditor">
                        <VariantEditor />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/customization/variants/:variantSlug/sections/:sectionId"
                    element={(
                      <AdminRoute name="SectionEditor">
                        <SectionEditor />
                      </AdminRoute>
                    )}
                  />

                  {/* Credit System Routes */}
                  <Route
                    path="/admin/api-keys"
                    element={(
                      <AdminRoute name="APIKeysSettings">
                        <APIKeysSettings />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/tier-limits"
                    element={(
                      <AdminRoute name="TierLimitsSettings">
                        <TierLimitsSettings />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/app-limits"
                    element={(
                      <AdminRoute name="AppLimitsSettings">
                        <AppLimitsSettings />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/usage"
                    element={(
                      <AdminRoute name="UsageDashboard">
                        <UsageDashboard />
                      </AdminRoute>
                    )}
                  />

                  {/* Stub Pages */}
                  <Route
                    path="/admin/apps"
                    element={(
                      <AdminRoute name="AppsManagement">
                        <AppsManagement />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/tiers"
                    element={(
                      <AdminRoute name="TiersManagement">
                        <TiersManagement />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/coupons"
                    element={(
                      <AdminRoute name="CouponsManagement">
                        <CouponsManagement />
                      </AdminRoute>
                    )}
                  />
                  <Route
                    path="/admin/accounts"
                    element={(
                      <AdminRoute name="UserAccounts">
                        <UserAccounts />
                      </AdminRoute>
                    )}
                  />

                  <Route path="*" element={<Navigate to="/" replace />} />
                </Routes>
              </LandingVariantProvider>
            </UserAuthProvider>
          </AdminAuthProvider>
        </ToastProvider>
      </BrowserRouter>
    </ErrorBoundary>
  );
}
