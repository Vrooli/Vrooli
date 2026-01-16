import type { App, Resource } from '@/types';
import type { AppSortOption } from '@/hooks/useAppCatalog';
import type { ResourceSortOption } from '@/hooks/useResourcesCatalog';
import {
  AppGrid,
  ResourceGrid,
  Section,
  SortControl,
  type SortOption,
} from './TabSwitcherCards';

export function AppsSection({
  showHistory,
  recentApps,
  apps,
  normalizedSearch,
  sortOption,
  onSortChange,
  onSelect,
  isLoading,
  onRetry,
  errorMessage,
  isRefreshing,
  sortOptions,
}: {
  showHistory: boolean;
  recentApps: App[];
  apps: App[];
  normalizedSearch: string;
  sortOption: AppSortOption;
  onSortChange: (value: string) => void;
  onSelect: (app: App, options?: { autoSelected?: boolean; navigationId?: string }) => void;
  isLoading?: boolean;
  onRetry?: () => void;
  errorMessage?: string | null;
  isRefreshing?: boolean;
  sortOptions: Array<SortOption<AppSortOption>>;
}) {
  return (
    <>
      {showHistory && (
        <Section
          title="Recently opened"
          description="Your most visited scenarios."
        >
          <AppGrid apps={recentApps} onSelect={onSelect} />
        </Section>
      )}
      <Section
        title={normalizedSearch ? 'Search results' : 'All scenarios'}
        description={normalizedSearch ? undefined : 'Browse the full catalog alphabetically or by status.'}
        actions={(
          <SortControl
            label="Sort scenarios"
            value={sortOption}
            options={sortOptions}
            onChange={onSortChange}
          />
        )}
      >
        <AppGrid
          apps={apps}
          onSelect={onSelect}
          emptyMessage="No scenarios match your search."
          isLoading={isLoading}
          onRetry={onRetry}
          errorMessage={errorMessage}
          isRefreshing={isRefreshing}
        />
      </Section>
    </>
  );
}

export function ResourcesSection({
  resources,
  normalizedSearch,
  sortOption,
  onSortChange,
  onSelect,
  isLoading,
  sortOptions,
}: {
  resources: Resource[];
  normalizedSearch: string;
  sortOption: ResourceSortOption;
  onSortChange: (value: string) => void;
  onSelect: (resource: Resource) => void;
  isLoading?: boolean;
  sortOptions: Array<SortOption<ResourceSortOption>>;
}) {
  return (
    <Section
      title={normalizedSearch ? 'Search results' : 'All resources'}
      description={normalizedSearch ? undefined : 'Operational tooling and shared services.'}
      actions={(
        <SortControl
          label="Sort resources"
          value={sortOption}
          options={sortOptions}
          onChange={onSortChange}
        />
      )}
    >
      <ResourceGrid
        resources={resources}
        onSelect={onSelect}
        emptyMessage="No resources match your search."
        isLoading={isLoading}
      />
    </Section>
  );
}
