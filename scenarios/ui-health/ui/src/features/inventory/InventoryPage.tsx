import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

export function InventoryPage() {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.pages.inventory}
      aria-labelledby="inventory-heading"
      className="flex flex-col gap-4"
    >
      <header className="flex flex-col gap-1">
        <h2 id="inventory-heading" className="text-2xl font-semibold tracking-tight">
          {t(strings.pages.inventory.title)}
        </h2>
        <p className="text-sm text-app-muted-foreground">{t(strings.pages.inventory.description)}</p>
      </header>
      <div className="rounded-panel border border-app-border bg-app-surface p-6 text-sm text-app-muted-foreground">
        {t(strings.pages.inventory.empty)}
      </div>
    </section>
  );
}
