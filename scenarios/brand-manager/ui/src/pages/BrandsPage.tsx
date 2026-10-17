import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { BrandsCard } from "../features/brands/BrandsCard";
import { useTranslation } from "../i18n";

export function BrandsPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.brands}
      aria-labelledby="brands-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="brands-heading" className="text-2xl font-semibold">
        {t(strings.pages.brands.title)}
      </h2>
      <BrandsCard />
    </section>
  );
}
