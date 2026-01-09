import { Layers, MousePointerClick, Server } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { useOverlayRouter } from '@/hooks/useOverlayRouter';
import { useAppsStore } from '@/state/appsStore';
import { useResourcesStore } from '@/state/resourcesStore';
import { useBrowserTabsStore } from '@/state/browserTabsStore';
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

type ShortcutState = {
  keys: string[];
  description: string;
};

const identifyDeviceShortcut = (): ShortcutState | null => {
  if (typeof navigator === 'undefined') {
    return null;
  }

  const nav = navigator as Navigator & { userAgentData?: { platform?: string } };
  const platform = `${nav.platform ?? ''} ${nav.userAgentData?.platform ?? ''}`.toLowerCase();
  const userAgent = (nav.userAgent ?? '').toLowerCase();
  const combined = `${platform} ${userAgent}`;
  const maxTouchPoints = typeof nav.maxTouchPoints === 'number' ? nav.maxTouchPoints : 0;

  const isIOS = /iphone|ipad|ipod/.test(combined) || (/mac/.test(platform) && maxTouchPoints > 1 && /ipad|iphone/.test(userAgent));
  const isAndroid = /android/.test(combined);
  const isMobile = isIOS || isAndroid || /mobile/.test(combined);

  if (isMobile) {
    return null;
  }

  const isMac = /mac/.test(combined) && !isIOS;

  if (isMac) {
    return {
      keys: ['⌘', 'K'],
      description: 'Command plus K',
    };
  }

  return {
    keys: ['Ctrl', 'K'],
    description: 'Control plus K',
  };
};

export default function HomeView() {
  const appsCount = useAppsStore(state => state.apps.length);
  const appsLoadingInitial = useAppsStore(state => state.loadingInitial);
  const appsInitialized = useAppsStore(state => state.hasInitialized);
  const resourcesCount = useResourcesStore(state => state.resources.length);
  const resourcesLoading = useResourcesStore(state => state.loading);
  const resourcesInitialized = useResourcesStore(state => state.hasInitialized);
  const tabsCount = useBrowserTabsStore(state => state.tabs.length);
  const historyCount = useBrowserTabsStore(state => state.history.length);
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

  const webSummary = useMemo(() => {
    if (tabsCount === 0 && historyCount === 0) {
      return 'No saved web sessions yet.';
    }
    if (tabsCount > 0) {
      return `${formatCount(tabsCount)} active tab${tabsCount === 1 ? '' : 's'}.`;
    }
    return `${formatCount(historyCount)} archived session${historyCount === 1 ? '' : 's'}.`;
  }, [tabsCount, historyCount]);

  const [shortcut, setShortcut] = useState<ShortcutState | null>(null);

  useEffect(() => {
    setShortcut(identifyDeviceShortcut());
  }, []);

  const handleOpenTabs = (segment: 'apps' | 'resources' | 'web') => {
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
            Launch scenarios, inspect shared resources, or pick up web sessions from the new tabs hub.
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
          <button type="button" onClick={() => handleOpenTabs('web')}>
            <span className="home-view__actions-icon" aria-hidden>
              <MousePointerClick size={18} />
            </span>
            <div>
              <strong>Resume web tabs</strong>
              <span className="home-view__metric">{webSummary}</span>
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
