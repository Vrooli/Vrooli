import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { assets, uiText } from "./workflowData";

export function InventoryPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.inventory}
      aria-labelledby="inventory-heading"
      className="flex flex-col gap-5"
    >
      <div>
        <h2 id="inventory-heading" className="text-3xl font-semibold">
          {t(strings.pages.inventory.title)}
        </h2>
        <p className="mt-2 max-w-3xl text-sm text-app-muted-foreground">
          {t(strings.pages.inventory.description)}
        </p>
      </div>

      <div className="grid gap-3 md:grid-cols-4">
        {(["Case", "Flow", "Action", "Seed"] as const).map((kind) => (
          <article key={kind} className="rounded-panel border border-app-border bg-app-surface p-4">
            <p className="text-xs font-semibold uppercase tracking-normal text-app-muted-foreground">
              {`${kind}${uiText.inventory.countSuffix}`}
            </p>
            <p className="mt-2 text-2xl font-semibold">
              {assets.filter((asset) => asset.kind === kind).length}
            </p>
          </article>
        ))}
      </div>

      <section className="rounded-panel border border-app-border bg-app-surface p-4">
        <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <h3 className="text-lg font-semibold">{uiText.inventory.catalog}</h3>
          <div className="flex flex-wrap gap-2 text-xs">
            {uiText.inventory.tabLabels.map((label) => (
              <button
                key={label}
                type="button"
                className="rounded-md border border-app-border px-3 py-1.5 font-medium text-app-muted-foreground hover:text-app-foreground"
              >
                {label}
              </button>
            ))}
          </div>
        </div>
        <div className="mt-4 overflow-x-auto">
          <table data-testid={selectors.workflow.assetTable} className="min-w-full text-left text-sm">
            <thead className="text-xs uppercase text-app-muted-foreground">
              <tr>
                {uiText.inventory.headers.map((header) => (
                  <th key={header} className="px-2 py-2 font-semibold">
                    {header}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {assets.map((asset) => (
                <tr key={asset.path} className="border-t border-app-border">
                  <td className="px-2 py-3">{asset.kind}</td>
                  <td className="px-2 py-3 font-medium">{asset.name}</td>
                  <td className="px-2 py-3 text-xs text-app-muted-foreground">{asset.path}</td>
                  <td className="px-2 py-3">{asset.requirements}</td>
                  <td className="px-2 py-3">{asset.safety}</td>
                  <td className="px-2 py-3">{asset.status}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </section>
  );
}
