export { BundledRuntimeSection } from "./BundledRuntimeSection";
export { EmbeddedServerSection } from "./EmbeddedServerSection";
export { ExternalServerSection } from "./ExternalServerSection";

// Re-export bundle types from the new location for backwards compatibility
export type {
  BundleResult,
  BundleSectionHandle,
  DeploymentManagerBundleHelperHandle,
} from "../sections/bundle/BundleSection";
