import { useState } from "react";
import { SearchInput } from "@vrooli/react-component-library/SearchInput/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { searchConfiguration, type ConfigurationDescriptor } from "../../api/configuration";
import { useDebouncedValue } from "../../hooks/useDebouncedValue";
import { useManualQueryRecovery } from "../../hooks/useManualQueryRecovery";
import { i18n } from "../../i18n";

interface ConfigurationSearchPanelProps {
  target: string;
  onNavigate: (descriptor: ConfigurationDescriptor) => void;
}

export function ConfigurationSearchPanel({ target, onNavigate }: ConfigurationSearchPanelProps) {
  const [query, setQuery] = useState("");
  const { debounced: debouncedQuery } = useDebouncedValue(query, 180);
  const search = useManualQueryRecovery(["configuration-search", target, debouncedQuery], () => searchConfiguration(debouncedQuery, target));
  const results = search.data?.results ?? [];
  const hasResults = search.data !== undefined;
  const refresh = () => { void search.refetch(); };

  return (
    <section className="configuration-search" aria-labelledby="configuration-search-title">
      <h1 id="configuration-search-title" className="text-2xl font-semibold">{i18n.t("onboarding.configuration.title")}</h1>
      <p className="mt-2 text-muted">{i18n.t("onboarding.configuration.description")}</p>
      <div className="mt-5">
        <SearchInput
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={i18n.t("onboarding.configuration.placeholder")}
          aria-label={i18n.t("onboarding.configuration.searchLabel")}
          data-testid="configuration-search"
        />
      </div>
      <div className="mt-5" aria-live="polite">
        {search.isFetching && <p role="status">{i18n.t("onboarding.configuration.loading")}</p>}
        {search.isError && <div role="alert">
          <p>{i18n.t("onboarding.configuration.error")}</p>
          <Button type="button" variant="secondary" onClick={refresh} disabled={search.isFetching}>Retry search</Button>
        </div>}
        {!search.isError && hasResults && <Button type="button" variant="secondary" onClick={refresh} disabled={search.isFetching}>Refresh results</Button>}
        {search.isError && hasResults && results.length > 0 && <p role="status">Showing matching results from the last successful search.</p>}
        {!search.isFetching && !search.isError && hasResults && results.length === 0 && <p>{i18n.t("onboarding.configuration.empty")}</p>}
        {hasResults && results.length > 0 && (
          <ul className="space-y-3" data-testid="configuration-results">
            {results.map((descriptor) => (
              <li key={descriptor.id} className="rounded-lg border border-border bg-surface p-4">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <h2 className="font-medium">{descriptor.title}</h2>
                    <p className="mt-1 text-sm text-muted">{descriptor.purpose}</p>
                    {descriptor.prerequisites.length > 0 && <p className="mt-2 text-xs text-muted">{i18n.t("onboarding.configuration.requires", { values: descriptor.prerequisites.join(", ") })}</p>}
                  </div>
                  <Button type="button" variant="secondary" onClick={() => onNavigate(descriptor)} data-testid={`configuration-open-${descriptor.id}`}>
                    {i18n.t("onboarding.configuration.open")}
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}
