import { strings } from "../../consts/strings";

/** The upscale-factor options the inspector offers. */
export const UPSCALE_SCALES = ["2", "4"] as const;

type TierKey = (typeof strings.workspace.enhance.tier)[keyof typeof strings.workspace.enhance.tier];

/**
 * Label key for each backend selection tier (`local-gpu` / `local-cpu` /
 * `byok-cloud` / `gateway-local` — see `internal/backends`). A literal map (rather than dynamic
 * `strings.…[tier]` access) so the unused-key audit can trace each callsite.
 *
 * Per-operation presentation (label/description/icon) lives in the capability
 * manifests and is consumed via `capabilityPresentation`; it is not duplicated
 * here.
 */
export const TIER_LABEL: Readonly<Record<string, TierKey>> = {
  "local-gpu": strings.workspace.enhance.tier.localGpu,
  "local-cpu": strings.workspace.enhance.tier.localCpu,
  "byok-cloud": strings.workspace.enhance.tier.byokCloud,
  "gateway-local": strings.workspace.enhance.tier.gatewayLocal,
};
