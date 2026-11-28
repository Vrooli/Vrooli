import { useEffect, useState } from "react";
import { useFilePreviewDeepLinkStore } from "../stores/useFilePreviewDeepLinkStore";

const NO_ACTIVE_SESSION = "No active session is available to open this file. Start or select a session, then retry.";

/** Defer a file notification until the persisted workspace has a usable pane. */
export function useFilePreviewDeepLink(options: {
  hydrated: boolean;
  activeSessionId: string | null;
  hasActiveSession: boolean;
  open: (sessionId: string, path: string) => void;
}) {
  const path = useFilePreviewDeepLinkStore((state) => state.path);
  const retryVersion = useFilePreviewDeepLinkStore((state) => state.retryVersion);
  const clear = useFilePreviewDeepLinkStore((state) => state.clear);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!path || !options.hydrated) return;
    if (!options.activeSessionId || !options.hasActiveSession) {
      setError(NO_ACTIVE_SESSION);
      return;
    }
    options.open(options.activeSessionId, path);
    setError(null);
    clear();
  }, [path, options.hydrated, options.activeSessionId, options.hasActiveSession, options.open, clear, retryVersion]);

  return {
    error,
    retry: () => { setError(null); useFilePreviewDeepLinkStore.getState().retry(); },
  };
}
