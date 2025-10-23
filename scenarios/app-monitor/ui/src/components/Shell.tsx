import { useCallback, useEffect, useMemo } from 'react';
import { createPortal } from 'react-dom';
import { Outlet } from 'react-router-dom';
import { Layers, MoreHorizontal } from 'lucide-react';
import clsx from 'clsx';
import { useAppsStore } from '@/state/appsStore';
import { useResourcesStore } from '@/state/resourcesStore';
import TabSwitcherDialog from '@/components/tabSwitcher/TabSwitcherDialog';
import ActionsDialog from '@/components/actions/ActionsDialog';
import ResponsiveDialog from '@/components/dialog/ResponsiveDialog';
import { useOverlayRouter } from '@/hooks/useOverlayRouter';
import { useShellOverlayStore } from '@/state/shellOverlayStore';
import './Shell.css';

type ShellProps = {
  isConnected: boolean;
};

const NAV_BADGE_DIGITS = 3;

export default function Shell({ isConnected }: ShellProps) {
  const appsCount = useAppsStore(state => state.apps.length);
  const resourcesCount = useResourcesStore(state => state.resources.length);
  const { overlay: activeOverlay, openOverlay, closeOverlay } = useOverlayRouter();
  const overlayHost = useShellOverlayStore(state => state.overlayHost);

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

  const overlayTarget = overlayHost && overlayHost.isConnected ? overlayHost : null;

  const mountOverlay = (node: JSX.Element) => (
    overlayTarget ? createPortal(node, overlayTarget) : node
  );

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

  useEffect(() => {
    const listenerOptions = { capture: true } as const;
    const handleShortcut = (event: KeyboardEvent) => {
      if (event.defaultPrevented) {
        return;
      }

      const key = event.key?.toLowerCase();
      if (key !== 'k') {
        return;
      }

      if (!(event.ctrlKey || event.metaKey) || event.altKey) {
        return;
      }

      const target = event.target as HTMLElement | null;
      if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)) {
        return;
      }

      event.preventDefault();
      event.stopPropagation();

      if (activeOverlay === 'tabs') {
        closeOverlay({ preserve: ['segment'] });
      } else {
        openOverlay('tabs');
      }
    };

    window.addEventListener('keydown', handleShortcut, listenerOptions);
    return () => {
      window.removeEventListener('keydown', handleShortcut, listenerOptions);
    };
  }, [activeOverlay, closeOverlay, openOverlay]);

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
          aria-label="Tabs"
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
          aria-label="More"
        >
          <span className="shell__nav-icon" aria-hidden>
            <MoreHorizontal size={20} />
          </span>
          <span className="shell__nav-label">More</span>
        </button>
      </nav>

      {activeOverlay === 'tabs' && mountOverlay(
        <ResponsiveDialog
          isOpen
          ariaLabel="Tab switcher"
          size="wide"
          className="shell__dialog shell__dialog--wide"
        >
          <TabSwitcherDialog />
        </ResponsiveDialog>,
      )}

      {activeOverlay === 'actions' && mountOverlay(
        <ResponsiveDialog
          isOpen
          ariaLabel="System actions"
          className="shell__dialog"
        >
          <ActionsDialog isConnected={isConnected} />
        </ResponsiveDialog>,
      )}
    </div>
  );
}
