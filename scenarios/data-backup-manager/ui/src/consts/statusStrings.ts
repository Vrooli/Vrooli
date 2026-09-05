/**
 * Slug → i18n-key maps for every status enum.
 *
 * Status labels are looked up dynamically (`RUN_STATUS_STRINGS[meta.slug]`),
 * but the `strings/no-unused-keys` lint rule only sees *static* `strings.x.y`
 * accessors. Spelling out every leaf here is therefore doing double duty: it
 * gives components a dynamic lookup table AND it is the static reference that
 * marks each `status.*` catalog key as used. Keep these exhaustive — a missing
 * slug is a TypeScript error (Record over the slug union), and a missing
 * catalog key is a lint error.
 */
import type {
  BackendSlug,
  CapPolicySlug,
  DriveClassSlug,
  OutcomeSlug,
  RestoreStatusSlug,
  RunStatusSlug,
  SourceKindSlug,
  TriggerSlug,
  UsageSlug,
  VerifiedSlug,
} from "../lib/status";
import { strings, type StringKey } from "./strings";

export const RUN_STATUS_STRINGS = {
  unknown: strings.status.run.unknown,
  pending: strings.status.run.pending,
  capturing: strings.status.run.capturing,
  snapshotting: strings.status.run.snapshotting,
  completed: strings.status.run.completed,
  partialFailed: strings.status.run.partialFailed,
  failed: strings.status.run.failed,
} satisfies Record<RunStatusSlug, StringKey>;

export const OUTCOME_STRINGS = {
  unknown: strings.status.outcome.unknown,
  succeeded: strings.status.outcome.succeeded,
  failed: strings.status.outcome.failed,
  blocked: strings.status.outcome.blocked,
} satisfies Record<OutcomeSlug, StringKey>;

export const RESTORE_STATUS_STRINGS = {
  unknown: strings.status.restore.unknown,
  requested: strings.status.restore.requested,
  restoring: strings.status.restore.restoring,
  verifying: strings.status.restore.verifying,
  verified: strings.status.restore.verified,
  restored: strings.status.restore.restored,
  failed: strings.status.restore.failed,
} satisfies Record<RestoreStatusSlug, StringKey>;

export const USAGE_STRINGS = {
  unknown: strings.status.usage.unknown,
  within: strings.status.usage.within,
  near: strings.status.usage.near,
  over: strings.status.usage.over,
} satisfies Record<UsageSlug, StringKey>;

export const CAP_POLICY_STRINGS = {
  unknown: strings.status.capPolicy.unknown,
  alertBlock: strings.status.capPolicy.alertBlock,
  alertOnly: strings.status.capPolicy.alertOnly,
} satisfies Record<CapPolicySlug, StringKey>;

export const BACKEND_STRINGS = {
  unknown: strings.status.backend.unknown,
  filesystem: strings.status.backend.filesystem,
  s3: strings.status.backend.s3,
} satisfies Record<BackendSlug, StringKey>;

export const SOURCE_KIND_STRINGS = {
  unknown: strings.status.sourceKind.unknown,
  filesystem: strings.status.sourceKind.filesystem,
  sqlite: strings.status.sourceKind.sqlite,
  postgres: strings.status.sourceKind.postgres,
  redis: strings.status.sourceKind.redis,
  qdrant: strings.status.sourceKind.qdrant,
  objectStorage: strings.status.sourceKind.objectStorage,
} satisfies Record<SourceKindSlug, StringKey>;

export const VERIFIED_STRINGS = {
  verified: strings.status.verified.verified,
  stale: strings.status.verified.stale,
  unverified: strings.status.verified.unverified,
} satisfies Record<VerifiedSlug, StringKey>;

export const TRIGGER_STRINGS = {
  unknown: strings.status.trigger.unknown,
  scheduler: strings.status.trigger.scheduler,
  manual: strings.status.trigger.manual,
} satisfies Record<TriggerSlug, StringKey>;

export const DRIVE_CLASS_STRINGS = {
  unknown: strings.status.driveClass.unknown,
  removable: strings.status.driveClass.removable,
  fixed: strings.status.driveClass.fixed,
  network: strings.status.driveClass.network,
} satisfies Record<DriveClassSlug, StringKey>;
