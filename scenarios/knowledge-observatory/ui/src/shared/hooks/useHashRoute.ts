import { useCallback, useEffect, useState } from "react";
import { parseRouteFromHash, routeToHash, type Route } from "../controllers/routeController";

export function useHashRoute() {
  const [route, setRoute] = useState<Route>(() =>
    typeof window === "undefined" ? "dashboard" : parseRouteFromHash(window.location.hash)
  );

  useEffect(() => {
    if (typeof window === "undefined") return;
    const onHashChange = () => setRoute(parseRouteFromHash(window.location.hash));
    window.addEventListener("hashchange", onHashChange);
    return () => window.removeEventListener("hashchange", onHashChange);
  }, []);

  const navigate = useCallback((nextRoute: Route) => {
    if (typeof window !== "undefined") {
      window.location.hash = routeToHash(nextRoute);
    }
    setRoute(nextRoute);
  }, []);

  return { route, navigate };
}
