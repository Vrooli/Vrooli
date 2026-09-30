import { strings } from "../../consts/strings";
import type { CandidateModel, HostSummary } from "../../api/models";

/**
 * Tone for a picker chip/affordance. Each maps to a paired token treatment so
 * meaning never rides on color alone: `positive` = clear-to-run, `info` = a
 * one-click step away, `caution` = needs attention / manual, `muted` = can't run
 * here (shown for transparency), `neutral` = informational.
 */
export type PickerTone = "positive" | "info" | "caution" | "muted" | "neutral";

/** The translation keys a picker chip can carry (fit / status / host / support). */
export type PickerStringKey =
  | (typeof strings.models.picker.fit)[keyof typeof strings.models.picker.fit]
  | (typeof strings.models.picker.state)[keyof typeof strings.models.picker.state]
  | (typeof strings.models.picker.host)[keyof typeof strings.models.picker.host]
  | (typeof strings.models.picker.support)[keyof typeof strings.models.picker.support];

/** A localized chip: a translation key + interpolation values + a tone. */
export interface PickerChip {
  key: PickerStringKey;
  values?: Record<string, string | number>;
  tone: PickerTone;
}

/** The action a row's primary button performs (or none). */
export type PickerActionKind =
  | "select"
  | "install-model"
  | "install-backend"
  | "enable"
  | "manual"
  | "none";

/** Everything the picker needs to render one candidate row. */
export interface CandidatePresentation {
  /** The host-aware "will it run here" badge. */
  fit: PickerChip;
  /** The ready-state status chip. */
  status: PickerChip;
  /**
   * The native/via-workflow support chip — present only for DERIVED candidates
   * (a model that serves this op through a derived technique, not a declared
   * native one). undefined for native candidates (no chip needed).
   */
  support?: PickerChip;
  /**
   * The derived-op quality caveat, raw text to surface in a banner on selection
   * ("" / undefined for native candidates).
   */
  caveat?: string;
  /** The primary action this row offers (may be "none"). */
  action: PickerActionKind;
  /** Whether the row itself is selectable (click-to-use). */
  selectable: boolean;
  /** Whether the row should read as de-emphasized (can't run here). */
  dimmed: boolean;
}

const READY_STATE = {
  ready: "ready",
  needsModelInstall: "needs_model_install",
  needsBackend: "needs_backend",
  needsBackendManual: "needs_backend_manual",
  needsBoth: "needs_both",
  disabled: "disabled",
  insufficient: "insufficient",
  unsupported: "unsupported",
  derivedUnproven: "derived_pipeline_unproven",
} as const;

const SUPPORT = { native: "native", derived: "derived" } as const;

/**
 * fitChip turns a candidate's host-aware fit_class into an affirmative badge. A
 * GPU-viable model reads "Runs on your GPU" (positive) — the fix for the old
 * static "Needs a GPU" chip that showed a warning even on a GPU host. A
 * CPU-capable model on a GPU host that can't currently fit reads as a CPU
 * fallback, not a failure.
 */
export function fitChip(candidate: CandidateModel, host: HostSummary | undefined): PickerChip {
  const fit = candidate.fit;
  switch (fit?.fitClass) {
    case "gpu":
      return { key: strings.models.picker.fit.gpu, tone: "positive" };
    case "cpu":
      return host?.hasGpu
        ? { key: strings.models.picker.fit.cpuFallback, tone: "neutral" }
        : { key: strings.models.picker.fit.cpu, tone: "positive" };
    case "insufficient_vram":
      return {
        key: strings.models.picker.fit.insufficientVram,
        values: { gb: fit.vramShortfallGb || 1 },
        tone: "caution",
      };
    case "no_gpu":
      return { key: strings.models.picker.fit.noGpu, tone: "caution" };
    case "unsupported_os":
      return {
        key: strings.models.picker.fit.unsupportedOs,
        values: { os: host?.os ?? "", arch: host?.arch ?? "" },
        tone: "muted",
      };
    default:
      return { key: strings.models.picker.fit.cpu, tone: "neutral" };
  }
}

const STATUS_BY_STATE: Record<string, PickerChip> = {
  [READY_STATE.ready]: { key: strings.models.picker.state.ready, tone: "positive" },
  [READY_STATE.needsModelInstall]: {
    key: strings.models.picker.state.needsModel,
    tone: "info",
  },
  [READY_STATE.needsBackend]: { key: strings.models.picker.state.needsBackend, tone: "info" },
  [READY_STATE.needsBackendManual]: {
    key: strings.models.picker.state.needsBackendManual,
    tone: "caution",
  },
  [READY_STATE.needsBoth]: { key: strings.models.picker.state.needsBoth, tone: "info" },
  [READY_STATE.disabled]: { key: strings.models.picker.state.disabled, tone: "neutral" },
  [READY_STATE.insufficient]: { key: strings.models.picker.state.insufficient, tone: "muted" },
  [READY_STATE.unsupported]: { key: strings.models.picker.state.unsupported, tone: "muted" },
  [READY_STATE.derivedUnproven]: {
    key: strings.models.picker.state.derivedUnproven,
    tone: "caution",
  },
};

/** statusChip resolves the ready-state status chip (falls back to neutral). */
export function statusChip(candidate: CandidateModel): PickerChip {
  return (
    STATUS_BY_STATE[candidate.readyState] ?? {
      key: strings.models.picker.state.ready,
      tone: "neutral",
    }
  );
}

/**
 * supportChip distinguishes a candidate that serves the op NATIVELY (declared)
 * from one that serves it via a DERIVED technique. It returns a chip only for
 * derived candidates (a "Via workflow" caution badge) — native candidates need
 * no badge and return undefined, so the common case stays visually quiet.
 */
export function supportChip(candidate: CandidateModel): PickerChip | undefined {
  if (candidate.support !== SUPPORT.derived) {
    return undefined;
  }
  return { key: strings.models.picker.support.viaWorkflow, tone: "caution" };
}

/**
 * actionFor decides the row's primary affordance from its ready_state. A
 * `needs_both` row leads with the model download (the backend install is offered
 * as the secondary affordance the component renders separately).
 */
export function actionFor(candidate: CandidateModel): PickerActionKind {
  switch (candidate.readyState) {
    case READY_STATE.ready:
      return "select";
    case READY_STATE.needsModelInstall:
    case READY_STATE.needsBoth:
      return "install-model";
    case READY_STATE.needsBackend:
      return "install-backend";
    case READY_STATE.needsBackendManual:
      return "manual";
    case READY_STATE.disabled:
      return "enable";
    default:
      return "none";
  }
}

/** present composes the full per-row presentation for a candidate. */
export function present(
  candidate: CandidateModel,
  host: HostSummary | undefined,
): CandidatePresentation {
  const action = actionFor(candidate);
  const dimmed =
    candidate.readyState === READY_STATE.insufficient ||
    candidate.readyState === READY_STATE.unsupported ||
    candidate.readyState === READY_STATE.derivedUnproven;
  return {
    fit: fitChip(candidate, host),
    status: statusChip(candidate),
    support: supportChip(candidate),
    caveat: candidate.caveat || undefined,
    action,
    selectable: candidate.readyState === READY_STATE.ready,
    dimmed,
  };
}

/** Whether a candidate also needs its backend engine installed (one-click). */
export function alsoNeedsBackend(candidate: CandidateModel): boolean {
  return (
    candidate.readyState === READY_STATE.needsBoth &&
    candidate.backend?.installTier === "auto"
  );
}

/** hostSummaryChip formats the host hardware line for the picker header. */
export function hostSummaryLine(host: HostSummary | undefined): PickerChip {
  if (!host || !host.hasGpu) {
    return { key: strings.models.picker.host.noGpu, tone: "neutral" };
  }
  if (!host.vramKnown) {
    return {
      key: strings.models.picker.host.gpuUnknown,
      values: { name: host.gpuName },
      tone: "neutral",
    };
  }
  return {
    key: strings.models.picker.host.gpu,
    values: { name: host.gpuName, total: host.vramTotalGb, free: host.vramFreeGb },
    tone: "neutral",
  };
}

/** Tone → paired chip token classes (background + text), never color-only. */
export const PICKER_TONE_CLASS: Record<PickerTone, string> = {
  positive: "bg-app-success/10 text-app-success",
  info: "bg-app-info/10 text-app-info",
  caution: "bg-app-warning/10 text-app-warning",
  muted: "bg-app-surface-muted text-app-muted-foreground",
  neutral: "bg-app-surface-muted text-app-muted-foreground",
};
