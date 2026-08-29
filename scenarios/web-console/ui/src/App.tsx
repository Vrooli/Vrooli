// DOC: docs/concepts/ARCHITECTURE.md#system-layers
import { lazy, Suspense, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { fetchHealth } from "./api/health";
import { HEALTH_RETRY_COUNT, HEALTH_RETRY_DELAY_MS } from "./consts/config";
import { strings } from "./consts/strings";
import { HandednessProvider } from "@vrooli/react-component-library/useHandedness";
import ErrorBoundary from "./components/ErrorBoundary";
import { useWorkspaceStore } from "./stores/useWorkspaceStore";
import {
  audioUnavailableBanner,
  connectionBanner,
} from "./components/banners/descriptors";
import type { MaybeBanner } from "./components/banners/types";
import { useCapabilities } from "./hooks/useCapabilities";

const Workspace = lazy(() => import("./components/Workspace"));

const PageFallback = () => {
  const { t } = useTranslation();
  return (
    <div className="flex h-wc-app items-center justify-center bg-wc-surface-base text-wc-text-muted">
      {t(strings.app.loading)}
    </div>
  );
};

export default function App() {
  const handedness = useWorkspaceStore((state) => state.handedness);
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [dismissed, setDismissed] = useState(false);
  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    retry: HEALTH_RETRY_COUNT,
    retryDelay: HEALTH_RETRY_DELAY_MS,
    // Keep polling so banner auto-clears when connection recovers
	refetchInterval: (query) => query.state.status === "error" ? 10 * 1000 : false,
  });
  const capabilitiesQuery = useCapabilities();
  const { isLoading, error, isFetching } = healthQuery;

  const audioCapability = capabilitiesQuery.data?.capabilities.find((capability) => capability.id === "audio-tools");
  // The API's own `message` rides along: it covers every reason code, including
  // the ones the UI has no bespoke copy for.
  const audioUnavailable = capabilitiesQuery.error
    ? { reason: "discovery_failed" }
    : audioCapability?.status === "unavailable"
      ? {
          reason: audioCapability.reasonCode || "scenario_not_running",
          message: audioCapability.message,
        }
      : null;

  // Reset dismissed state when connection recovers or drops again
  const showBanner = !!error && !dismissed;

  /**
   * App-level notices are handed to Workspace rather than rendered here.
   * Two stacked banner surfaces meant two safe-area owners — App reserved the
   * notch and told Workspace to skip it via a `topSafeAreaReserved` prop — and
   * nothing arbitrated between them. One region, one owner, one arbitration.
   */
  const appBanners: MaybeBanner[] = [
    showBanner &&
      connectionBanner(t, {
        retrying: isFetching,
        onRetry: () => {
          setDismissed(false);
          void queryClient.invalidateQueries({ queryKey: ["health"] });
        },
        onDismiss: () => { setDismissed(true); },
      }),
    audioUnavailableBanner(t, audioUnavailable),
  ];

  return (
    // The reach-side preference is published once, at the root, so every
    // anchored surface below resolves the same answer. Writing direction is
    // applied on top of it by the library and is not this setting's concern.
    <HandednessProvider value={handedness}>
    <ErrorBoundary region="app">
      <div className="wc-ios-tint-edge wc-ios-tint-edge-bottom" aria-hidden="true" />

      {/* Always render workspace — even during initial load or error */}
      <Suspense fallback={<PageFallback />}>
        {isLoading && !error ? (
          <PageFallback />
        ) : (
          <Workspace appBanners={appBanners} />
        )}
      </Suspense>
    </ErrorBoundary>
    </HandednessProvider>
  );
}
