import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { SkeletonRows } from "../../components/Skeleton";
import { SessionPanel, SessionUnavailable } from "../session/SessionGate";
import { RootEnrollment } from "../session/RootEnrollment";
import { SignedOutCard } from "../session/SignedOutCard";
import { useSession } from "../session/useSession";
import { SettingsScreen } from "./SettingsScreen";

/** Who is signed in to this Hub in this browser, and the way to sign in or out. */
export function SessionPage() {
  const { t } = useTranslation();
  const session = useSession();
  return (
    <SettingsScreen title={t(strings.settings.section.session)} description={t(strings.settings.sessionPage.description)} backToSettings>
      {session.isPending ? <SkeletonRows rows={1} /> : session.isError ? <SessionUnavailable retry={() => { void session.refetch(); }} /> : session.data.signedIn ? (
        <>
          <SessionPanel />
          <p className="text-sm text-app-muted-foreground">{t(strings.session.draftsBrowserOnly)}</p>
        </>
      ) : <SignedOutCard />}
      <RootEnrollment />
    </SettingsScreen>
  );
}
