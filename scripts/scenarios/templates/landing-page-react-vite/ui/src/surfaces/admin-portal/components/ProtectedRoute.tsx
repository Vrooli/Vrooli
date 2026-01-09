import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { useAdminAuth } from '../../../app/providers/AdminAuthProvider';

export function ProtectedRoute({ children }: { children?: React.ReactNode }) {
  const location = useLocation();
  const { isAuthenticated, isSessionLoading } = useAdminAuth();

  if (isSessionLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-950 text-slate-200">
        <div className="text-center">
          <p className="text-sm uppercase tracking-wide text-slate-500">Verifying admin session</p>
          <p className="text-lg font-semibold mt-2">Please hold on...</p>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/admin/login" replace state={{ from: location.pathname }} />;
  }

  if (children) {
    return <>{children}</>;
  }

  return <Outlet />;
}
