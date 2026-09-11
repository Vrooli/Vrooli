import { useEffect, type ReactNode } from "react";
import { ViewportEnvironmentProvider } from "@vrooli/react-component-library/useViewportEnvironment/1";
import { useCorrectedViewport } from "../hooks/useAppViewport";

/**
 * Set on <html> while the app is extended past the viewport the platform
 * reported. iOS paints no `position: fixed` content below that viewport, so
 * `styles.css` positions the body and the overlay roots against the page while
 * it is present; the page never scrolls, so the geometry is the same.
 */
export const VIEWPORT_EXTENDED_ATTRIBUTE = "data-wc-viewport-extended";

/**
 * Hands every library overlay the corrected viewport, so a drawer sized to the
 * viewport ends where the app does rather than where a mis-measured viewport
 * says. It reads the platform's own snapshot, so it alone knows whether the
 * correction extended the app. See lib/viewportCorrection.ts.
 */
export function AppViewportProvider({ children }: { children: ReactNode }) {
  const { viewport, extended } = useCorrectedViewport();

  useEffect(() => {
    const root = document.documentElement;
    if (extended) root.setAttribute(VIEWPORT_EXTENDED_ATTRIBUTE, "");
    else root.removeAttribute(VIEWPORT_EXTENDED_ATTRIBUTE);
  }, [extended]);
  useEffect(() => () => { document.documentElement.removeAttribute(VIEWPORT_EXTENDED_ATTRIBUTE); }, []);

  return <ViewportEnvironmentProvider value={viewport}>{children}</ViewportEnvironmentProvider>;
}
