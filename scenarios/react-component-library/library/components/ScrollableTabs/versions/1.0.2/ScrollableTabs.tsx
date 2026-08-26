/**
 * @libraryId react-component-library:ScrollableTabs
 * @displayName ScrollableTabs
 * @description A tab surface that keeps navigation usable when labels exceed the available inline space.
 * @version 1.0.2
 * @tags ["navigation","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource navigation.scrollable-tabs */
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

import { Tabs, type TabsProps } from "../../../Tabs/versions/1.0.0/Tabs";

export const ScrollableTabs = withClassName(function ScrollableTabs(props: TabsProps) {
  return (
    <div data-scrollable-tabs style={{ maxWidth: "100%", overflow: "hidden" }}>
      <Tabs {...props} />
    </div>
  );
});
