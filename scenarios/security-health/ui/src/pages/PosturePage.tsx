import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { PostureCard } from "../features/posture/PostureCard";
import { HealthCard } from "../features/health/HealthCard";
import { useTranslation } from "../i18n";

/**
 * Posture is the home page: API health strip + the validation scan surface
 * (severity-grouped findings with remediation). Index route for the console.
 */
export function PosturePage() {
  const { t } = useTranslation();

  return (
    <section data-testid={selectors.pages.posture} className="flex flex-col gap-4">
      <div>
        <h2 className="text-2xl font-semibold">
          {t(strings.pages.posture.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.posture.description)}</p>
      </div>
      <PostureCard />
      <HealthCard />
    </section>
  );
}
