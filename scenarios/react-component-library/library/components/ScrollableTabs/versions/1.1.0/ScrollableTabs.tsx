/**
 * @libraryId react-component-library:ScrollableTabs
 * @displayName ScrollableTabs
 * @description A tab surface that keeps navigation usable when labels exceed the available inline space.
 * @version 1.1.0
 * @tags ["navigation","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource navigation.scrollable-tabs */
import { Tabs, type TabsProps } from "@vrooli/react-component-library/Tabs/1.1.0";

export type { TabsProps as ScrollableTabsProps };

/**
 * A compatibility delegate to {@link Tabs}. Start new work from `Tabs`.
 *
 * This asset existed because the tab strip could not carry its own overflow.
 * From `Tabs@1.1.0` it can, so the wrapper has nothing left to add — and what
 * it did add was harmful: it wrapped the strip in `overflow: hidden`, which
 * clipped the tabs the component is named for instead of letting them scroll.
 * The export is retained so existing pins keep resolving; it now forwards
 * every prop to `Tabs` unchanged.
 */
export function ScrollableTabs(props: TabsProps) {
  return <Tabs {...props} />;
}
