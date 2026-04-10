/**
 * Proto enum adapters for backward compatibility.
 *
 * These adapters convert between legacy string-based types and proto enums,
 * allowing gradual migration of the codebase.
 */

import {
  Platform,
  StageName,
  StageStatus,
  BuildStatus,
  UploadStatus,
  DeploymentMode,
  Framework,
  TemplateType,
} from "@vrooli/proto-types/scenario-to-desktop/v1/base/shared_pb";
import {
  PlatformBuildStatus,
  SmokeTestStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/build_pb";
import {
  PreflightStatus,
  CheckStatus,
  SecretClass,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/preflight_pb";

// ==================== Platform ====================

/** Legacy platform string type */
export type LegacyPlatform = "win" | "mac" | "linux";

/** Convert legacy platform string to proto enum */
export const platformFromString = (platform: LegacyPlatform): Platform => {
  switch (platform) {
    case "win":
      return Platform.WIN;
    case "mac":
      return Platform.MAC;
    case "linux":
      return Platform.LINUX;
    default:
      return Platform.UNSPECIFIED;
  }
};

/** Convert proto enum to legacy platform string */
export const platformToString = (platform: Platform): LegacyPlatform | null => {
  switch (platform) {
    case Platform.WIN:
      return "win";
    case Platform.MAC:
      return "mac";
    case Platform.LINUX:
      return "linux";
    default:
      return null;
  }
};

// ==================== StageName ====================

/** Legacy stage name string type */
export type LegacyStageName =
  | "preflight"
  | "generate"
  | "build"
  | "bundle"
  | "smoke_test"
  | "smoketest"
  | "deploy"
  | "distribution";

/** Convert legacy stage name string to proto enum */
export const stageNameFromString = (name: LegacyStageName): StageName => {
  switch (name) {
    case "preflight":
      return StageName.PREFLIGHT;
    case "generate":
      return StageName.GENERATE;
    case "build":
      return StageName.BUILD;
    case "bundle":
      return StageName.BUNDLE;
    case "smoke_test":
    case "smoketest":
      return StageName.SMOKE_TEST;
    case "deploy":
    case "distribution":
      return StageName.DISTRIBUTION;
    default:
      return StageName.UNSPECIFIED;
  }
};

/** Convert proto enum to legacy stage name string */
export const stageNameToString = (
  stage: StageName
): LegacyStageName | null => {
  switch (stage) {
    case StageName.PREFLIGHT:
      return "preflight";
    case StageName.GENERATE:
      return "generate";
    case StageName.BUILD:
      return "build";
    case StageName.BUNDLE:
      return "bundle";
    case StageName.SMOKE_TEST:
      return "smoketest";
    case StageName.DISTRIBUTION:
      return "deploy";
    default:
      return null;
  }
};

// ==================== StageStatus ====================

/** Legacy stage status string type */
export type LegacyStageStatus =
  | "pending"
  | "running"
  | "completed"
  | "failed"
  | "skipped"
  | "cancelled";

/** Convert legacy stage status string to proto enum */
export const stageStatusFromString = (
  status: LegacyStageStatus
): StageStatus => {
  switch (status) {
    case "pending":
      return StageStatus.PENDING;
    case "running":
      return StageStatus.RUNNING;
    case "completed":
      return StageStatus.COMPLETED;
    case "failed":
      return StageStatus.FAILED;
    case "skipped":
      return StageStatus.SKIPPED;
    case "cancelled":
      return StageStatus.CANCELLED;
    default:
      return StageStatus.UNSPECIFIED;
  }
};

/** Convert proto enum to legacy stage status string */
export const stageStatusToString = (
  status: StageStatus
): LegacyStageStatus | null => {
  switch (status) {
    case StageStatus.PENDING:
      return "pending";
    case StageStatus.RUNNING:
      return "running";
    case StageStatus.COMPLETED:
      return "completed";
    case StageStatus.FAILED:
      return "failed";
    case StageStatus.SKIPPED:
      return "skipped";
    case StageStatus.CANCELLED:
      return "cancelled";
    default:
      return null;
  }
};

// ==================== BuildStatus ====================

/** Legacy build status string type */
export type LegacyBuildStatus = "building" | "ready" | "partial" | "failed";

/** Convert legacy build status string to proto enum */
export const buildStatusFromString = (status: LegacyBuildStatus): BuildStatus => {
  switch (status) {
    case "building":
      return BuildStatus.BUILDING;
    case "ready":
      return BuildStatus.READY;
    case "partial":
      return BuildStatus.PARTIAL;
    case "failed":
      return BuildStatus.FAILED;
    default:
      return BuildStatus.UNSPECIFIED;
  }
};

/** Convert proto enum to legacy build status string */
export const buildStatusToString = (
  status: BuildStatus
): LegacyBuildStatus | null => {
  switch (status) {
    case BuildStatus.BUILDING:
      return "building";
    case BuildStatus.READY:
      return "ready";
    case BuildStatus.PARTIAL:
      return "partial";
    case BuildStatus.FAILED:
      return "failed";
    default:
      return null;
  }
};

// ==================== PlatformBuildStatus ====================

/** Legacy platform build status string type */
export type LegacyPlatformBuildStatus =
  | "pending"
  | "building"
  | "ready"
  | "failed"
  | "skipped";

/** Convert legacy platform build status string to proto enum */
export const platformBuildStatusFromString = (
  status: LegacyPlatformBuildStatus
): PlatformBuildStatus => {
  switch (status) {
    case "pending":
      // Note: Proto doesn't have PENDING, map to UNSPECIFIED
      return PlatformBuildStatus.UNSPECIFIED;
    case "building":
      return PlatformBuildStatus.BUILDING;
    case "ready":
      return PlatformBuildStatus.READY;
    case "failed":
      return PlatformBuildStatus.FAILED;
    case "skipped":
      return PlatformBuildStatus.SKIPPED;
    default:
      return PlatformBuildStatus.UNSPECIFIED;
  }
};

/** Convert proto enum to legacy platform build status string */
export const platformBuildStatusToString = (
  status: PlatformBuildStatus
): LegacyPlatformBuildStatus | null => {
  switch (status) {
    case PlatformBuildStatus.BUILDING:
      return "building";
    case PlatformBuildStatus.READY:
      return "ready";
    case PlatformBuildStatus.FAILED:
      return "failed";
    case PlatformBuildStatus.SKIPPED:
      return "skipped";
    default:
      return null;
  }
};

// ==================== SmokeTestStatus ====================

/** Legacy smoke test status string type */
export type LegacySmokeTestStatus = "running" | "passed" | "failed";

/** Convert legacy smoke test status string to proto enum */
export const smokeTestStatusFromString = (
  status: LegacySmokeTestStatus
): SmokeTestStatus => {
  switch (status) {
    case "running":
      return SmokeTestStatus.RUNNING;
    case "passed":
      return SmokeTestStatus.PASSED;
    case "failed":
      return SmokeTestStatus.FAILED;
    default:
      return SmokeTestStatus.UNSPECIFIED;
  }
};

/** Convert proto enum to legacy smoke test status string */
export const smokeTestStatusToString = (
  status: SmokeTestStatus
): LegacySmokeTestStatus | null => {
  switch (status) {
    case SmokeTestStatus.RUNNING:
      return "running";
    case SmokeTestStatus.PASSED:
      return "passed";
    case SmokeTestStatus.FAILED:
      return "failed";
    default:
      return null;
  }
};

// ==================== DeploymentMode ====================

/** Legacy deployment mode string type */
export type LegacyDeploymentMode = "proxy" | "bundled";

/** Convert legacy deployment mode string to proto enum */
export const deploymentModeFromString = (
  mode: LegacyDeploymentMode
): DeploymentMode => {
  switch (mode) {
    case "proxy":
      return DeploymentMode.PROXY;
    case "bundled":
      return DeploymentMode.BUNDLED;
    default:
      return DeploymentMode.UNSPECIFIED;
  }
};

/** Convert proto enum to legacy deployment mode string */
export const deploymentModeToString = (
  mode: DeploymentMode
): LegacyDeploymentMode | null => {
  switch (mode) {
    case DeploymentMode.PROXY:
      return "proxy";
    case DeploymentMode.BUNDLED:
      return "bundled";
    default:
      return null;
  }
};

// ==================== Framework ====================

/** Legacy framework string type */
export type LegacyFramework = "electron" | "tauri" | "neutralino";

/** Convert legacy framework string to proto enum */
export const frameworkFromString = (framework: LegacyFramework): Framework => {
  switch (framework) {
    case "electron":
      return Framework.ELECTRON;
    case "tauri":
      return Framework.TAURI;
    case "neutralino":
      return Framework.NEUTRALINO;
    default:
      return Framework.UNSPECIFIED;
  }
};

/** Convert proto enum to legacy framework string */
export const frameworkToString = (
  framework: Framework
): LegacyFramework | null => {
  switch (framework) {
    case Framework.ELECTRON:
      return "electron";
    case Framework.TAURI:
      return "tauri";
    case Framework.NEUTRALINO:
      return "neutralino";
    default:
      return null;
  }
};

// ==================== TemplateType ====================

/** Legacy template type string type */
export type LegacyTemplateType = "basic" | "advanced" | "multi-window" | "kiosk";

/** Convert legacy template type string to proto enum */
export const templateTypeFromString = (
  type: LegacyTemplateType
): TemplateType => {
  switch (type) {
    case "basic":
      return TemplateType.BASIC;
    case "advanced":
      return TemplateType.ADVANCED;
    case "multi-window":
      return TemplateType.MULTI_WINDOW;
    case "kiosk":
      return TemplateType.KIOSK;
    default:
      return TemplateType.UNSPECIFIED;
  }
};

/** Convert proto enum to legacy template type string */
export const templateTypeToString = (
  type: TemplateType
): LegacyTemplateType | null => {
  switch (type) {
    case TemplateType.BASIC:
      return "basic";
    case TemplateType.ADVANCED:
      return "advanced";
    case TemplateType.MULTI_WINDOW:
      return "multi-window";
    case TemplateType.KIOSK:
      return "kiosk";
    default:
      return null;
  }
};

// ==================== UploadStatus ====================

/** Legacy upload status string type */
export type LegacyUploadStatus =
  | "pending"
  | "uploading"
  | "completed"
  | "failed";

/** Convert legacy upload status string to proto enum */
export const uploadStatusFromString = (
  status: LegacyUploadStatus
): UploadStatus => {
  switch (status) {
    case "pending":
      return UploadStatus.PENDING;
    case "uploading":
      return UploadStatus.UPLOADING;
    case "completed":
      return UploadStatus.COMPLETED;
    case "failed":
      return UploadStatus.FAILED;
    default:
      return UploadStatus.UNSPECIFIED;
  }
};

/** Convert proto enum to legacy upload status string */
export const uploadStatusToString = (
  status: UploadStatus
): LegacyUploadStatus | null => {
  switch (status) {
    case UploadStatus.PENDING:
      return "pending";
    case UploadStatus.UPLOADING:
      return "uploading";
    case UploadStatus.COMPLETED:
      return "completed";
    case UploadStatus.FAILED:
      return "failed";
    default:
      return null;
  }
};

// ==================== PreflightStatus ====================

/** Legacy preflight status string type */
export type LegacyPreflightStatus =
  | "running"
  | "passed"
  | "failed"
  | "warnings";

/** Convert legacy preflight status string to proto enum */
export const preflightStatusFromString = (
  status: LegacyPreflightStatus
): PreflightStatus => {
  switch (status) {
    case "running":
      return PreflightStatus.RUNNING;
    case "passed":
      return PreflightStatus.PASSED;
    case "failed":
      return PreflightStatus.FAILED;
    case "warnings":
      return PreflightStatus.WARNINGS;
    default:
      return PreflightStatus.UNSPECIFIED;
  }
};

/** Convert proto enum to legacy preflight status string */
export const preflightStatusToString = (
  status: PreflightStatus
): LegacyPreflightStatus | null => {
  switch (status) {
    case PreflightStatus.RUNNING:
      return "running";
    case PreflightStatus.PASSED:
      return "passed";
    case PreflightStatus.FAILED:
      return "failed";
    case PreflightStatus.WARNINGS:
      return "warnings";
    default:
      return null;
  }
};

// ==================== CheckStatus ====================

/** Legacy check status string type */
export type LegacyCheckStatus =
  | "pending"
  | "running"
  | "pass"
  | "fail"
  | "warning"
  | "skipped";

/** Convert legacy check status string to proto enum */
export const checkStatusFromString = (status: LegacyCheckStatus): CheckStatus => {
  switch (status) {
    case "pending":
      return CheckStatus.PENDING;
    case "running":
      return CheckStatus.RUNNING;
    case "pass":
      return CheckStatus.PASSED;
    case "fail":
      return CheckStatus.FAILED;
    case "skipped":
      return CheckStatus.SKIPPED;
    // Note: Proto doesn't have WARNING state
    case "warning":
      return CheckStatus.PASSED;
    default:
      return CheckStatus.UNSPECIFIED;
  }
};

/** Convert proto enum to legacy check status string */
export const checkStatusToString = (
  status: CheckStatus
): LegacyCheckStatus | null => {
  switch (status) {
    case CheckStatus.PENDING:
      return "pending";
    case CheckStatus.RUNNING:
      return "running";
    case CheckStatus.PASSED:
      return "pass";
    case CheckStatus.FAILED:
      return "fail";
    case CheckStatus.SKIPPED:
      return "skipped";
    default:
      return null;
  }
};

// ==================== SecretClass ====================

/** Legacy secret class string type */
export type LegacySecretClass =
  | "api_key"
  | "password"
  | "token"
  | "connection_string"
  | "certificate"
  | "generic";

/** Convert legacy secret class string to proto enum */
export const secretClassFromString = (cls: LegacySecretClass): SecretClass => {
  switch (cls) {
    case "api_key":
      return SecretClass.API_KEY;
    case "password":
      return SecretClass.PASSWORD;
    case "token":
      return SecretClass.TOKEN;
    case "connection_string":
      return SecretClass.CONNECTION_STRING;
    case "certificate":
      return SecretClass.CERTIFICATE;
    case "generic":
      return SecretClass.GENERIC;
    default:
      return SecretClass.UNSPECIFIED;
  }
};

/** Convert proto enum to legacy secret class string */
export const secretClassToString = (
  cls: SecretClass
): LegacySecretClass | null => {
  switch (cls) {
    case SecretClass.API_KEY:
      return "api_key";
    case SecretClass.PASSWORD:
      return "password";
    case SecretClass.TOKEN:
      return "token";
    case SecretClass.CONNECTION_STRING:
      return "connection_string";
    case SecretClass.CERTIFICATE:
      return "certificate";
    case SecretClass.GENERIC:
      return "generic";
    default:
      return null;
  }
};
