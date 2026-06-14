import { Badge } from "../components/ui/badge";
import { selectors } from "../consts/selectors";
import { useTranslation } from "../i18n";

export function TopBar() {
  const { t } = useTranslation();

  return (
    <header
      className="border-b border-border/50 bg-background/80 px-6 py-5 backdrop-blur-sm sm:px-10"
      data-testid={selectors.layout.topBar}
    >
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-2">
          <Badge className="uppercase tracking-wide" variant="secondary">
            {t("app.eyebrow")}
          </Badge>
          <div>
            <h1 className="text-2xl font-semibold text-foreground sm:text-3xl">
              {t("app.title")}
            </h1>
            <p className="mt-2 max-w-3xl text-sm text-muted-foreground">
              {t("app.description")}
            </p>
          </div>
        </div>
      </div>
    </header>
  );
}
