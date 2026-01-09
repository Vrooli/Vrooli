/**
 * Preflight validation UI components.
 *
 * This module provides the building blocks for the preflight validation UI.
 * Components are designed to be composable and testable in isolation.
 */

// Core step components
export { PreflightStepHeader } from "./PreflightStepHeader";
export { PreflightCheckList } from "./PreflightCheckList";

// Coverage visualization
export { CoverageBadge } from "./CoverageBadge";
export { CoverageMap } from "./CoverageMap";

// Validation step
export { ValidationIssuesPanel, ValidationWarningsPanel } from "./ValidationIssuesPanel";

// Secrets step
export { MissingSecretsForm } from "./MissingSecretsForm";

// Runtime step
export { RuntimeInfoPanel } from "./RuntimeInfoPanel";

// Services step
export { ServicesReadinessGrid } from "./ServicesReadinessGrid";

// Diagnostics step
export { LogTailsPanel, FingerprintsPanel, PortSummaryPanel } from "./DiagnosticsPanels";
