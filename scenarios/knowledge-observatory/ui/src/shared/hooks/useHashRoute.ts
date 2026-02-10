// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
import { useCallback, useEffect, useState } from "react";
import {
  collectionToHash,
  parseCollectionNameFromHash,
  parseRouteFromHash,
  routeToHash,
  type Route,
} from "../controllers/routeController";

export function useHashRoute() {
  const [route, setRoute] = useState<Route>(() => (typeof window === "undefined" ? "dashboard" : parseRouteFromHash(window.location.hash)));
  const [collectionName, setCollectionName] = useState<string>(() =>
    typeof window === "undefined" ? "" : parseCollectionNameFromHash(window.location.hash)
  );

  useEffect(() => {
    if (typeof window === "undefined") return;
    const onHashChange = () => {
      setRoute(parseRouteFromHash(window.location.hash));
      setCollectionName(parseCollectionNameFromHash(window.location.hash));
    };
    window.addEventListener("hashchange", onHashChange);
    return () => window.removeEventListener("hashchange", onHashChange);
  }, []);

  const navigate = useCallback((nextRoute: Route) => {
    if (typeof window !== "undefined") {
      window.location.hash = routeToHash(nextRoute);
    }
    setRoute(nextRoute);
    if (nextRoute !== "collection") {
      setCollectionName("");
    }
  }, []);

  const navigateToCollection = useCallback((name: string) => {
    const normalized = name.trim();
    if (!normalized) {
      return;
    }
    if (typeof window !== "undefined") {
      window.location.hash = collectionToHash(normalized);
    }
    setRoute("collection");
    setCollectionName(normalized);
  }, []);

  return { route, navigate, collectionName, navigateToCollection };
}
