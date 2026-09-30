import type { RefObject } from "react";
import type {
  BundleResult,
  BundleSectionHandle,
} from "../sections/bundle/BundleSection";
import type { ValidationError } from "../../domain/generator";

/** State shared between the configuration form and dependent pipeline sections. */
export interface ExposedFormState {
  bundleManifestPath: string;
  isBundled: boolean;
  bundleManifest?: unknown;
  onBundleManifestChange: (path: string) => void;
  onBundleComplete: (result: BundleResult) => void;
  bundleHelperRef: RefObject<BundleSectionHandle>;
}

/** Validation state exposed to the parent generate action. */
export interface ValidationState {
  errors: ValidationError[];
  clearErrors: () => void;
  isPending: boolean;
  isError: boolean;
  errorMessage: string | null;
  isUpdateMode: boolean;
}
