import { useEffect } from "react";

/**
 * Sets `document.title` to "<page> · <app>" while the calling component is
 * mounted, restoring the previous title on unmount. Gives every surface a
 * distinct, screen-reader-announced page title (navigation-integrity-audit).
 */
export function useDocumentTitle(title: string, appName = "Tunnel Manager") {
  useEffect(() => {
    const previous = document.title;
    document.title = `${title} · ${appName}`;
    return () => {
      document.title = previous;
    };
  }, [title, appName]);
}
