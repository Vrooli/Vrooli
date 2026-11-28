import { installMicLifecycleCleanup, type LifecycleReleaseScope } from "./micOwnership";
import { decideMicLifecycle, isStandaloneDisplayMode } from "./micLifecyclePolicy";

export interface VoicePageLifecycleOptions {
  onVisibilityChange: (hidden: boolean) => void;
  cancelPendingStarts: () => void;
  isActiveRecording: () => boolean;
  stopActiveRecording: () => void;
  suspendAudio: () => void;
}

/** Install the voice hook's page visibility handler and the shared mic backstop. */
export function installVoicePageLifecycle(options: VoicePageLifecycleOptions): () => void {
  const resolveScope = (event: "hidden" | "pagehide" | "freeze"): LifecycleReleaseScope =>
    decideMicLifecycle({ event, standalonePwa: isStandaloneDisplayMode() }).release === "all"
      ? "all"
      : "non-active";
  const uninstallBackstop = installMicLifecycleCleanup(resolveScope);

  const onVisibility = () => {
    const hidden = typeof document !== "undefined" && document.visibilityState === "hidden";
    options.onVisibilityChange(hidden);
    if (!hidden) return;

    options.cancelPendingStarts();
    if (options.isActiveRecording()) options.stopActiveRecording();
    options.suspendAudio();
  };

  document.addEventListener("visibilitychange", onVisibility);
  options.onVisibilityChange(typeof document !== "undefined" && document.visibilityState === "hidden");

  return () => {
    document.removeEventListener("visibilitychange", onVisibility);
    uninstallBackstop();
  };
}
