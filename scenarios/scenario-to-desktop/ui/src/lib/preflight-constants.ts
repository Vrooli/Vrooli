/**
 * Constants for the preflight validation UI.
 * Contains style mappings, guidance text, and configuration.
 */

import type { BundlePreflightCheck } from "./api";

// ============================================================================
// Types
// ============================================================================

export type PreflightStepState = "pending" | "testing" | "pass" | "fail" | "warning" | "skipped";

export type PreflightStepStatus = {
  state: PreflightStepState;
  label: string;
};

export type PreflightIssueGuidance = {
  title: string;
  meaning: string;
  remediation: string[];
};

export type CoverageCell = {
  label: string;
  active: boolean;
};

export type CoverageRow = {
  id: string;
  label: string;
  note: string;
  cells: CoverageCell[];
};

export type HealthPeekState = {
  open: boolean;
  status: "idle" | "loading" | "ready" | "error";
  body?: string;
  error?: string;
  statusCode?: number;
  url?: string;
  contentType?: string;
  truncated?: boolean;
};

// ============================================================================
// Issue Guidance
// ============================================================================

/**
 * Detailed guidance for each type of preflight validation issue.
 * Maps issue codes to user-friendly titles, explanations, and remediation steps.
 */
export const PREFLIGHT_ISSUE_GUIDANCE: Record<string, PreflightIssueGuidance> = {
  manifest_invalid: {
    title: "Bundle manifest is invalid",
    meaning: "The manifest structure does not pass schema or platform validation.",
    remediation: [
      "Re-export the bundle from deployment-manager for the current OS/arch.",
      "Confirm the manifest points to assets that exist inside the bundle root."
    ]
  },
  binary_not_defined: {
    title: "Binary missing in manifest",
    meaning: "The service has no binary entry for this platform in bundle.json.",
    remediation: [
      "Ensure the service is built for this OS/arch, then re-export the bundle.",
      "Check deployment-manager export settings include service binaries."
    ]
  },
  binary_missing: {
    title: "Binary file not found",
    meaning: "The manifest references a binary path that doesn't exist in the bundle root.",
    remediation: [
      "Rebuild the service binary and re-export the bundle.",
      "Verify the binary path in bundle.json matches the staged file path."
    ]
  },
  binary_is_directory: {
    title: "Binary path is a directory",
    meaning: "The manifest points at a directory, but an executable file is required.",
    remediation: [
      "Update the manifest to point to the executable file.",
      "Re-export the bundle after rebuilding the service."
    ]
  },
  asset_missing: {
    title: "Asset file not found",
    meaning: "The manifest references an asset path that doesn't exist in the bundle root.",
    remediation: [
      "Rebuild the UI/assets and re-export the bundle.",
      "Confirm the asset path in bundle.json matches the staged file path."
    ]
  },
  asset_checksum_mismatch: {
    title: "Asset checksum mismatch",
    meaning: "The file exists but its checksum differs from the manifest entry.",
    remediation: [
      "Re-export the bundle so checksums match the staged files.",
      "Verify no post-export steps modify the bundle files."
    ]
  },
  asset_size_suspicious: {
    title: "Asset size is suspiciously small",
    meaning: "The asset is much smaller than expected, which often indicates a failed build.",
    remediation: [
      "Rebuild the asset and re-export the bundle.",
      "Verify the build output is complete (not a stub file)."
    ]
  },
  asset_size_warning: {
    title: "Asset size warning",
    meaning: "The asset is larger than expected but within the slack budget.",
    remediation: [
      "Re-export the bundle to update size metadata if this is expected.",
      "Review build artifacts for unexpected growth."
    ]
  }
};

// ============================================================================
// Step Styles
// ============================================================================

/**
 * Tailwind classes for preflight step containers based on state.
 */
export const PREFLIGHT_STEP_STYLES: Record<PreflightStepState, string> = {
  pending: "border-slate-800/70 bg-slate-950/70 text-slate-200",
  testing: "border-blue-800/70 bg-blue-950/60 text-blue-200",
  pass: "border-emerald-800/70 bg-emerald-950/40 text-emerald-200",
  fail: "border-red-800/70 bg-red-950/60 text-red-200",
  warning: "border-amber-800/70 bg-amber-950/60 text-amber-200",
  skipped: "border-slate-800/70 bg-slate-950/60 text-slate-300"
};

/**
 * Tailwind classes for preflight step number circles based on state.
 */
export const PREFLIGHT_STEP_CIRCLE_STYLES: Record<PreflightStepState, string> = {
  pending: "border-slate-800/70 bg-slate-950 text-slate-200",
  testing: "border-blue-800/70 bg-blue-950/60 text-blue-200",
  pass: "border-emerald-800/70 bg-emerald-950/40 text-emerald-200",
  fail: "border-red-800/70 bg-red-950/60 text-red-200",
  warning: "border-amber-800/70 bg-amber-950/60 text-amber-200",
  skipped: "border-slate-800/70 bg-slate-950/60 text-slate-300"
};

// ============================================================================
// Check Styles
// ============================================================================

/**
 * Tailwind classes for individual preflight check badges based on status.
 */
export const PREFLIGHT_CHECK_STYLES: Record<BundlePreflightCheck["status"], string> = {
  pass: "border-emerald-800/70 text-emerald-200",
  fail: "border-red-800/70 text-red-200",
  warning: "border-amber-800/70 text-amber-200",
  skipped: "border-slate-800/70 text-slate-300"
};

/**
 * Human-readable labels for preflight check statuses.
 */
export const PREFLIGHT_CHECK_LABELS: Record<BundlePreflightCheck["status"], string> = {
  pass: "PASS",
  fail: "FAIL",
  warning: "WARN",
  skipped: "SKIP"
};

// ============================================================================
// Coverage Map
// ============================================================================

/**
 * Coverage columns showing what aspects are validated.
 */
export const COVERAGE_COLUMNS: CoverageCell[] = [
  { label: "Runtime", active: true },
  { label: "Validation", active: true },
  { label: "Secrets", active: true },
  { label: "Services", active: false },
  { label: "Electron UI", active: false }
];

/**
 * Coverage rows comparing preflight vs full app validation.
 */
export const COVERAGE_ROWS: CoverageRow[] = [
  {
    id: "preflight",
    label: "Preflight",
    note: "Runtime + services + logs",
    cells: COVERAGE_COLUMNS.map((cell) => ({
      ...cell,
      active: cell.label === "Electron UI" ? false : true
    }))
  },
  {
    id: "full",
    label: "Full app",
    note: "Runtime + services + UI",
    cells: COVERAGE_COLUMNS.map((cell) => ({ ...cell, active: true }))
  }
];

// ============================================================================
// Job Step State Mapping
// ============================================================================

/**
 * Maps backend job step states to UI labels.
 */
export const JOB_STEP_STATE_LABELS: Record<string, string> = {
  pending: "Pending",
  running: "Running",
  pass: "Pass",
  fail: "Fail",
  warning: "Warning",
  skipped: "Skipped"
};
