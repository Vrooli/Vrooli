import { Aperture, Scissors, Sparkles, Wand2, type LucideIcon } from "lucide-react";

import { strings } from "../../consts/strings";

type AIOpStrings = typeof strings.workspace.aiOp;
type AIOpEntry = AIOpStrings[keyof AIOpStrings];

/** Per-AI-op presentation: friendly label/description i18n keys and an icon. */
export interface AIOpPresentation {
  labelKey: AIOpEntry["label"];
  descKey: AIOpEntry["desc"];
  Icon: LucideIcon;
}

/**
 * Maps each known enhancement op (discovered from `AIService.ListAIOperations`,
 * category `enhancement`) to its humanized label/description keys and a lucide
 * icon. Driven by the live catalog, so a backend that adds an enhancement op
 * still renders (via the fallback below) without a code change here. `denoise`
 * is the backend's "reduce noise / deblur" op — surfaced as "Unblur & denoise"
 * because that is what users reach for it to do.
 */
export const AI_CATALOG: Readonly<Record<string, AIOpPresentation>> = {
  background_removal: {
    labelKey: strings.workspace.aiOp.background_removal.label,
    descKey: strings.workspace.aiOp.background_removal.desc,
    Icon: Scissors,
  },
  upscale: {
    labelKey: strings.workspace.aiOp.upscale.label,
    descKey: strings.workspace.aiOp.upscale.desc,
    Icon: Sparkles,
  },
  denoise: {
    labelKey: strings.workspace.aiOp.denoise.label,
    descKey: strings.workspace.aiOp.denoise.desc,
    Icon: Aperture,
  },
};

/** Fallback icon for an enhancement op the catalog doesn't yet name. */
export const AI_FALLBACK_ICON: LucideIcon = Wand2;

/** Presentation for a known AI op, or `undefined` if it isn't in the catalog. */
export const aiPresentation = (operation: string): AIOpPresentation | undefined =>
  AI_CATALOG[operation];

/** The upscale-factor options the inspector offers. */
export const UPSCALE_SCALES = ["2", "4"] as const;

type TierKey = (typeof strings.workspace.enhance.tier)[keyof typeof strings.workspace.enhance.tier];

/**
 * Label key for each backend selection tier (`local-gpu` / `local-cpu` /
 * `byok-cloud` — see `internal/backends`). A literal map (rather than dynamic
 * `strings.…[tier]` access) so the unused-key audit can trace each callsite.
 */
export const TIER_LABEL: Readonly<Record<string, TierKey>> = {
  "local-gpu": strings.workspace.enhance.tier.localGpu,
  "local-cpu": strings.workspace.enhance.tier.localCpu,
  "byok-cloud": strings.workspace.enhance.tier.byokCloud,
};
