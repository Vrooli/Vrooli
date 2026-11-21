import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { AuthoringWizard } from "../features/authoring/AuthoringWizard";
import { useTranslation } from "../i18n";

/** Authoring wizard page. */
export function AuthoringPage() {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.pages.authoring}
      aria-labelledby="authoring-heading"
      className="flex flex-col gap-4"
    >
      <header className="flex flex-col gap-1">
        <h2 id="authoring-heading" className="text-2xl font-semibold">
          {t(strings.pages.authoring.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.authoring.description)}</p>
      </header>
      <AuthoringWizard />
    </section>
  );
}
