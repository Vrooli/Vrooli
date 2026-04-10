/**
 * @deprecated This component is deprecated. Use BundleSection instead.
 * BundleSection now provides a unified, linear-step interface for bundle generation.
 *
 * This file is kept for backwards compatibility but delegates to BundleSection.
 */

import type { Ref } from "react";
import type { BundleResult, BundleSectionHandle } from "../sections/bundle/BundleSection";

// Re-export types for backwards compatibility
export type { BundleResult, BundleSectionHandle as DeploymentManagerBundleHelperHandle };

interface BundledRuntimeSectionProps {
  bundleManifestPath: string;
  onBundleManifestChange: (value: string) => void;
  scenarioName: string;
  bundleHelperRef: Ref<BundleSectionHandle>;
  onBundleExported?: (manifestPath: string) => void;
  onBundleComplete?: (result: BundleResult) => void;
  initialBundleResult?: BundleResult | null;
}

/**
 * @deprecated Use BundleSection directly instead.
 * This component is now a no-op placeholder that returns null.
 * All bundle functionality has been consolidated into BundleSection.
 */
export function BundledRuntimeSection(_props: BundledRuntimeSectionProps) {
  // This component is deprecated - BundleSection now handles everything directly.
  // Returning null since BundleSection is now rendered directly by the parent.
  return null;
}
