import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { LibraryView } from "../features/library/LibraryView";
import { useTranslation } from "../i18n";

/** Library route — the gallery of every image you've created or edited. */
export function LibraryPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.library}
      aria-labelledby="library-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="library-heading" className="text-2xl font-semibold">
        {t(strings.pages.library.title)}
      </h2>
      <LibraryView />
    </section>
  );
}
