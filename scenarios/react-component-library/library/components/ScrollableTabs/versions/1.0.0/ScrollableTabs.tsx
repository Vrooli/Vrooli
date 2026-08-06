/** @vrooliComponentSource navigation.scrollable-tabs */
import { Tabs, type TabsProps } from "../../../Tabs/versions/1.0.0/Tabs";

export function ScrollableTabs(props: TabsProps) {
  return (
    <div data-scrollable-tabs style={{ maxWidth: "100%", overflow: "hidden" }}>
      <Tabs {...props} />
    </div>
  );
}
