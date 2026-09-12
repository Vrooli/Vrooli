import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { ApplyCard } from "../features/apply/ApplyCard";
import { useTranslation } from "../i18n";

export function ApplyPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.apply}
      aria-labelledby="apply-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="apply-heading" className="text-2xl font-semibold">
        {t(strings.pages.apply.title)}
      </h2>
      <ApplyCard />
    </section>
  );
}
