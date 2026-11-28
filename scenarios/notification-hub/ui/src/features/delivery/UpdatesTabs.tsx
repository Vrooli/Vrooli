import { useLocation } from "react-router-dom";

import { SectionTabs } from "../../components/SectionTabs";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { preserveRoutedTestMode } from "../routedTestMode";

/** Updates and Delivery are two views of what the Hub sent; this switches between them. */
export function UpdatesTabs({ current }: { current: "updates" | "delivery" }) {
  const { t } = useTranslation();
  const { search } = useLocation();
  return (
    <SectionTabs
      label={t(strings.pages.dashboard.sectionsLabel)}
      tabs={[
        { id: "updates", testId: selectors.updates.tab({ id: "updates" }), label: t(strings.pages.dashboard.title), to: preserveRoutedTestMode("/updates", search), current: current === "updates" },
        { id: "delivery", testId: selectors.updates.tab({ id: "delivery" }), label: t(strings.pages.delivery.title), to: preserveRoutedTestMode("/delivery", search), current: current === "delivery" },
      ]}
    />
  );
}
