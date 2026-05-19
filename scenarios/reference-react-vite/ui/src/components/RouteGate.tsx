// RouteGate — enforces the `requires` predicates declared in
// ui/flow/navigation.json for each protected route. The deep-link
// policies in the spec are realised here:
//   - auth_required_routes_redirect_to_login: preserves the target
//     so the post-login redirect respects the original URL.
//   - admin_routes_redirect_non_admins / beta_routes_redirect_when_flag_off:
//     bounce back to /dashboard when the predicate is not met.
//   - auth_pages_redirect_when_already_logged_in: handled inline.
import { Navigate, Outlet, useLocation } from "react-router-dom";
import type { ReactNode } from "react";
import { useAppContext } from "../contexts/AppContext";
import { ROUTES } from "../routes.generated";

interface Props {
  requireAuth?: boolean;
  requireRole?: "admin";
  requireBeta?: boolean;
  redirectLoggedIn?: boolean;
  children?: ReactNode;
}

export function RouteGate({
  requireAuth,
  requireRole,
  requireBeta,
  redirectLoggedIn,
  children,
}: Props) {
  const { auth, role, featureBeta } = useAppContext();
  const location = useLocation();

  if (redirectLoggedIn && auth === "logged_in") {
    return <Navigate to={ROUTES.dashboard} replace />;
  }
  if (requireAuth && auth !== "logged_in") {
    const target = encodeURIComponent(location.pathname + location.search);
    return <Navigate to={`${ROUTES.login}?next=${target}`} replace />;
  }
  if (requireRole === "admin" && role !== "admin") {
    return <Navigate to={ROUTES.dashboard} replace />;
  }
  if (requireBeta && !featureBeta) {
    return <Navigate to={ROUTES.dashboard} replace />;
  }
  return <>{children ?? <Outlet />}</>;
}
