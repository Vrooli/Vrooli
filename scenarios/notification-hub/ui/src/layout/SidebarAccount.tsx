import { Link, useLocation } from "react-router-dom";

import { strings } from "../consts/strings";
import { preserveRoutedTestMode } from "../features/routedTestMode";
import { useSession } from "../features/session/useSession";
import { useTranslation } from "../i18n";

/** The foot of the desktop sidebar: who the Hub is signed in as, or the way to sign in. */
export function SidebarAccount() {
  const { t } = useTranslation();
  const { search } = useLocation();
  const session = useSession();
  if (session.isPending) return null;
  return session.isSuccess && session.data.signedIn ? (
    <p className="truncate px-2 py-1 text-xs text-app-muted-foreground" title={session.data.email || session.data.subject}>
      {t(strings.session.signedInAs, { email: session.data.email || session.data.subject })}
    </p>
  ) : (
    <Link to={preserveRoutedTestMode("/settings", search)} className="flex min-h-11 items-center px-2 text-sm font-medium text-app-primary hover:underline">
      {t(strings.layout.account.signedOut)}
    </Link>
  );
}
