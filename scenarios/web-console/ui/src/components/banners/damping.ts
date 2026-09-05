/**
 * Temporal damping is the library's — see `react-component-library:Banner`.
 *
 * This module remains as the app's import seam. `INSTANT_DAMPING` is what the
 * app's own tests reach for when they are asserting content rather than timing.
 */
export {
  createPresentationState,
  DEFAULT_DAMPING,
  dismissBanner,
  INSTANT_DAMPING,
  makeDampingResolver,
  reconcileBanners,
  resolveDamping,
  type BannerDamping,
  type BannerPhase,
  type PresentationState,
  type PresentedBanner,
  type ReconcileResult,
  type TrackedBanner,
} from "@vrooli/react-component-library/Banner";
