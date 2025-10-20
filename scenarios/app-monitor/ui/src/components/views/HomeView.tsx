import { Layers, MousePointerClick, Server } from 'lucide-react';
import { useMemo } from 'react';
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

export default function HomeView() {
  const appsCount = useAppsStore(state => state.apps.length);
  const resourcesCount = useResourcesStore(state => state.resources.length);
  const tabsCount = useBrowserTabsStore(state => state.tabs.length);
  const historyCount = useBrowserTabsStore(state => state.history.length);
  const { openOverlay } = useOverlayRouter();

  const webSummary = useMemo(() => {
    if (tabsCount === 0 && historyCount === 0) {
      return 'No saved web sessions yet.';
    }
    if (tabsCount > 0) {
      return `${formatCount(tabsCount)} active tab${tabsCount === 1 ? '' : 's'}.`;
    }
    return `${formatCount(historyCount)} archived session${historyCount === 1 ? '' : 's'}.`;
  }, [tabsCount, historyCount]);

  const handleOpenTabs = (segment: 'apps' | 'resources' | 'web') => {
    openOverlay('tabs', {
      params: { segment },
    });
  };

  return (
    <div className="home-view">
      <section className="home-view__panel" aria-labelledby="home-view-title">
        <header className="home-view__header">
          <h1 id="home-view-title">App Monitor, streamlined</h1>
          <p>
            Launch scenario previews, inspect resources, or reopen web sessions without the legacy sidebar.
            Use the bottom navigation to bring critical surfaces into focus.
          </p>
        </header>

        <div className="home-view__actions" role="group" aria-label="Primary actions">
          <button type="button" onClick={() => handleOpenTabs('apps')}>
            <span className="home-view__actions-icon" aria-hidden>
              <Layers size={18} />
            </span>
            <div>
              <strong>Browse scenarios</strong>
              <span>{formatCount(appsCount)} available</span>
            </div>
          </button>
          <button type="button" onClick={() => handleOpenTabs('resources')}>
            <span className="home-view__actions-icon" aria-hidden>
              <Server size={18} />
            </span>
            <div>
              <strong>Inspect resources</strong>
              <span>{formatCount(resourcesCount)} reachable</span>
            </div>
          </button>
          <button type="button" onClick={() => handleOpenTabs('web')}>
            <span className="home-view__actions-icon" aria-hidden>
              <MousePointerClick size={18} />
            </span>
            <div>
              <strong>Resume web tabs</strong>
              <span>{webSummary}</span>
            </div>
          </button>
        </div>

        <footer className="home-view__footer" aria-label="Usage tips">
          <p>
            Tip: open the tabs dialog from anywhere with the bottom
            <span className="home-view__hint">Tabs</span> button or press
            <kbd>Ctrl</kbd>+<kbd>K</kbd>.
          </p>
        </footer>
      </section>
    </div>
  );
}
