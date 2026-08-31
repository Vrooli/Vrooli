/**
 * @libraryId react-component-library:ScrollableTabs
 * @displayName ScrollableTabs
 * @version 1.1.3
 * @tags ["navigation","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource navigation.scrollable-tabs */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import { Tabs, type TabsProps } from "@vrooli/react-component-library/Tabs/1";

export type { TabsProps as ScrollableTabsProps };

/**
 * A compatibility delegate to {@link Tabs}. Start new work from `Tabs`.
 *
 * This asset existed because the tab strip could not carry its own overflow.
 * From `Tabs@1.1.0` it can, so the wrapper has nothing left to add — and what
 * it did add was harmful: it wrapped the strip in `overflow: hidden`, which
 * clipped the tabs the component is named for instead of letting them scroll.
 * The addressable root and the className seam are retained so existing pins
 * and flows keep resolving; every prop forwards to `Tabs` unchanged.
 */
export const ScrollableTabs = withClassName(function ScrollableTabs(props: TabsProps) {
  return (
    <div data-testid="navigation.scrollable-tabs" data-scrollable-tabs>
      <Tabs {...props} />
    </div>
  );
});
