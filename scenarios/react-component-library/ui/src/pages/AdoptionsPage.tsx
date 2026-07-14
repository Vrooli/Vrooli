/**
 * AdoptionsPage — full-width adoption registry page (req 08).
 */
import { AdoptionsCard } from "../features/adoptions/AdoptionsCard";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useSearchParams } from "react-router-dom";

export function AdoptionsPage() {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const componentId = searchParams.get("componentId") ?? undefined;
  return (
    <div data-testid="adoptions-page" className="flex flex-col gap-4">
      <header>
        <h1 className="text-2xl font-semibold text-app-foreground">
          {t(strings.adoptions.title)}
        </h1>
        <p className="mt-1 text-sm text-app-muted-foreground">
          {t(strings.adoptions.subtitle)}
        </p>
      </header>
      <AdoptionsCard componentId={componentId} />
    </div>
  );
}
