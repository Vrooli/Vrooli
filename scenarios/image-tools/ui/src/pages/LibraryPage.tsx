import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { LibraryView } from "../features/library/LibraryView";
import { LooksView } from "../features/looks/LooksView";
import { useTranslation } from "../i18n";

/** Library route — the gallery of every image you've created or edited, plus
 * the reusable Look/Style library. */
export function LibraryPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.library}
      aria-labelledby="library-heading"
      className="flex flex-col gap-8"
    >
      <div className="flex flex-col gap-4">
        <h2 id="library-heading" className="text-2xl font-semibold">
          {t(strings.pages.library.title)}
        </h2>
        <LibraryView />
      </div>
      <div className="flex flex-col gap-4">
        <h2 id="looks-heading" className="text-2xl font-semibold">
          {t(strings.looks.title)}
        </h2>
        <LooksView />
      </div>
    </section>
  );
}
