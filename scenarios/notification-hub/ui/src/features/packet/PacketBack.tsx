import { ChevronLeft } from "lucide-react";
import { Link } from "react-router-dom";

import { BackLink } from "../../components/BackLink";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * The way back from a decision, named for the list it returns to. The list
 * restores focus to this decision's row when it was the one opened.
 * `compact` is the arrow alone, for the start of the title row; its accessible
 * name is still the list's name.
 */
export function PacketBack({ returnTo, askId, compact = false }: { returnTo: string; askId?: string; compact?: boolean }) {
  const { t } = useTranslation();
  const label = t(returnTo.startsWith("/history") ? strings.layout.nav.history : strings.layout.nav.asks);
  const state = askId ? { restoreAskId: askId } : undefined;
  if (compact) {
    return (
      <Link
        to={returnTo}
        state={state}
        aria-label={label}
        title={label}
        className="-ms-2 inline-flex min-h-11 min-w-11 flex-none items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground focus-visible:outline focus-visible:outline-2"
      >
        <ChevronLeft size={22} aria-hidden="true" className="rtl:rotate-180" />
      </Link>
    );
  }
  return (
    // On a phone the link's 44px target rises into the pane's top gutter, so the
    // first line of the page is not a row of dead space under the status bar.
    <div className="-mt-4 md:mt-0">
      <BackLink to={returnTo} state={state}>
        {label}
      </BackLink>
    </div>
  );
}
