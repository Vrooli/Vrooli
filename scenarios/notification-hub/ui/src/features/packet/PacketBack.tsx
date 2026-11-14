import { BackLink } from "../../components/BackLink";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/** The way back from a decision, named for the list it returns to. The list restores focus to this decision's row when it was the one opened. */
export function PacketBack({ returnTo, askId }: { returnTo: string; askId?: string }) {
  const { t } = useTranslation();
  return (
    <BackLink to={returnTo} state={askId ? { restoreAskId: askId } : undefined}>
      {t(returnTo.startsWith("/history") ? strings.layout.nav.history : strings.layout.nav.asks)}
    </BackLink>
  );
}
