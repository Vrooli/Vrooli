import { ComponentsPage } from "./ComponentsPage";

/**
 * Library is the asset index. It deliberately renders the existing
 * ComponentsPage rather than forking filtering, preview or coverage logic;
 * the page's visible title still says "Components" until its owner renames it.
 */
export function LibraryPage() {
  return (
    <div data-testid="library-page">
      <ComponentsPage />
    </div>
  );
}
