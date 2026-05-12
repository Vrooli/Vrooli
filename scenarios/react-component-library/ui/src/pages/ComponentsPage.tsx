/**
 * ComponentsPage — full-width library page.
 *
 * Today this wraps `<ComponentsCard />` (lifted out of the placeholder
 * App composition). As the page grows scenario-specific UX (search,
 * facet sidebar, bulk re-index, drift summary) we expand here without
 * re-routing.
 */
import { ComponentsCard } from "../features/components/ComponentsCard";
import { useTranslation } from "../i18n";

export function ComponentsPage() {
  const { t } = useTranslation();
  return (
    <div data-testid="components-page" className="flex flex-col gap-4">
      <header>
        <h1 className="text-2xl font-semibold text-app-foreground">
          {t("components.title", { defaultValue: "Component Library" })}
        </h1>
        <p className="mt-1 text-sm text-app-muted-foreground">
          {t("components.subtitle", {
            defaultValue:
              "Components are indexed from the source root by their @libraryId header. Edit one to open the live preview.",
          })}
        </p>
      </header>
      <ComponentsCard />
    </div>
  );
}
