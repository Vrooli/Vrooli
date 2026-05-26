/**
 * Single source of truth for status semantics across the six surfaces.
 *
 * Every enum the API exposes (run status, restore status, per-target outcome,
 * destination usage, cap policy, source kind) maps here to a *tone* and a
 * stable *slug*. Components never inline status colors — they pass a tone to
 * `StatusChip` and translate the slug — so DESIGN.md's color roles
 * (green=done, amber=attention, red=failed/blocked, cyan=technical,
 * blue=primary) stay consistent everywhere and a single edit restyles all of
 * them.
 *
 * The slug doubles as an i18n key segment (`strings.status.<group>.<slug>`) and
 * a selector parameter, so it must stay stable.
 */
import {
  RunStatus,
  TargetOutcomeStatus,
  TriggerSource,
} from "@vrooli/proto-types/data-backup-manager/v1/runs/runs_pb";
import {
  RestoreStatus,
  RestoreMode,
} from "@vrooli/proto-types/data-backup-manager/v1/restores/restores_pb";
import {
  UsageState,
  CapPolicy,
  BackendKind,
} from "@vrooli/proto-types/data-backup-manager/v1/destinations/destinations_pb";
import { SourceKind } from "@vrooli/proto-types/data-backup-manager/v1/sources/sources_pb";
import { DriveClass } from "@vrooli/proto-types/data-backup-manager/v1/discovery/discovery_pb";

/** Semantic status tone. Maps to the DESIGN.md color roles. */
export type Tone = "neutral" | "primary" | "info" | "success" | "warning" | "danger";

export interface StatusMeta<Slug extends string = string> {
  tone: Tone;
  slug: Slug;
}

/** Tailwind text-color class per tone. The chip dot inherits via `bg-current`. */
export const TONE_TEXT_CLASS: Record<Tone, string> = {
  neutral: "text-app-muted-foreground",
  primary: "text-app-primary",
  info: "text-app-info",
  success: "text-app-success",
  warning: "text-app-warning",
  danger: "text-app-danger",
};

/** Tailwind solid-background class per tone (usage bars, fills). */
export const TONE_BG_CLASS: Record<Tone, string> = {
  neutral: "bg-app-muted-foreground",
  primary: "bg-app-primary",
  info: "bg-app-info",
  success: "bg-app-success",
  warning: "bg-app-warning",
  danger: "bg-app-danger",
};

// ---- Run status -------------------------------------------------------------

export type RunStatusSlug =
  | "unknown"
  | "pending"
  | "capturing"
  | "snapshotting"
  | "completed"
  | "partialFailed"
  | "failed";

export function runStatusMeta(status: RunStatus): StatusMeta<RunStatusSlug> {
  switch (status) {
    case RunStatus.PENDING:
      return { tone: "info", slug: "pending" };
    case RunStatus.CAPTURING:
      return { tone: "info", slug: "capturing" };
    case RunStatus.SNAPSHOTTING:
      return { tone: "info", slug: "snapshotting" };
    case RunStatus.COMPLETED:
      return { tone: "success", slug: "completed" };
    // Partial failure is attention, not a flat failure — some targets succeeded.
    case RunStatus.PARTIAL_FAILED:
      return { tone: "warning", slug: "partialFailed" };
    case RunStatus.FAILED:
      return { tone: "danger", slug: "failed" };
    default:
      return { tone: "neutral", slug: "unknown" };
  }
}

/** A run is in-flight (worth polling) until it reaches a terminal state. */
export function isRunInFlight(status: RunStatus): boolean {
  return (
    status === RunStatus.PENDING ||
    status === RunStatus.CAPTURING ||
    status === RunStatus.SNAPSHOTTING
  );
}

// ---- Per-target outcome -----------------------------------------------------

export type OutcomeSlug = "unknown" | "succeeded" | "failed" | "blocked";

export function outcomeMeta(status: TargetOutcomeStatus): StatusMeta<OutcomeSlug> {
  switch (status) {
    case TargetOutcomeStatus.SUCCEEDED:
      return { tone: "success", slug: "succeeded" };
    case TargetOutcomeStatus.FAILED:
      return { tone: "danger", slug: "failed" };
    // Cap-blocked is distinct: nothing was written, no data was lost or evicted.
    case TargetOutcomeStatus.BLOCKED:
      return { tone: "warning", slug: "blocked" };
    default:
      return { tone: "neutral", slug: "unknown" };
  }
}

export type TriggerSlug = "unknown" | "scheduler" | "manual";

export function triggerSlug(trigger: TriggerSource): TriggerSlug {
  switch (trigger) {
    case TriggerSource.SCHEDULER:
      return "scheduler";
    case TriggerSource.MANUAL:
      return "manual";
    default:
      return "unknown";
  }
}

// ---- Restore / verify status ------------------------------------------------

export type RestoreStatusSlug =
  | "unknown"
  | "requested"
  | "restoring"
  | "verifying"
  | "verified"
  | "restored"
  | "failed";

export function restoreStatusMeta(status: RestoreStatus): StatusMeta<RestoreStatusSlug> {
  switch (status) {
    case RestoreStatus.REQUESTED:
      return { tone: "info", slug: "requested" };
    case RestoreStatus.RESTORING:
      return { tone: "info", slug: "restoring" };
    case RestoreStatus.VERIFYING:
      return { tone: "info", slug: "verifying" };
    case RestoreStatus.VERIFIED:
      return { tone: "success", slug: "verified" };
    case RestoreStatus.RESTORED:
      return { tone: "success", slug: "restored" };
    case RestoreStatus.FAILED:
      return { tone: "danger", slug: "failed" };
    default:
      return { tone: "neutral", slug: "unknown" };
  }
}

/** A restore/verify is in-flight (worth polling) until terminal. */
export function isRestoreInFlight(status: RestoreStatus): boolean {
  return (
    status === RestoreStatus.REQUESTED ||
    status === RestoreStatus.RESTORING ||
    status === RestoreStatus.VERIFYING
  );
}

export type RestoreModeSlug = "unknown" | "restore" | "verify";

export function restoreModeSlug(mode: RestoreMode): RestoreModeSlug {
  switch (mode) {
    case RestoreMode.RESTORE:
      return "restore";
    case RestoreMode.VERIFY:
      return "verify";
    default:
      return "unknown";
  }
}

// ---- Destination usage + cap policy + backend -------------------------------

export type UsageSlug = "unknown" | "within" | "near" | "over";

export function usageMeta(state: UsageState): StatusMeta<UsageSlug> {
  switch (state) {
    case UsageState.WITHIN:
      return { tone: "success", slug: "within" };
    case UsageState.NEAR:
      return { tone: "warning", slug: "near" };
    case UsageState.OVER:
      return { tone: "danger", slug: "over" };
    default:
      return { tone: "neutral", slug: "unknown" };
  }
}

export type CapPolicySlug = "unknown" | "alertBlock" | "alertOnly";

export function capPolicySlug(policy: CapPolicy): CapPolicySlug {
  switch (policy) {
    case CapPolicy.ALERT_BLOCK:
      return "alertBlock";
    case CapPolicy.ALERT_ONLY:
      return "alertOnly";
    default:
      return "unknown";
  }
}

export type BackendSlug = "unknown" | "filesystem" | "s3";

export function backendSlug(kind: BackendKind): BackendSlug {
  switch (kind) {
    case BackendKind.FILESYSTEM:
      return "filesystem";
    case BackendKind.S3:
      return "s3";
    default:
      return "unknown";
  }
}

// ---- Source kind ------------------------------------------------------------

export type SourceKindSlug =
  | "unknown"
  | "filesystem"
  | "sqlite"
  | "postgres"
  | "redis"
  | "qdrant"
  | "objectStorage";

export function sourceKindSlug(kind: SourceKind): SourceKindSlug {
  switch (kind) {
    case SourceKind.FILESYSTEM:
      return "filesystem";
    case SourceKind.SQLITE:
      return "sqlite";
    case SourceKind.POSTGRES:
      return "postgres";
    case SourceKind.REDIS:
      return "redis";
    case SourceKind.QDRANT:
      return "qdrant";
    case SourceKind.OBJECT_STORAGE:
      return "objectStorage";
    default:
      return "unknown";
  }
}

/** The source kinds offered in the register-target form, in display order. */
export const SOURCE_KIND_OPTIONS: ReadonlyArray<{ kind: SourceKind; slug: SourceKindSlug }> = [
  { kind: SourceKind.FILESYSTEM, slug: "filesystem" },
  { kind: SourceKind.SQLITE, slug: "sqlite" },
  { kind: SourceKind.POSTGRES, slug: "postgres" },
  { kind: SourceKind.REDIS, slug: "redis" },
  { kind: SourceKind.QDRANT, slug: "qdrant" },
  { kind: SourceKind.OBJECT_STORAGE, slug: "objectStorage" },
];

// ---- Drive class (discovery destination suggestions) ------------------------

export type DriveClassSlug = "unknown" | "removable" | "fixed" | "network";

export function driveClassMeta(c: DriveClass): StatusMeta<DriveClassSlug> {
  switch (c) {
    // A plugged-in removable/external drive is the highlighted offsite-style
    // destination — primary tone draws the eye.
    case DriveClass.REMOVABLE:
      return { tone: "primary", slug: "removable" };
    case DriveClass.FIXED:
      return { tone: "info", slug: "fixed" };
    case DriveClass.NETWORK:
      return { tone: "info", slug: "network" };
    default:
      return { tone: "neutral", slug: "unknown" };
  }
}

// ---- Verified-restore posture (the product's spine) -------------------------

/** Default age past which a verified target is re-flagged as stale. */
export const STALE_VERIFY_MS = 30 * 24 * 60 * 60 * 1000;

export type VerifiedSlug = "verified" | "stale" | "unverified";

/**
 * Verified-restore posture for one target. `lastVerifiedAt === undefined` means
 * backed up but never proven restorable — the central distinction this product
 * surfaces. `stale` means it *was* verified but longer ago than `staleMs`.
 */
export function verifiedMeta(
  lastVerifiedAt: Date | undefined,
  now: Date = new Date(),
  staleMs: number = STALE_VERIFY_MS,
): StatusMeta<VerifiedSlug> {
  if (!lastVerifiedAt) return { tone: "warning", slug: "unverified" };
  if (now.getTime() - lastVerifiedAt.getTime() > staleMs) {
    return { tone: "warning", slug: "stale" };
  }
  return { tone: "success", slug: "verified" };
}
