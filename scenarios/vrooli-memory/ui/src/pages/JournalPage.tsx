import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { JournalTimeline } from "../features/journal/JournalTimeline";
import { useTranslation } from "../i18n";

export function JournalPage() {
  const { t } = useTranslation();
  return <section data-testid={selectors.pages.journal} aria-labelledby="journal-heading" className="flex flex-col gap-4"><h2 id="journal-heading" className="text-2xl font-semibold">{t(strings.pages.journal.title)}</h2><JournalTimeline /></section>;
}
