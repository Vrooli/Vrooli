import type { TFunction } from "i18next";
import { useTranslation } from "react-i18next";
import BannerRegion from "../BannerRegion";
import { INSTANT_DAMPING } from "../damping";
import type { MaybeBanner } from "../types";

/**
 * Render banner descriptors the way the app does — through `BannerRegion`,
 * with a real `t` from the provider — so a test exercises arbitration,
 * presentation and translation together rather than a component in isolation.
 *
 * Damping is disabled here on purpose. These suites assert *what* a banner
 * says and does; *when* it appears and leaves is a policy with its own
 * millisecond-resolution suite in `damping.test.ts`, and mixing the two would
 * make every content assertion wait on a timer.
 */
export function BannerHarness({
  build,
}: {
  build: (t: TFunction) => MaybeBanner | MaybeBanner[];
}) {
  const { t } = useTranslation();
  const built = build(t);
  return (
    <BannerRegion
      banners={Array.isArray(built) ? built : [built]}
      damping={INSTANT_DAMPING}
    />
  );
}
