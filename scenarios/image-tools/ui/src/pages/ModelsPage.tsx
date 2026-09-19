import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { AddCustomModelForm } from "../features/models/AddCustomModelForm";
import { BackendsCard } from "../features/models/BackendsCard";
import { BlocklistCard } from "../features/models/BlocklistCard";
import { ImportModelWizard } from "../features/models/ImportModelWizard";
import { ModelsCard } from "../features/models/ModelsCard";
import { useTranslation } from "../i18n";

export function ModelsPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.models}
      aria-labelledby="models-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="models-heading" className="text-2xl font-semibold">
        {t(strings.pages.models.title)}
      </h2>
      <ModelsCard />
      <BackendsCard />
      <ImportModelWizard />
      <AddCustomModelForm />
      <BlocklistCard />
    </section>
  );
}
