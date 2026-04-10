import { useState, useEffect, useCallback } from "react";

export type Route =
  | { page: "brands" }
  | { page: "brand-detail"; id: string }
  | { page: "brand-create" }
  | { page: "brand-edit"; id: string }
  | { page: "scanner" }
  | { page: "standards" };

function parseHash(): Route {
  const hash = window.location.hash.slice(1) || "/";
  if (hash === "/" || hash === "/brands") return { page: "brands" };
  if (hash === "/brands/new") return { page: "brand-create" };
  if (hash === "/scanner") return { page: "scanner" };
  if (hash === "/standards") return { page: "standards" };
  const editMatch = hash.match(/^\/brands\/([^/]+)\/edit$/);
  if (editMatch?.[1]) return { page: "brand-edit", id: editMatch[1] };
  const detailMatch = hash.match(/^\/brands\/([^/]+)$/);
  if (detailMatch?.[1]) return { page: "brand-detail", id: detailMatch[1] };
  return { page: "brands" };
}

export function useRouter() {
  const [route, setRoute] = useState<Route>(parseHash);

  useEffect(() => {
    const handler = () => setRoute(parseHash());
    window.addEventListener("hashchange", handler);
    return () => window.removeEventListener("hashchange", handler);
  }, []);

  const navigate = useCallback((path: string) => {
    window.location.hash = path;
  }, []);

  return { route, navigate };
}
