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
      <div className="min-h-screen bg-[#07090F] flex items-center justify-center">
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
                element={
                  <ErrorBoundary level="route" name="PublicLanding">
                    <PublicRouteGuard>
                      <PublicLanding />
                    </PublicRouteGuard>
                  </ErrorBoundary>
                }
              />
              <Route
                path="/health"
                element={
                  <ErrorBoundary level="route" name="Health">
                    <PublicRouteGuard>
                      <PublicLanding />
                    </PublicRouteGuard>
                  </ErrorBoundary>
                }
              />
              <Route
                path="/checkout"
                element={
                  <ErrorBoundary level="route" name="Checkout">
                    <PublicRouteGuard>
                      <CheckoutPage />
                    </PublicRouteGuard>
                  </ErrorBoundary>
                }
              />
              <Route
                path="/feedback"
                element={
                  <ErrorBoundary level="route" name="Feedback">
                    <PublicRouteGuard>
                      <FeedbackPage />
                    </PublicRouteGuard>
                  </ErrorBoundary>
                }
              />

              {/* Admin login (unprotected) */}
              <Route
                path="/admin/login"
                element={
                  <ErrorBoundary level="route" name="AdminLogin">
                    <AdminLogin />
                  </ErrorBoundary>
                }
              />

              {/* User auth routes (unprotected, not gated by coming soon) */}
              <Route
                path="/auth/login"
                element={
                  <ErrorBoundary level="route" name="UserLogin">
                    <UserLogin />
                  </ErrorBoundary>
                }
              />
              <Route
                path="/auth/verify"
                element={
                  <ErrorBoundary level="route" name="VerifyMagicLink">
                    <VerifyMagicLink />
                  </ErrorBoundary>
                }
              />

              {/* Protected admin routes */}
              <Route
                path="/admin"
                element={
                  <ProtectedRoute>
                    <ErrorBoundary level="route" name="AdminHome">
                      <AdminHome />
                    </ErrorBoundary>
                  </ProtectedRoute>
                }
              />

              <Route
                path="/admin/analytics"
                element={
                  <ProtectedRoute>
                    <ErrorBoundary level="route" name="AdminAnalytics">
                      <AdminAnalytics />
                    </ErrorBoundary>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/admin/analytics/:variantSlug"
                element={
                  <ProtectedRoute>
                    <ErrorBoundary level="route" name="AdminAnalyticsVariant">
                      <AdminAnalytics />
                    </ErrorBoundary>
                  </ProtectedRoute>
                }
              />

              <Route
                path="/admin/customization"
                element={
                  <ProtectedRoute>
                    <ErrorBoundary level="route" name="Customization">
                      <Customization />
                    </ErrorBoundary>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/admin/billing"
                element={
                  <ProtectedRoute>
                    <ErrorBoundary level="route" name="BillingSettings">
                      <BillingSettings />
                    </ErrorBoundary>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/admin/downloads"
                element={
                  <ProtectedRoute>
                    <ErrorBoundary level="route" name="DownloadSettings">
                      <DownloadSettings />
                    </ErrorBoundary>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/admin/branding"
                element={
                  <ProtectedRoute>
                    <ErrorBoundary level="route" name="BrandingSettings">
                      <BrandingSettings />
                    </ErrorBoundary>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/admin/profile"
                element={
                  <ProtectedRoute>
                    <ErrorBoundary level="route" name="ProfileSettings">
                      <ProfileSettings />
                    </ErrorBoundary>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/admin/docs"
                element={
                  <ProtectedRoute>
                    <ErrorBoundary level="route" name="DocsViewer">
                      <DocsViewer />
                    </ErrorBoundary>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/admin/feedback"
                element={
                  <ProtectedRoute>
                    <ErrorBoundary level="route" name="FeedbackManagement">
                      <FeedbackManagement />
                    </ErrorBoundary>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/admin/waitlist"
                element={
                  <ProtectedRoute>
                    <ErrorBoundary level="route" name="WaitlistManagement">
                      <WaitlistManagement />
                    </ErrorBoundary>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/admin/customization/agent"
                element={
                  <ProtectedRoute>
                    <ErrorBoundary level="route" name="AgentCustomization">
                      <AgentCustomization />
                    </ErrorBoundary>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/admin/customization/variants/:slug"
                element={
                  <ProtectedRoute>
                    <ErrorBoundary level="route" name="VariantEditor">
                      <VariantEditor />
                    </ErrorBoundary>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/admin/customization/variants/:variantSlug/sections/:sectionId"
                element={
                  <ProtectedRoute>
                    <ErrorBoundary level="route" name="SectionEditor">
                      <SectionEditor />
                    </ErrorBoundary>
                  </ProtectedRoute>
                }
              />

              {/* Credit System Routes */}
              <Route
                path="/admin/api-keys"
                element={
                  <ProtectedRoute>
                    <ErrorBoundary level="route" name="APIKeysSettings">
                      <APIKeysSettings />
                    </ErrorBoundary>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/admin/tier-limits"
                element={
                  <ProtectedRoute>
                    <ErrorBoundary level="route" name="TierLimitsSettings">
                      <TierLimitsSettings />
                    </ErrorBoundary>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/admin/app-limits"
                element={
                  <ProtectedRoute>
                    <ErrorBoundary level="route" name="AppLimitsSettings">
                      <AppLimitsSettings />
                    </ErrorBoundary>
                  </ProtectedRoute>
                }
              />
              <Route
                path="/admin/usage"
                element={
                  <ProtectedRoute>
                    <ErrorBoundary level="route" name="UsageDashboard">
                      <UsageDashboard />
                    </ErrorBoundary>
                  </ProtectedRoute>
                }
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
