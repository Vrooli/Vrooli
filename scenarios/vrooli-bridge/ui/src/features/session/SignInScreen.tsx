import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { TopBar } from "../../layout/TopBar";
import { OwnerSignIn } from "./OwnerSignIn";

/**
 * Unauthenticated gate surface. The bridge console is owner-gated end-to-end
 * (every fleet RPC requires an owner JWT), so with no owner token we present the
 * sign-in / create-account form as the whole surface instead of a dashboard that
 * could only render "please sign in" errors. The TopBar chrome stays so theme
 * and locale remain reachable before sign-in.
 */
export function SignInScreen() {
  const { t } = useTranslation();
  return (
    <div
      data-testid={selectors.session.signInScreen}
      className="flex min-h-screen flex-col bg-app-background text-app-foreground"
    >
      <TopBar />
      <main className="flex flex-1 items-center justify-center p-6">
        <div className="flex w-full max-w-md flex-col gap-4">
          <div className="flex flex-col gap-1">
            <h1 className="text-2xl font-semibold">{t(strings.session.screenTitle)}</h1>
            <p className="text-sm text-app-muted-foreground">{t(strings.session.screenIntro)}</p>
          </div>
          <p
            data-testid={selectors.session.firstTimeNote}
            className="rounded-panel border border-app-primary/30 bg-app-primary/10 p-3 text-sm text-app-foreground"
          >
            {t(strings.session.firstTimeNote)}
          </p>
          <OwnerSignIn />
        </div>
      </main>
    </div>
  );
}
