import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { listRecoverableSessions } from "../../api/sessions";
import { useSessionRecovery } from "../../hooks/useSessionRecovery";
import { crashRecoveryBanner, sessionRecoveryBanner } from "./descriptors";
import type { MaybeBanner } from "./types";

/**
 * Startup session recovery, as a banner.
 *
 * While the API reattaches surviving tmux sessions in the background the
 * session list can be incomplete — without this the workspace looks empty and
 * the user assumes their durable sessions were lost. Returns null when no
 * recovery is (or was just) observed, so steady-state chrome is unchanged.
 */
export function useSessionRecoveryBanner(): MaybeBanner {
  const { t } = useTranslation();
  const { inProgress, total, recovered, adopted, justCompleted } = useSessionRecovery();
  if (!inProgress && !justCompleted) return null;
  return sessionRecoveryBanner(t, { inProgress, total, recovered, adopted });
}

/**
 * Sessions left behind by a previous run, as a banner. Refreshes on the
 * `web-console:recoverable-changed` event the archive drawer emits.
 */
export function useCrashRecoveryBanner(onOpenArchive: () => void): MaybeBanner {
  const { t } = useTranslation();
  const [count, setCount] = useState(0);

  const refresh = useCallback(() => {
    void listRecoverableSessions()
      .then((rows) => { setCount(rows.length); })
      .catch(() => { setCount(0); });
  }, []);

  useEffect(() => {
    refresh();
    window.addEventListener("web-console:recoverable-changed", refresh);
    return () => { window.removeEventListener("web-console:recoverable-changed", refresh); };
  }, [refresh]);

  if (count === 0) return null;
  return crashRecoveryBanner(t, count, onOpenArchive);
}
