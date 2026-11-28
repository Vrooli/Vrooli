import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { ValidationBoard } from "../features/validation/ValidationBoard";
import { useTranslation } from "../i18n";

/** Validation board page. */
export function ValidationPage() {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.pages.validation}
      aria-labelledby="validation-heading"
      className="flex flex-col gap-4"
    >
      <header className="flex flex-col gap-1">
        <h2 id="validation-heading" className="text-2xl font-semibold">
          {t(strings.pages.validation.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.validation.description)}</p>
      </header>
      <ValidationBoard />
    </section>
  );
}
