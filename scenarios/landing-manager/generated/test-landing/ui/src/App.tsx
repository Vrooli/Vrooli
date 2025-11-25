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

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <VariantProvider>
          <Routes>
            <Route path="/" element={<PublicHome />} />
            <Route path="/health" element={<PublicHome />} />

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
        </VariantProvider>
      </AuthProvider>
    </BrowserRouter>
  );
}
