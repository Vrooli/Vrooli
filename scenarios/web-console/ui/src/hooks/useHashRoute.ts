import { useState, useEffect, useCallback } from "react";

export type Route = "workspace" | "settings" | "sessions";

const VALID_ROUTES: Route[] = ["workspace", "settings", "sessions"];

function parseHash(): Route {
  const raw = window.location.hash.replace("#/", "").split("?")[0];
  if (VALID_ROUTES.includes(raw as Route)) return raw as Route;
  return "workspace";
}

/**
 * Simple hash-based router for single-page navigation.
 * Routes are defined as `#/workspace`, `#/settings`, `#/sessions`.
 */
export function useHashRoute() {
  const [route, setRoute] = useState<Route>(parseHash);

  useEffect(() => {
    const handler = () => setRoute(parseHash());
    window.addEventListener("hashchange", handler);
    return () => window.removeEventListener("hashchange", handler);
  }, []);

  const navigate = useCallback((to: Route) => {
    window.location.hash = `#/${to}`;
  }, []);

  return { route, navigate };
}
