import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { PostureCard } from "../features/posture/PostureCard";
import { useTranslation } from "../i18n";

/**
 * Secrets page: the same validation scan as Posture, but filtered to the
 * `gitleaks` scanner so only secret findings surface. Values are never shown —
 * the backend reports file:line only, and the redaction note makes that
 * contract visible to the operator.
 */
export function SecretsPage() {
  const { t } = useTranslation();

  return (
    <section data-testid={selectors.pages.secrets} className="flex flex-col gap-4">
      <div>
        <h2 className="text-2xl font-semibold">
          {t(strings.pages.secrets.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.secrets.description)}</p>
      </div>
      <p data-testid={selectors.secrets.redactedNote} className="text-xs text-app-muted-foreground">
        {t(strings.secrets.redactedNote)}
      </p>
      <PostureCard scannerFilter="gitleaks" />
    </section>
  );
}
