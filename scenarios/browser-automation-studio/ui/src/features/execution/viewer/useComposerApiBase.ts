import { useEffect, useState } from "react";
import { getConfig } from "@/config";
import { logger } from "@utils/logger";
import { stripApiSuffix } from "../utils/exportHelpers";

/**
 * Derives the API base used by the replay composer iframe and keeps it cached on window
 * so the embedded composer can resolve asset URLs consistently across reloads.
 */
export const useComposerApiBase = (executionId: string) => {
  const [composerApiBase, setComposerApiBase] = useState<string | null>(() => {
    if (typeof window === "undefined") {
      return null;
    }
    const existing = (
      window as typeof window & {
        __BAS_EXPORT_API_BASE__?: unknown;
      }
    ).__BAS_EXPORT_API_BASE__;
    return typeof existing === "string" && existing.trim().length > 0
      ? existing.trim()
      : null;
  });

  useEffect(() => {
    if (composerApiBase) {
      return;
    }

    let cancelled = false;

    (async () => {
      try {
        const configData = await getConfig();
        if (cancelled) {
          return;
        }
        const derived = stripApiSuffix(configData.API_URL);
        if (derived) {
          setComposerApiBase((current) =>
            current && current.length > 0 ? current : derived,
          );
        }
      } catch (error) {
        if (!cancelled) {
          logger.warn(
            "Failed to derive API base for replay composer",
            { component: "ExecutionViewer", executionId },
            error,
          );
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [composerApiBase, executionId]);

  useEffect(() => {
    if (typeof window === "undefined" || !composerApiBase) {
      return;
    }
    (
      window as typeof window & { __BAS_EXPORT_API_BASE__?: string }
    ).__BAS_EXPORT_API_BASE__ = composerApiBase;
  }, [composerApiBase]);

  return { composerApiBase, setComposerApiBase };
};

export default useComposerApiBase;
