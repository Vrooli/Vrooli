import type { QualityTierLabel } from "../api/studio";

import { strings } from "./strings";

/**
 * The string key for one quality tier.
 *
 * A lookup rather than a ternary at each call site: the tier is shown on the
 * catalog tile, the style page and — once routing is surfaced — the candidate
 * record, and three copies of the same branch is how the specimen tile and the
 * style page came to disagree about what "metered" meant.
 */
export function qualityTierString(
  tier: QualityTierLabel,
):
  | typeof strings.pages.catalog.tierProcedural
  | typeof strings.pages.catalog.tierLocalModel
  | typeof strings.pages.catalog.tierFrontierModel {
  switch (tier) {
    case "local_model":
      return strings.pages.catalog.tierLocalModel;
    case "frontier_model":
      return strings.pages.catalog.tierFrontierModel;
    default:
      return strings.pages.catalog.tierProcedural;
  }
}
