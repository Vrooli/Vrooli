import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { DesignCard } from "../features/design/DesignCard";
import { useTranslation } from "../i18n";

export function DesignPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.design}
      aria-labelledby="design-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="design-heading" className="text-2xl font-semibold">
        {t(strings.pages.design.title)}
      </h2>
      <DesignCard />
    </section>
  );
}
