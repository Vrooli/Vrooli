import { useCallback, useEffect, useMemo } from 'react';
import { Outlet } from 'react-router-dom';
import { Layers, MoreHorizontal } from 'lucide-react';
import clsx from 'clsx';
import { useAppsStore } from '@/state/appsStore';
import { useResourcesStore } from '@/state/resourcesStore';
import TabSwitcherDialog from '@/components/tabSwitcher/TabSwitcherDialog';
import ActionsDialog from '@/components/actions/ActionsDialog';
import { useOverlayRouter } from '@/hooks/useOverlayRouter';
import './Shell.css';

type ShellProps = {
  isConnected: boolean;
};

const NAV_BADGE_DIGITS = 3;

export default function Shell({ isConnected }: ShellProps) {
  const appsCount = useAppsStore(state => state.apps.length);
  const resourcesCount = useResourcesStore(state => state.resources.length);
  const { overlay: activeOverlay, openOverlay, closeOverlay } = useOverlayRouter();

  const totalSurfaceCount = useMemo(() => {
    const count = appsCount + resourcesCount;
    if (Number.isNaN(count) || count < 0) {
      return 0;
    }
    return count;
  }, [appsCount, resourcesCount]);

  const formattedCount = useMemo(() => {
    if (totalSurfaceCount <= 0) {
      return '0';
    }
    if (totalSurfaceCount >= 10 ** NAV_BADGE_DIGITS) {
      return `${10 ** NAV_BADGE_DIGITS - 1}+`;
    }
    return String(totalSurfaceCount);
  }, [totalSurfaceCount]);

  const anyOverlayOpen = activeOverlay !== null;

  useEffect(() => {
    if (!anyOverlayOpen) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        closeOverlay();
      }
    };

    document.addEventListener('keydown', handleKeyDown);

    const originalOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = originalOverflow;
    };
  }, [anyOverlayOpen, closeOverlay]);

  const handleToggleTabs = useCallback(() => {
    if (activeOverlay === 'tabs') {
      closeOverlay({ preserve: ['segment'] });
    } else {
      openOverlay('tabs');
    }
  }, [activeOverlay, closeOverlay, openOverlay]);

  const handleToggleActions = useCallback(() => {
    if (activeOverlay === 'actions') {
      closeOverlay();
    } else {
      openOverlay('actions');
    }
  }, [activeOverlay, closeOverlay, openOverlay]);

  return (
    <div className="shell">
      <div className="shell__content">
        <Outlet />
      </div>

      <nav className="shell__bottom-nav" aria-label="App Monitor navigation">
        <button
          type="button"
          className={clsx('shell__nav-btn', activeOverlay === 'tabs' && 'shell__nav-btn--active')}
          onClick={handleToggleTabs}
          aria-pressed={activeOverlay === 'tabs'}
          aria-haspopup="dialog"
        >
          <span className="shell__nav-icon" aria-hidden>
            <Layers size={20} />
          </span>
          <span className="shell__nav-label">Tabs</span>
          <span className="shell__nav-badge" aria-hidden>
            {formattedCount}
          </span>
        </button>

        <button
          type="button"
          className={clsx('shell__nav-btn', activeOverlay === 'actions' && 'shell__nav-btn--active')}
          onClick={handleToggleActions}
          aria-pressed={activeOverlay === 'actions'}
          aria-haspopup="dialog"
        >
          <span className="shell__nav-icon" aria-hidden>
            <MoreHorizontal size={20} />
          </span>
          <span className="shell__nav-label">More</span>
        </button>
      </nav>

      {activeOverlay === 'tabs' && (
        <div className="shell__overlay" role="dialog" aria-modal="true" aria-label="Tab switcher">
          <div className="shell__overlay-content shell__overlay-content--wide">
            <TabSwitcherDialog />
          </div>
        </div>
      )}

      {activeOverlay === 'actions' && (
        <div className="shell__overlay" role="dialog" aria-modal="true" aria-label="System actions">
          <div className="shell__overlay-content">
            <ActionsDialog isConnected={isConnected} />
          </div>
        </div>
      )}
    </div>
  );
}
