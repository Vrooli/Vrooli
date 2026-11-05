import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { FocusBoard } from "../features/focus/FocusBoard";
import { useTranslation } from "../i18n";

/** Focus & gaps page: ranked next-best gaps + the gaps registry. */
export function FocusPage() {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.pages.focus}
      aria-labelledby="focus-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="focus-heading" className="text-2xl font-semibold">
        {t(strings.pages.focus.title)}
      </h2>
      <p className="text-app-muted-foreground">{t(strings.pages.focus.description)}</p>
      <FocusBoard />
    </section>
  );
}
