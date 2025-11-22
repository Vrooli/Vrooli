import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './contexts/AuthContext';
import { VariantProvider } from './contexts/VariantContext';
import { ProtectedRoute } from './components/ProtectedRoute';
import { AdminLogin } from './pages/AdminLogin';
import { AdminHome } from './pages/AdminHome';
import { AdminAnalytics } from './pages/AdminAnalytics';
import { Customization } from './pages/Customization';
import { VariantEditor } from './pages/VariantEditor';
import { SectionEditor } from './pages/SectionEditor';
import { AgentCustomization } from './pages/AgentCustomization';
import PublicHome from './pages/PublicHome';

/**
 * Main App Router
 *
 * Navigation structure (implements OT-P0-010: ≤ 3 clicks to any customization card):
 * - Click 1: Admin Home → Customization
 * - Click 2: Customization → Variant
 * - Click 3: Variant → Section Editor
 *
 * [REQ:ADMIN-NAV]
 */
export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <VariantProvider>
          <Routes>
            {/* Public routes */}
            <Route path="/" element={<PublicHome />} />
            <Route path="/health" element={<PublicHome />} />

            {/* Admin routes - hidden from public (OT-P0-007) */}
            <Route path="/admin/login" element={<AdminLogin />} />

            {/* Protected admin routes */}
            <Route
              path="/admin"
              element={
                <ProtectedRoute>
                  <AdminHome />
                </ProtectedRoute>
              }
            />

            {/* Analytics routes */}
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

            {/* Customization routes */}
            <Route
              path="/admin/customization"
              element={
                <ProtectedRoute>
                  <Customization />
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

            {/* 404 redirect */}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </VariantProvider>
      </AuthProvider>
    </BrowserRouter>
  );
}
