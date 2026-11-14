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
    </section>
  );
}
