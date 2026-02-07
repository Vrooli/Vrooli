import { Layers, Server } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { useOverlayRouter } from '@/hooks/useOverlayRouter';
import { useAppsStore } from '@/state/appsStore';
import { useResourcesStore } from '@/state/resourcesStore';
import { resolveTabSwitcherShortcut, type ShortcutState } from '@/utils/tabSwitcherShortcut';
import './HomeView.css';

const formatCount = (value: number): string => {
  if (!Number.isFinite(value) || value < 0) {
    return '0';
  }
  if (value > 999) {
    return '999+';
  }
  return String(value);
};

export default function HomeView() {
  const appsCount = useAppsStore(state => state.apps.length);
  const appsLoadingInitial = useAppsStore(state => state.loadingInitial);
  const appsInitialized = useAppsStore(state => state.hasInitialized);
  const resourcesCount = useResourcesStore(state => state.resources.length);
  const resourcesLoading = useResourcesStore(state => state.loading);
  const resourcesInitialized = useResourcesStore(state => state.hasInitialized);
  const { openOverlay } = useOverlayRouter();

  const scenariosMeta = useMemo(() => {
    if (appsLoadingInitial && !appsInitialized) {
      return { label: 'Loading…', isLoading: true } as const;
    }
    if (!appsInitialized && appsCount === 0) {
      return { label: 'Unavailable', isLoading: false } as const;
    }
    if (appsCount === 0) {
      return { label: 'No scenarios yet', isLoading: false } as const;
    }
    return {
      label: `${formatCount(appsCount)} scenario${appsCount === 1 ? '' : 's'}`,
      isLoading: false,
    } as const;
  }, [appsCount, appsInitialized, appsLoadingInitial]);

  const resourcesMeta = useMemo(() => {
    if (resourcesLoading && !resourcesInitialized) {
      return { label: 'Loading…', isLoading: true } as const;
    }
    if (!resourcesInitialized && resourcesCount === 0) {
      return { label: 'Unavailable', isLoading: false } as const;
    }
    if (resourcesCount === 0) {
      return { label: 'No resources yet', isLoading: false } as const;
    }
    return {
      label: `${formatCount(resourcesCount)} resource${resourcesCount === 1 ? '' : 's'}`,
      isLoading: false,
    } as const;
  }, [resourcesCount, resourcesInitialized, resourcesLoading]);

  const [shortcut, setShortcut] = useState<ShortcutState | null>(null);

  useEffect(() => {
    setShortcut(resolveTabSwitcherShortcut());
  }, []);

  const handleOpenTabs = (segment: 'apps' | 'resources') => {
    openOverlay('tabs', {
      params: { segment },
    });
  };

  return (
    <div className="home-view">
      <section className="home-view__panel" aria-labelledby="home-view-title">
        <header className="home-view__header">
          <h1 id="home-view-title">App Monitor control room</h1>
          <p>
            Launch scenarios or inspect shared resources from the tabs hub.
            The bottom navigation keeps the switcher and status panels within thumb reach.
          </p>
        </header>

        <div className="home-view__actions" role="group" aria-label="Primary actions">
          <button type="button" onClick={() => handleOpenTabs('apps')}>
            <span className="home-view__actions-icon" aria-hidden>
              <Layers size={18} />
            </span>
            <div>
              <strong>Browse scenarios</strong>
              <span
                className="home-view__metric"
                aria-live="polite"
                aria-busy={scenariosMeta.isLoading}
              >
                {scenariosMeta.label}
              </span>
            </div>
          </button>
          <button type="button" onClick={() => handleOpenTabs('resources')}>
            <span className="home-view__actions-icon" aria-hidden>
              <Server size={18} />
            </span>
            <div>
              <strong>Inspect resources</strong>
              <span
                className="home-view__metric"
                aria-live="polite"
                aria-busy={resourcesMeta.isLoading}
              >
                {resourcesMeta.label}
              </span>
            </div>
          </button>
        </div>

        <footer className="home-view__footer" aria-label="Usage tips">
          <p>
            Tip: open the tabs dialog from anywhere with the bottom
            <span className="home-view__hint">Tabs</span> button
            {shortcut ? (
              <>
                {' '}or press <ShortcutChip shortcut={shortcut} />.
              </>
            ) : '.'}
          </p>
      </footer>
      </section>
    </div>
  );
}

function ShortcutChip({ shortcut }: { shortcut: ShortcutState }) {
  return (
    <span className="home-view__shortcut-chip">
      <span className="visually-hidden">{shortcut.description}</span>
      <span className="home-view__shortcut-visual" aria-hidden="true">
        {shortcut.keys.map((key, index) => (
          <span key={`${key}-${index}`} className="home-view__shortcut-group">
            <span className="home-view__shortcut-key">{key}</span>
            {index < shortcut.keys.length - 1 && (
              <span className="home-view__shortcut-plus" aria-hidden="true">+</span>
            )}
          </span>
        ))}
      </span>
    </span>
  );
}
