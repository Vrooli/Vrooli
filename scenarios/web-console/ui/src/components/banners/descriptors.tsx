import { History, Loader2, Mic, RefreshCw, RotateCcw, Volume2, VolumeX } from "lucide-react";
import type { TFunction } from "i18next";
import type { VoiceRejection } from "../../audio-integration";
import type { ErrorInfo } from "../../lib/errors";
import { strings } from "../../consts/strings";
import type { SummarizeErrorState } from "../../types/summarize";
import { BANNER_PRIORITY, type BannerDescriptor } from "./types";

/**
 * Every top-chrome notice, as data.
 *
 * These replaced eight bespoke banner components. Each one previously carried
 * its own markup, colour vocabulary and accessibility choices; what actually
 * differs between them is a title, a description, and one or two actions, so
 * that is all these functions return. Presentation is `Banner`, ordering is
 * `arbitrateBanners`.
 *
 * Banner `testId` values are unchanged from the components they replace, so
 * existing selectors and BAS cases keep resolving. Action test ids are now
 * derived uniformly as `${banner.testId}-${action.id}` rather than each banner
 * inventing its own convention (`summarize-error-retry` beside
 * `enable-audio-banner-enable`). One action kept its legacy id because BAS
 * depends on it — see `connectionBanner`.
 */

/** Format a duration like 12_500 → "12.5s". */
export function formatDurationMs(ms: number): string {
  if (ms < 1000) return `${String(ms)}ms`;
  const seconds = ms / 1000;
  if (seconds < 60) return `${seconds.toFixed(1)}s`;
  const minutes = Math.floor(seconds / 60);
  const rem = Math.round(seconds - minutes * 60);
  return `${String(minutes)}m${rem.toString().padStart(2, "0")}s`;
}

// ── System ────────────────────────────────────────────────────────────────

export function connectionBanner(
  t: TFunction,
  opts: { retrying: boolean; onRetry: () => void; onDismiss: () => void },
): BannerDescriptor {
  return {
    id: "connection-lost",
    testId: "connection-banner",
    tone: "danger",
    priority: BANNER_PRIORITY.connectionLost,
    title: t(strings.app.connectionBanner.message),
    actions: [
      {
        id: "retry",
        // Kept verbatim: the `open-workspace` BAS action clicks this until the
        // workspace appears, so it is an external contract, not a test detail.
        testId: "health-retry-button",
        label: opts.retrying
          ? t(strings.app.connectionBanner.retrying)
          : t(strings.app.connectionBanner.retry),
        onSelect: opts.onRetry,
        busy: opts.retrying,
        primary: true,
        icon: RefreshCw,
      },
    ],
    onDismiss: opts.onDismiss,
    dismissLabel: t(strings.app.connectionBanner.dismissAriaLabel),
  };
}

/**
 * Every dependency-status token `capabilities.checkers` can report, mapped to
 * the state it describes. Several codes distinguish causes that matter to an
 * operator reading logs but not to someone who just wants to dictate, so they
 * collapse onto shared copy.
 *
 * Kept exhaustive on purpose: an unmapped code falls back to the API's own
 * untranslated sentence, which works but is a sign copy is missing.
 */
const AUDIO_UNAVAILABLE_REASONS = {
  // Reachability, decided in the browser.
  discovery_failed: strings.banners.audioUnavailable.discoveryFailed,
  resolver_not_configured: strings.banners.audioUnavailable.resolverNotConfigured,

  // Not running.
  scenario_not_running: strings.banners.audioUnavailable.scenarioNotRunning,
  scenario_stopped: strings.banners.audioUnavailable.scenarioNotRunning,

  // Misconfigured.
  env_misconfigured: strings.banners.audioUnavailable.envMisconfigured,
  scenario_slug_missing: strings.banners.audioUnavailable.envMisconfigured,

  // On its way up — worth saying, because waiting is the right response.
  scenario_starting: strings.banners.audioUnavailable.starting,
  scenario_start_in_progress: strings.banners.audioUnavailable.starting,

  // Tried and failed.
  scenario_start_failed: strings.banners.audioUnavailable.startFailed,
  scenario_start_abandoned: strings.banners.audioUnavailable.startFailed,

  // We could not find out either way.
  scenario_status_cli_failed: strings.banners.audioUnavailable.statusUnknown,
  scenario_status_malformed_json: strings.banners.audioUnavailable.statusUnknown,
  scenario_status_missing: strings.banners.audioUnavailable.statusUnknown,
  scenario_status_missing_scenario: strings.banners.audioUnavailable.statusUnknown,
  scenario_status_unknown: strings.banners.audioUnavailable.statusUnknown,

  // Up, but unwell.
  scenario_degraded: strings.banners.audioUnavailable.degraded,
  scenario_health_error: strings.banners.audioUnavailable.degraded,
  scenario_health_not_running: strings.banners.audioUnavailable.degraded,
  scenario_health_unknown: strings.banners.audioUnavailable.degraded,
} as const;

export function audioUnavailableBanner(
  t: TFunction,
  unavailable: { reason?: string; message?: string } | null | undefined,
): BannerDescriptor | null {
  const reason = unavailable?.reason;
  if (!reason) return null;
  // Three tiers, best first. The API defines fifteen reason codes and only a
  // handful earn bespoke translated copy — but it also returns its own
  // human-readable `message`, which this banner used to discard in favour of
  // printing the raw token at the reader. An untranslated server sentence beats
  // "scenario_status_cli_failed". The token form is the last resort, and seeing
  // it means nobody has written copy for a state that actually occurs.
  const key = Object.prototype.hasOwnProperty.call(AUDIO_UNAVAILABLE_REASONS, reason)
    ? AUDIO_UNAVAILABLE_REASONS[reason as keyof typeof AUDIO_UNAVAILABLE_REASONS]
    : undefined;
  return {
    id: "audio-unavailable",
    testId: "audio-unavailable-banner",
    tone: "warning",
    priority: BANNER_PRIORITY.audioUnavailable,
    title: t(strings.banners.audioUnavailable.title),
    description:
      key ? t(key)
        : unavailable.message ?? t(strings.banners.audioUnavailable.generic, { reason }),
    data: { "data-audio-state": "unavailable" },
  };
}

export function createErrorBanner(
  t: TFunction,
  error: ErrorInfo,
  opts: { onDismiss: () => void; onRetry?: () => void },
): BannerDescriptor {
  return {
    id: "create-error",
    testId: "create-error-banner",
    tone: "danger",
    priority: BANNER_PRIORITY.createError,
    title: error.message,
    description: error.recovery,
    actions: opts.onRetry
      ? [
          {
            id: "retry-button",
            label: t(strings.errorBanner.retry),
            onSelect: opts.onRetry,
            primary: true,
            icon: RotateCcw,
          },
        ]
      : undefined,
    onDismiss: opts.onDismiss,
    dismissLabel: t(strings.errorBanner.dismiss),
  };
}

export function summarizeErrorBanner(
  t: TFunction,
  state: SummarizeErrorState,
  opts: { onRetry: () => void; onDismiss: () => void },
): BannerDescriptor {
  const retrying = state.status === "retrying";
  return {
    id: "summarize-error",
    testId: "summarize-error-banner",
    tone: "danger",
    priority: BANNER_PRIORITY.summarizeError,
    title:
      state.source === "auto"
        ? t(strings.summarizeError.autoFailed)
        : t(strings.summarizeError.failed),
    description: state.message,
    data: { "data-source": state.source, "data-status": state.status },
    actions: [
      {
        id: "retry",
        label: retrying ? t(strings.summarizeError.retrying) : t(strings.summarizeError.retry),
        title: t(strings.summarizeError.retryTitle),
        onSelect: opts.onRetry,
        busy: retrying,
        primary: true,
        icon: RotateCcw,
      },
    ],
    onDismiss: opts.onDismiss,
    dismissLabel: t(strings.summarizeError.dismissAriaLabel),
  };
}

export function enableAudioBanner(
  t: TFunction,
  opts: { enabling: boolean; onEnable: () => void; onDismiss: () => void },
): BannerDescriptor {
  return {
    id: "enable-audio",
    testId: "enable-audio-banner",
    tone: "info",
    priority: BANNER_PRIORITY.enableAudio,
    icon: Volume2,
    title: t(strings.enableAudioBanner.title),
    description: t(strings.enableAudioBanner.description),
    data: { "data-audio-state": "enable-audio" },
    actions: [
      {
        id: "enable",
        label: opts.enabling
          ? t(strings.enableAudioBanner.enabling)
          : t(strings.enableAudioBanner.enable),
        title: t(strings.enableAudioBanner.enableTitle),
        onSelect: opts.onEnable,
        busy: opts.enabling,
        primary: true,
        icon: Volume2,
      },
    ],
    // Not withdrawn while enabling: `Banner` disables the close button for the
    // duration of a busy action, which keeps the footprint stable.
    onDismiss: opts.onDismiss,
    dismissLabel: t(strings.enableAudioBanner.dismiss),
  };
}

export function trackingDegradedBanner(t: TFunction): BannerDescriptor {
  return {
    id: "tracking-degraded",
    testId: "tracking-degraded-banner",
    tone: "info",
    priority: BANNER_PRIORITY.trackingDegraded,
    title: t(strings.banners.trackingDegraded),
  };
}

// ── Voice ─────────────────────────────────────────────────────────────────

export function voiceFallbackBanner(
  t: TFunction,
  notice: string,
  onDismiss: () => void,
): BannerDescriptor {
  return {
    id: "voice-fallback",
    testId: "voice-status-banner",
    tone: "warning",
    priority: BANNER_PRIORITY.voiceFallback,
    title: notice,
    onDismiss,
    dismissLabel: t(strings.banners.dismiss),
  };
}

export function voiceErrorBanner(
  t: TFunction,
  message: string,
): BannerDescriptor {
  return {
    id: "voice-error",
    testId: "voice-error-banner",
    tone: "warning",
    priority: BANNER_PRIORITY.voiceError,
    icon: Mic,
    title: t(strings.voiceRecovery.errorTitle),
    description: message,
  };
}

export function voiceTranscribingBanner(
  t: TFunction,
  onCancel: () => void,
): BannerDescriptor {
  return {
    id: "voice-transcribing",
    testId: "voice-transcribing-banner",
    tone: "info",
    priority: BANNER_PRIORITY.voiceTranscribing,
    icon: Loader2,
    spin: true,
    title: t(strings.voiceRecovery.transcribingTitle),
    actions: [
      { id: "cancel", label: t(strings.voiceRecovery.cancel), onSelect: onCancel, primary: true },
    ],
  };
}

export function voiceStaleMicBanner(
  t: TFunction,
  onRelease: () => void,
): BannerDescriptor {
  return {
    id: "voice-stale-mic",
    testId: "voice-stale-mic-banner",
    tone: "warning",
    priority: BANNER_PRIORITY.voiceStaleMic,
    icon: Mic,
    title: t(strings.voiceRecovery.staleMicTitle),
    actions: [
      {
        id: "release-mic",
        label: t(strings.voiceRecovery.releaseMic),
        onSelect: onRelease,
        primary: true,
      },
    ],
  };
}

export function ttsSpeakingBanner(t: TFunction, onStop: () => void): BannerDescriptor {
  return {
    id: "tts-speaking",
    testId: "tts-speaking-banner",
    tone: "info",
    priority: BANNER_PRIORITY.ttsSpeaking,
    icon: Volume2,
    title: t(strings.voiceRecovery.ttsSpeakingTitle),
    actions: [
      {
        id: "stop-speech",
        label: t(strings.voiceRecovery.stopSpeech),
        onSelect: onStop,
        primary: true,
        icon: VolumeX,
      },
    ],
  };
}

export function voiceRejectionBanner(
  t: TFunction,
  rejection: VoiceRejection,
  opts: { onRetry: () => void; onDismiss: () => void },
): BannerDescriptor {
  const base = {
    id: "voice-rejection",
    testId: "voice-rejection-banner",
    tone: "warning" as const,
    priority: BANNER_PRIORITY.voiceRejection,
    onDismiss: opts.onDismiss,
    dismissLabel: t(strings.voiceRejection.dismissAriaLabel),
  };

  if (rejection.kind === "explanatory") {
    return {
      ...base,
      title: t(strings.voiceRejection.title),
      description: t(strings.voiceRejection.explanatoryDetail, {
        reason: rejection.reason,
        score: rejection.score.toFixed(2),
        threshold: rejection.threshold.toFixed(2),
      }),
      data: { "data-audio-state": "rejection", "data-kind": "explanatory" },
    };
  }

  const retrying = rejection.status === "retrying";
  const failed = rejection.status === "failed";
  const empty = rejection.cause === "empty-transcript";

  return {
    ...base,
    title: empty ? t(strings.voiceRejection.emptyTitle) : t(strings.voiceRejection.title),
    description: empty
      ? t(strings.voiceRejection.emptyDetail, {
          duration: formatDurationMs(rejection.durationMs),
        })
      : t(strings.voiceRejection.retryableDetail, {
          score: rejection.score.toFixed(2),
          threshold: rejection.threshold.toFixed(2),
          duration: formatDurationMs(rejection.durationMs),
        }),
    detail: failed && rejection.errorMessage ? rejection.errorMessage : undefined,
    data: {
      "data-audio-state": "rejection",
      "data-kind": "retryable",
      "data-cause": rejection.cause,
      "data-status": rejection.status,
    },
    actions: [
      {
        id: "retry",
        // An empty-transcript turn was never a "match" question — the verb is
        // always "Retry". A speaker-rejected turn offers "Transcribe anyway"
        // until it fails.
        label: retrying
          ? t(strings.voiceRejection.transcribing)
          : empty || failed
            ? t(strings.voiceRejection.retry)
            : t(strings.voiceRejection.transcribeAnyway),
        title: empty
          ? t(strings.voiceRejection.emptyRetryTitle)
          : t(strings.voiceRejection.retryTitle),
        onSelect: opts.onRetry,
        busy: retrying,
        primary: true,
        icon: RotateCcw,
      },
    ],
  };
}

// ── Recovery ──────────────────────────────────────────────────────────────

export function sessionRecoveryBanner(
  t: TFunction,
  state: { inProgress: boolean; total: number; recovered: number; adopted: number },
): BannerDescriptor {
  if (state.inProgress) {
    return {
      id: "session-recovery",
      testId: "session-recovery-banner",
      tone: "info",
      priority: BANNER_PRIORITY.sessionRecovery,
      icon: Loader2,
      spin: true,
      title:
        state.total > 0
          ? t(strings.sessionRecovery.recovering, {
              recovered: state.recovered,
              total: state.total,
            })
          : t(strings.sessionRecovery.recoveringUnknown),
    };
  }
  return {
    id: "session-recovery",
    testId: "session-recovery-banner",
    tone: "info",
    priority: BANNER_PRIORITY.sessionRecovery,
    icon: History,
    title: t(strings.sessionRecovery.recovered, { count: state.recovered + state.adopted }),
    actions: [
      {
        id: "view",
        label: t(strings.sessionRecovery.view),
        onSelect: () => { window.location.reload(); },
        primary: true,
        icon: RefreshCw,
      },
    ],
  };
}

export function crashRecoveryBanner(
  t: TFunction,
  count: number,
  onOpenArchive: () => void,
): BannerDescriptor {
  return {
    id: "crash-recovery",
    testId: "crash-recovery-notice",
    tone: "warning",
    priority: BANNER_PRIORITY.crashRecovery,
    icon: History,
    title: t(strings.recoverableSessions.heading, { count }),
    actions: [
      {
        id: "view-archive",
        label: t(strings.recoverableSessions.viewArchive),
        onSelect: onOpenArchive,
        primary: true,
      },
    ],
  };
}
