import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { getProxyInfo } from '@vrooli/api-base';
import { AdminAuthProvider } from './app/providers/AdminAuthProvider';
import { LandingVariantProvider } from './app/providers/LandingVariantProvider';
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
import { PublicLanding } from './surfaces/public-landing/routes/PublicLanding';

// getRouterBasename resolves the deployment sub-path from the proxy info so
// route matching works when the app is mounted behind a reverse proxy prefix.
function getRouterBasename(): string {
  const info = getProxyInfo();
  const path = info?.primary?.path ?? info?.basePath;
  if (!path) return '';
  return path.replace(/\/+$/, '');
}

export default function App() {
  const basename = getRouterBasename();
  return (
    <BrowserRouter basename={basename} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <AdminAuthProvider>
        <LandingVariantProvider>
          <Routes>
            <Route path="/" element={<PublicLanding />} />
            <Route path="/health" element={<PublicLanding />} />

            <Route path="/admin/login" element={<AdminLogin />} />

            <Route
              path="/admin"
              element={
                <ProtectedRoute>
                  <AdminHome />
                </ProtectedRoute>
              }
            />

            <Route
              path="/admin/analytics"
              element={
                <ProtectedRoute>
                  <AdminAnalytics />
                </ProtectedRoute>
              }
            />
            <Route
              path="/admin/analytics/:variantSlug"
              element={
                <ProtectedRoute>
                  <AdminAnalytics />
                </ProtectedRoute>
              }
            />

            <Route
              path="/admin/customization"
              element={
                <ProtectedRoute>
                  <Customization />
                </ProtectedRoute>
              }
            />
            <Route
              path="/admin/billing"
              element={
                <ProtectedRoute>
                  <BillingSettings />
                </ProtectedRoute>
              }
            />
            <Route
              path="/admin/downloads"
              element={
                <ProtectedRoute>
                  <DownloadSettings />
                </ProtectedRoute>
              }
            />
            <Route
              path="/admin/branding"
              element={
                <ProtectedRoute>
                  <BrandingSettings />
                </ProtectedRoute>
              }
            />
            <Route
              path="/admin/docs"
              element={
                <ProtectedRoute>
                  <DocsViewer />
                </ProtectedRoute>
              }
            />
            <Route
              path="/admin/customization/agent"
              element={
                <ProtectedRoute>
                  <AgentCustomization />
                </ProtectedRoute>
              }
            />
            <Route
              path="/admin/customization/variants/:slug"
              element={
                <ProtectedRoute>
                  <VariantEditor />
                </ProtectedRoute>
              }
            />
            <Route
              path="/admin/customization/variants/:variantSlug/sections/:sectionId"
              element={
                <ProtectedRoute>
                  <SectionEditor />
                </ProtectedRoute>
              }
            />

            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </LandingVariantProvider>
      </AdminAuthProvider>
    </BrowserRouter>
  );
}
