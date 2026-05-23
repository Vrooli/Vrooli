import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { cn } from "../../lib/utils";
import { ErrorState } from "../../components/ErrorState";
import { LoadingState } from "../../components/LoadingState";
import { useListDomains } from "./controllers/useGraphController";

export interface GraphFilterBarProps {
  scenario: string;
  /** Currently-selected domain keys. Empty set means "All". */
  selected: ReadonlySet<string>;
  onChange: (next: ReadonlySet<string>) => void;
}

export function GraphFilterBar({ scenario, selected, onChange }: GraphFilterBarProps) {
  const { t } = useTranslation();
  const domains = useListDomains({ scenario });

  if (domains.isPending) {
    return (
      <div data-testid={selectors.features.graph.filterBar.loading}>
        <LoadingState label={t(strings.shared.loading.label)} />
      </div>
    );
  }

  if (domains.isError) {
    return (
      <div data-testid={selectors.features.graph.filterBar.error}>
        <ErrorState
          title={t(strings.shared.error.title)}
          message={domains.error instanceof Error
            ? domains.error.message
            : String(domains.error)}
          retryLabel={t(strings.shared.error.retry)}
          onRetry={() => {
            void domains.refetch();
          }}
        />
      </div>
    );
  }

  const list = domains.data.domains;
  if (list.length === 0) {
    return (
      <p
        data-testid={selectors.features.graph.filterBar.empty}
        className="text-xs text-app-muted-foreground"
      >
        {t(strings.features.graph.filterBar.noDomains)}
      </p>
    );
  }

  const toggle = (key: string): void => {
    const next = new Set(selected);
    if (next.has(key)) {
      next.delete(key);
    } else {
      next.add(key);
    }
    onChange(next);
  };
  const clearAll = (): void => {
    onChange(new Set());
  };

  return (
    <div
      data-testid={selectors.features.graph.filterBar.root}
      role="group"
      aria-label={t(strings.features.graph.filterBar.label)}
      className="flex flex-wrap items-center gap-2"
    >
      <button
        type="button"
        data-testid={selectors.features.graph.filterBar.allChip}
        onClick={clearAll}
        aria-pressed={selected.size === 0}
        className={cn(
          "rounded-pill border px-3 py-1 text-xs font-medium transition-colors",
          selected.size === 0
            ? "border-app-primary bg-app-primary text-app-primary-foreground"
            : "border-app-border bg-app-surface-muted text-app-foreground hover:bg-app-surface",
        )}
      >
        {t(strings.features.graph.filterBar.allDomains)}
      </button>
      {list.map((domain) => {
        const isOn = selected.has(domain.name);
        return (
          <button
            type="button"
            key={domain.name}
            data-testid={selectors.features.graph.filterBar.chip({ key: domain.name })}
            onClick={() => toggle(domain.name)}
            aria-pressed={isOn}
            className={cn(
              "rounded-pill border px-3 py-1 text-xs font-medium transition-colors",
              isOn
                ? "border-app-primary bg-app-primary text-app-primary-foreground"
                : "border-app-border bg-app-surface-muted text-app-foreground hover:bg-app-surface",
            )}
          >
            {domain.name}
          </button>
        );
      })}
    </div>
  );
}
