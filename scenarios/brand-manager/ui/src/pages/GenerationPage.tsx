import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { GenerationCard } from "../features/generation/GenerationCard";
import { useTranslation } from "../i18n";

export function GenerationPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.generation}
      aria-labelledby="generation-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="generation-heading" className="text-2xl font-semibold">
        {t(strings.pages.generation.title)}
      </h2>
      <GenerationCard />
    </section>
  );
}
