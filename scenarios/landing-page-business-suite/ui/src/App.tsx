// DOC: docs/concepts/ARCHITECTURE.md - UI architecture and component structure
// DOC: docs/QUICKSTART.md - Getting started guide
// DOC: docs/guides/ADMIN_GUIDE.md - Admin portal usage
//
// App.tsx is a thin composer: it wires the providers and delegates the
// route table to per-surface modules under app/routes/.
import { Suspense } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AdminAuthProvider } from './app/providers/AdminAuthProvider';
import { UserAuthProvider } from './app/providers/UserAuthProvider';
import { LandingVariantProvider } from './app/providers/LandingVariantProvider';
import { ErrorBoundary } from './shared/ui/ErrorBoundary';
import { ToastProvider } from './shared/ui/Toast';
import { publicRoutes } from './app/routes/publicRoutes';
import { adminRoutes } from './app/routes/adminRoutes';
import { userAuthRoutes } from './app/routes/userAuthRoutes';

export default function App() {
  return (
    <ErrorBoundary level="app" name="App">
      <BrowserRouter>
        <ToastProvider>
          <AdminAuthProvider>
            <UserAuthProvider>
              <LandingVariantProvider>
                <Suspense fallback={<RouteLoadingFallback />}>
                  <Routes>
                    {publicRoutes}
                    {userAuthRoutes}
                    {adminRoutes}
                    <Route path="*" element={<Navigate to="/" replace />} />
                  </Routes>
                </Suspense>
              </LandingVariantProvider>
            </UserAuthProvider>
          </AdminAuthProvider>
        </ToastProvider>
      </BrowserRouter>
    </ErrorBoundary>
  );
}

function RouteLoadingFallback() {
  return (
    <div className="min-h-screen bg-bg-base flex items-center justify-center">
      <div className="animate-pulse text-slate-400">Loading...</div>
    </div>
  );
}
