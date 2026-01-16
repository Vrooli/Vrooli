import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AdminAuthProvider } from './app/providers/AdminAuthProvider';
import { LandingVariantProvider } from './app/providers/LandingVariantProvider';
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
import { PublicLanding } from './surfaces/public-landing/routes/PublicLanding';
import { CheckoutPage } from './surfaces/public-landing/routes/CheckoutPage';
import { FeedbackPage } from './surfaces/public-landing/routes/FeedbackPage';
import { ProfileSettings } from './surfaces/admin-portal/routes/ProfileSettings';

export default function App() {
  return (
    <ErrorBoundary level="app" name="App">
      <BrowserRouter>
        <ToastProvider>
          <AdminAuthProvider>
            <LandingVariantProvider>
            <Routes>
              {/* Public routes */}
              <Route
                path="/"
                element={
                  <ErrorBoundary level="route" name="PublicLanding">
                    <PublicLanding />
                  </ErrorBoundary>
                }
              />
              <Route
                path="/health"
                element={
                  <ErrorBoundary level="route" name="Health">
                    <PublicLanding />
                  </ErrorBoundary>
                }
              />
              <Route
                path="/checkout"
                element={
                  <ErrorBoundary level="route" name="Checkout">
                    <CheckoutPage />
                  </ErrorBoundary>
                }
              />
              <Route
                path="/feedback"
                element={
                  <ErrorBoundary level="route" name="Feedback">
                    <FeedbackPage />
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

              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
            </LandingVariantProvider>
          </AdminAuthProvider>
        </ToastProvider>
      </BrowserRouter>
    </ErrorBoundary>
  );
}
