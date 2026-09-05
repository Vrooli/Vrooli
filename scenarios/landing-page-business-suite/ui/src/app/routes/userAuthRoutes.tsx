/* eslint-disable react-refresh/only-export-components */
import React, { ReactNode, lazy } from 'react';
import { Route } from 'react-router-dom';
import { ErrorBoundary } from '../../shared/ui/ErrorBoundary';
import { onProfilerRender } from '../../lib/profiler';

const AdminLogin = lazy(() =>
  import('../../surfaces/admin-portal/routes/AdminLogin').then((module) => ({
    default: module.AdminLogin,
  }))
);
const UserLogin = lazy(() =>
  import('../../surfaces/user-auth').then((module) => ({
    default: module.UserLogin,
  }))
);
const VerifyMagicLink = lazy(() =>
  import('../../surfaces/user-auth').then((module) => ({
    default: module.VerifyMagicLink,
  }))
);

function AppRoute({ name, children }: { name: string; children: ReactNode }) {
  return (
    <ErrorBoundary level="route" name={name}>
      <React.Profiler id={name} onRender={onProfilerRender}>
        {children}
      </React.Profiler>
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
