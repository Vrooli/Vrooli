import { selectors } from "../consts/selectors";
import { CatalogPage } from "./CatalogPage";

/**
 * The home page is the catalog.
 *
 * `experience/pages/catalog.json` declares `/` as its route and calls it "the
 * entry point and the discovery surface", which is the right call for a product
 * whose catalog *is* the product: the first thing an operator should see is the
 * whole style space, not a dashboard summarising it.
 */
export function DashboardPage() {
  return (
    // A plain div, not a labelled section. The catalog below is already a
    // labelled region, and nesting a second one that points at the same heading
    // gives the page two landmarks claiming one name — which axe reports and a
    // screen-reader user hears as a duplicate.
    <div data-testid={selectors.pages.dashboard} className="flex flex-col gap-4">
      <CatalogPage />
    </div>
  );
}
