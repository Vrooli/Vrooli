import { useState } from "react";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { HistoryPanel } from "../features/history/HistoryPanel";
import { SearchPanel, type ReplayRequest } from "../features/search/SearchPanel";
import { type SearchMode } from "../lib/searchHistory";
import { useTranslation } from "../i18n";

/**
 * Search page. Composes the unified search surface with the recent-search
 * history sidebar; selecting a history entry replays it through the panel.
 */
export function SearchPage() {
  const { t } = useTranslation();
  const [replay, setReplay] = useState<ReplayRequest | null>(null);

  const handleReplay = (query: string, mode: SearchMode) => {
    setReplay((prev) => ({ query, mode, nonce: (prev?.nonce ?? 0) + 1 }));
  };

  return (
    <section
      data-testid={selectors.pages.search}
      aria-labelledby="search-heading"
      className="flex flex-col gap-6"
    >
      <div className="flex flex-col gap-1">
        <h2 id="search-heading" className="text-2xl font-semibold">
          {t(strings.pages.search.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.search.description)}</p>
      </div>

      <div className="grid gap-6 lg:grid-cols-[1fr_18rem]">
        <SearchPanel replay={replay} />
        <HistoryPanel onReplay={handleReplay} />
      </div>
    </section>
  );
}
