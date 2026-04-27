import { ReactNode } from 'react';
import { Route } from 'react-router-dom';
import { ErrorBoundary } from '../../shared/ui/ErrorBoundary';
import { AdminLogin } from '../../surfaces/admin-portal/routes/AdminLogin';
import { UserLogin, VerifyMagicLink } from '../../surfaces/user-auth';

function AppRoute({ name, children }: { name: string; children: ReactNode }) {
  return (
    <ErrorBoundary level="route" name={name}>
      {children}
    </ErrorBoundary>
  );
}

export const userAuthRoutes = (
  <>
    <Route path="/admin/login" element={<AppRoute name="AdminLogin"><AdminLogin /></AppRoute>} />
    <Route path="/auth/login" element={<AppRoute name="UserLogin"><UserLogin /></AppRoute>} />
    <Route path="/auth/verify" element={<AppRoute name="VerifyMagicLink"><VerifyMagicLink /></AppRoute>} />
  </>
);
