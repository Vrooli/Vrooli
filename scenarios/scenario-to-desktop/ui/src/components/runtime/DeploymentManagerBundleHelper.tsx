/**
 * @deprecated This component is deprecated. Use BundleSection instead.
 * This file re-exports types for backwards compatibility.
 */

// Re-export types from the new location for backwards compatibility
export type {
  BundleResult,
  BundleSectionHandle as DeploymentManagerBundleHelperHandle,
} from "../sections/bundle/BundleSection";

// Note: The actual component is no longer used - BundleSection now handles everything.
// If you need bundle functionality, use BundleSection directly.
