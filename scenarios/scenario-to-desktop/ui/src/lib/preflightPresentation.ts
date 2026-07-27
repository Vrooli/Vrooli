import type { PreflightResponse } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/preflight_results_pb";
import {
  CheckStatus,
  PreflightCheckStep,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/preflight_results_pb";

export type PreflightPresentation = {
  validation?: {
    valid: boolean;
    errors: Array<{
      code: string;
      service?: string;
      path?: string;
      message: string;
    }>;
    warnings: Array<{ code: string; message: string }>;
    missing_binaries: Array<{
      service_id: string;
      platform: string;
      path: string;
    }>;
    missing_assets: Array<{ service_id: string; path: string }>;
    invalid_checksums: Array<{
      service_id: string;
      path: string;
      expected?: string;
      actual?: string;
    }>;
  };
  ready?: {
    ready: boolean;
    details: Record<string, ServiceReadinessPresentation>;
    snapshot_at?: string;
    waited_seconds: number;
  };
  ports: Record<string, Record<string, number>>;
  telemetry?: { path: string; upload_url?: string };
  runtime?: RuntimePresentation;
  fingerprints: Array<{
    service_id: string;
    platform?: string;
    binary_path?: string;
    binary_resolved_path?: string;
    binary_sha256?: string;
    binary_size_bytes?: number;
    binary_mtime?: string;
    error?: string;
  }>;
  logTails: Array<{
    service_id: string;
    lines: number;
    content?: string;
    error?: string;
  }>;
  checks: Array<{
    id: string;
    step: string;
    name: string;
    status: "pass" | "fail" | "warning" | "skipped";
    detail?: string;
  }>;
  errors: string[];
};
export type ServiceReadinessPresentation = {
  ready: boolean;
  skipped?: boolean;
  message?: string;
  exit_code?: number;
  started_at?: string;
  ready_at?: string;
  updated_at?: string;
};
export type RuntimePresentation = {
  instance_id: string;
  started_at?: string;
  dry_run: boolean;
  manifest_hash?: string;
  app_name?: string;
  app_version?: string;
  ipc_host?: string;
  ipc_port?: number;
  runtime_version?: string;
  build_version?: string;
  bundle_root?: string;
  app_data_dir?: string;
};

const steps: Record<PreflightCheckStep, string> = {
  [PreflightCheckStep.UNSPECIFIED]: "",
  [PreflightCheckStep.VALIDATION]: "validation",
  [PreflightCheckStep.SECRETS]: "secrets",
  [PreflightCheckStep.RUNTIME]: "runtime",
  [PreflightCheckStep.SERVICES]: "services",
  [PreflightCheckStep.DIAGNOSTICS]: "diagnostics",
};
const statuses: Record<CheckStatus, "pass" | "fail" | "warning" | "skipped"> = {
  [CheckStatus.UNSPECIFIED]: "warning",
  [CheckStatus.PENDING]: "warning",
  [CheckStatus.RUNNING]: "warning",
  [CheckStatus.PASSED]: "pass",
  [CheckStatus.FAILED]: "fail",
  [CheckStatus.SKIPPED]: "skipped",
};

export function presentPreflight(
  result: PreflightResponse,
): PreflightPresentation {
  const ports: Record<string, Record<string, number>> = {};
  for (const item of result.ports)
    (ports[item.serviceId] ??= {})[item.name] = item.port;
  return {
    validation: result.validation && {
      valid: result.validation.valid,
      errors: result.validation.errors.map(
        ({ code, service, path, message }) => ({
          code,
          service,
          path,
          message,
        }),
      ),
      warnings: result.validation.warnings.map(({ code, message }) => ({
        code,
        message,
      })),
      missing_binaries: result.validation.missingBinaries.map(
        ({ serviceId, platform, path }) => ({
          service_id: serviceId,
          platform,
          path,
        }),
      ),
      missing_assets: result.validation.missingAssets.map(
        ({ serviceId, path }) => ({ service_id: serviceId, path }),
      ),
      invalid_checksums: result.validation.invalidChecksums.map(
        ({ serviceId, path, expected, actual }) => ({
          service_id: serviceId,
          path,
          expected,
          actual,
        }),
      ),
    },
    ready: result.ready && {
      ready: result.ready.ready,
      details: Object.fromEntries(
        result.ready.details.map((item) => [
          item.serviceId,
          {
            ready: item.ready,
            skipped: item.skipped,
            message: item.message,
            exit_code: item.exitCode,
            started_at: timestamp(item.startedAt),
            ready_at: timestamp(item.readyAt),
            updated_at: timestamp(item.updatedAt),
          },
        ]),
      ),
      snapshot_at: timestamp(result.ready.snapshotAt),
      waited_seconds: result.ready.waitedSeconds,
    },
    ports,
    telemetry: result.telemetry && {
      path: result.telemetry.path,
      upload_url: result.telemetry.uploadUrl,
    },
    runtime: result.runtime && {
      instance_id: result.runtime.instanceId,
      started_at: timestamp(result.runtime.startedAt),
      dry_run: result.runtime.dryRun,
      manifest_hash: result.runtime.manifestHash,
      app_name: result.runtime.appName,
      app_version: result.runtime.appVersion,
      ipc_host: result.runtime.ipcHost,
      ipc_port: result.runtime.ipcPort,
      runtime_version: result.runtime.runtimeVersion,
      build_version: result.runtime.buildVersion,
      bundle_root: result.runtime.bundleRoot,
      app_data_dir: result.runtime.appDataDir,
    },
    fingerprints: result.serviceFingerprints.map((item) => ({
      service_id: item.serviceId,
      platform: item.platform,
      binary_path: item.binaryPath,
      binary_resolved_path: item.binaryResolvedPath,
      binary_sha256: item.binarySha256,
      binary_size_bytes:
        item.binarySizeBytes === undefined
          ? undefined
          : Number(item.binarySizeBytes),
      binary_mtime: timestamp(item.binaryMtime),
      error: item.error,
    })),
    logTails: result.logTails.map((item) => ({
      service_id: item.serviceId,
      lines: item.lines,
      content: item.content,
      error: item.error,
    })),
    checks: result.checks.map((item) => ({
      id: item.id,
      step: steps[item.step],
      name: item.name,
      status: statuses[item.status],
      detail: item.detail,
    })),
    errors: result.errors.map((item) => item.message),
  };
}
function timestamp(
  value: { seconds: bigint; nanos: number } | undefined,
): string | undefined {
  return !value || value.seconds === 0n
    ? undefined
    : new Date(
        Number(value.seconds) * 1000 + Math.floor(value.nanos / 1_000_000),
      ).toISOString();
}
