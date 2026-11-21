import { useId, type ReactNode } from "react";
import { BackLink } from "../../components/BackLink";
import { Page } from "../../components/Page";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { useWideSettings } from "./layoutMode";
import { useSettingsPath } from "./testContext";

export interface ScreenBack {
  to: string;
  label: string;
}

/**
 * One settings screen: the page title, a way back, and the body. A phone needs
 * a link back to the list of settings; beside the list the list is the way
 * back, so `backToSettings` shows nothing there. `back` names a parent screen
 * and is always shown.
 */
export function SettingsScreen({ title, description, back, backToSettings = false, testId, children }: {
  title: string;
  description?: ReactNode;
  back?: ScreenBack;
  backToSettings?: boolean;
  testId?: string;
  children: ReactNode;
}) {
  const { t } = useTranslation();
  const wide = useWideSettings();
  const path = useSettingsPath();
  const headingId = useId();
  const target: ScreenBack | undefined = back ?? (backToSettings && !wide ? { to: "/settings", label: t(strings.pages.settings.title) } : undefined);
  return (
    <Page
      title={title}
      description={description}
      headingId={headingId}
      testId={testId}
      leading={target && <BackLink to={path(target.to)} data-testid={selectors.settings.back}>{target.label}</BackLink>}
    >
      {children}
    </Page>
  );
}
