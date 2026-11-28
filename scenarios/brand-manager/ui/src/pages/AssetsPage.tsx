import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { AssetsCard } from "../features/assets/AssetsCard";
import { useTranslation } from "../i18n";

export function AssetsPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.assets}
      aria-labelledby="assets-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="assets-heading" className="text-2xl font-semibold">
        {t(strings.pages.assets.title)}
      </h2>
      <AssetsCard />
    </section>
  );
}
