import { Link, Navigate, Outlet, useMatch } from "react-router-dom";
import { Page } from "../components/Page";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useSession } from "../features/session/useSession";
import { useWideSettings } from "../features/settings/layoutMode";
import { Notice } from "../features/settings/Notice";
import { SectionList } from "../features/settings/SectionList";
import type { SectionKey } from "../features/settings/sections";
import { useSettingsPath } from "../features/settings/testContext";

const SECTION_BY_SEGMENT: Record<string, SectionKey> = {
  reach: "reach",
  "quiet-hours": "quietHours",
  escalation: "escalation",
  appearance: "appearance",
  session: "session",
  sources: "sources",
};

/** The first setting in the list; what a wide screen opens beside the list. */
const DEFAULT_SECTION = "/settings/reach";

/**
 * Settings. A phone gets the list as its own page and every setting opens over
 * it. A wide screen keeps the list on the left and the open setting on the
 * right, so changing one thing never costs a trip back.
 */
export function SettingsPage() {
  const { t } = useTranslation();
  const wide = useWideSettings();
  const match = useMatch("/settings/:section/*");
  const current = match?.params.section ? SECTION_BY_SEGMENT[match.params.section] : undefined;
  if (!wide) return <div data-testid={selectors.pages.settings}><Outlet /></div>;
  return (
    <div data-testid={selectors.pages.settings} className="mx-auto grid w-full max-w-[66rem] grid-cols-[17rem_minmax(0,44rem)] items-start gap-10">
      <nav aria-label={t(strings.pages.settings.title)} className="flex flex-col gap-4">
        <p className="px-1 text-xl font-semibold">{t(strings.pages.settings.title)}</p>
        <SectionList current={current} compact />
      </nav>
      <Outlet />
    </div>
  );
}

/** `/settings`: the list on a phone, the first setting on a wide screen. */
export function SettingsIndex() {
  const { t } = useTranslation();
  const wide = useWideSettings();
  const session = useSession();
  const path = useSettingsPath();
  if (wide) return <Navigate to={path(DEFAULT_SECTION)} replace />;
  const signedOut = session.isSuccess && !session.data.signedIn;
  return (
    <Page title={t(strings.pages.settings.title)} headingId="settings-heading">
      {signedOut && (
        <Notice tone="info" title={t(strings.settings.signedOut.title)}>
          <p>{t(strings.settings.signedOut.body)}</p>
          <Link to={path("/settings/session")} className="mt-1 inline-flex min-h-11 items-center font-medium text-app-primary underline underline-offset-4">{t(strings.settings.signedOut.action)}</Link>
        </Notice>
      )}
      <SectionList />
    </Page>
  );
}
