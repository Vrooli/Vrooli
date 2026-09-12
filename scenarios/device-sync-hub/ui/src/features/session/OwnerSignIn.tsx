import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { OwnerLoginForm } from "../onboarding/OwnerLoginForm";
import { useSession } from "./SessionProvider";

/**
 * Owner sign-in surface for the paired shell (Settings): shows the signed-in
 * owner with a sign-out, or the same-origin sign-in / create-account form when
 * the owner token is absent or expired. Reuses OwnerLoginForm so there is one
 * owner-auth implementation across onboarding and settings; the browser never
 * calls scenario-authenticator directly.
 */
export function OwnerSignIn() {
  const { t } = useTranslation();
  const { isOwner, ownerEmail, clearOwnerToken } = useSession();

  return (
    <div data-testid={selectors.owner.panel} className="flex flex-col gap-3">
      {isOwner ? (
        <div className="flex flex-wrap items-center gap-3">
          <p data-testid={selectors.owner.status} className="text-sm text-app-success">
            {ownerEmail ? t(strings.setupDevice.signedInAs, { owner: ownerEmail }) : t(strings.owner.signedIn)}
          </p>
          <Button
            data-testid={selectors.owner.signOutButton}
            variant="outline"
            size="sm"
            onClick={clearOwnerToken}
          >
            {t(strings.owner.signOut)}
          </Button>
        </div>
      ) : (
        <>
          <p className="text-sm text-app-muted-foreground">{t(strings.owner.intro)}</p>
          <OwnerLoginForm />
        </>
      )}
    </div>
  );
}
