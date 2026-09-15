import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { NewTargetForm } from "../features/targets/NewTargetForm";
import { useTranslation } from "../i18n";

/** Start-extract page — single column, contains the new-target form. */
export function NewTargetPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.newTarget}
      aria-labelledby="new-target-heading"
      className="mx-auto flex w-full max-w-2xl flex-col gap-4"
    >
      <header className="flex flex-col gap-2">
        <h2 id="new-target-heading" className="text-2xl font-semibold">
          {t(strings.pages.newTarget.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.newTarget.description)}</p>
      </header>
      <NewTargetForm />
    </section>
  );
}
