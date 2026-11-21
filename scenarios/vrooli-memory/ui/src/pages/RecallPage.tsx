import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { RecallPanel } from "../features/recall/RecallPanel";
import { useTranslation } from "../i18n";

export function RecallPage() {
  const { t } = useTranslation();
  return <section data-testid={selectors.pages.recall} aria-labelledby="recall-heading" className="flex flex-col gap-4"><h2 id="recall-heading" className="text-2xl font-semibold">{t(strings.pages.recall.title)}</h2><RecallPanel /></section>;
}
