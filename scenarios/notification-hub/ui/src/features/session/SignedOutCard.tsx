import { Link } from "react-router-dom";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { SignInForm } from "./SessionGate";

/** Shown where a page needs the owner and there is no Hub session: the reason, and the way in. */
export function SignedOutCard() {
  const { t } = useTranslation();
  return (
    <section aria-label={t(strings.session.title)} className="mx-auto flex w-full max-w-sm flex-col items-center gap-4 rounded-panel border border-app-border bg-app-surface p-6">
      <img src="/public/icon-192.png" alt="" aria-hidden="true" width={48} height={48} className="h-12 w-12 rounded-xl" />
      <SignInForm />
      <div className="w-full border-t border-app-border pt-4">
        <p className="text-sm text-app-muted-foreground">{t(strings.session.readOnlyBrowse)}</p>
        <nav aria-label={t(strings.layout.bottomNavLabel)} className="mt-2 flex flex-wrap gap-4">
          <Link to="/updates" className="min-h-11 content-center text-sm font-medium text-app-primary hover:underline">{t(strings.layout.nav.dashboard)}</Link>
          <Link to="/delivery" className="min-h-11 content-center text-sm font-medium text-app-primary hover:underline">{t(strings.pages.delivery.title)}</Link>
        </nav>
      </div>
    </section>
  );
}
