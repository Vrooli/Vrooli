/**
 * @libraryId react-component-library:ScrollableTabs
 * @displayName ScrollableTabs
 * @description A tab surface that keeps navigation usable when labels exceed the available inline space.
 * @version 1.0.3
 * @tags ["navigation","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource navigation.scrollable-tabs */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import {
  Tabs,
  type TabsProps,
} from "@vrooli/react-component-library/Tabs/1";

export const ScrollableTabs = withClassName(function ScrollableTabs(
  props: TabsProps,
) {
  return (
    <div
      data-testid="navigation.scrollable-tabs"
      data-scrollable-tabs
      style={{ maxWidth: "100%", overflow: "hidden" }}
    >
      <Tabs {...props} />
    </div>
  );
});
