import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { useSession } from "./SessionProvider";

/**
 * Shows this browser's pairing state and a sign-out that drops the device token
 * (this browser leaves the trust group locally and returns to the join screen).
 */
export function SessionPanel() {
  const { t } = useTranslation();
  const { isPaired, session, signOut } = useSession();

  return (
    <div data-testid={selectors.session.panel} className="flex flex-col gap-2">
      {isPaired ? (
        <>
          <p className="text-sm text-app-foreground">
            {t(strings.session.tokenPresent)}
            {session.device?.name ? ` (${session.device.name})` : ""}
          </p>
          <Button
            data-testid={selectors.session.signOutButton}
            variant="outline"
            size="sm"
            className="w-fit"
            onClick={signOut}
          >
            {t(strings.session.signOut)}
          </Button>
        </>
      ) : (
        <p className="text-sm text-app-muted-foreground">{t(strings.session.notPaired)}</p>
      )}
    </div>
  );
}
