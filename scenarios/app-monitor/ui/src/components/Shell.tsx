import { useCallback, useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { Outlet, useLocation } from 'react-router-dom';
import { Layers, MoreHorizontal, Settings } from 'lucide-react';
import ErrorBoundary, { SectionErrorFallback } from '@/components/ErrorBoundary';
import clsx from 'clsx';
import TabSwitcherDialog from '@/components/tabSwitcher/TabSwitcherDialog';
import ActionsDialog from '@/components/actions/ActionsDialog';
import WorkspaceManagerDialog from '@/components/workspace/WorkspaceManagerDialog';
import ResponsiveDialog from '@/components/dialog/ResponsiveDialog';
import { useOverlayRouter } from '@/hooks/useOverlayRouter';
import { useKeyboardScope } from '@/hooks/useKeyboardScopes';
import { useShellOverlayStore } from '@/state/shellOverlayStore';
import { useDraggablePosition } from '@/hooks/useDraggablePosition';
import { isTabSwitcherShortcutEvent } from '@/utils/tabSwitcherShortcut';
import './Shell.css';

type ShellProps = {
  isConnected: boolean;
};

const DESKTOP_BREAKPOINT = 768;
const NAV_STORAGE_KEY = 'app-monitor:shell-nav-position';
const NAV_FLOATING_MARGIN = 24;
const NAV_BUTTON_SIZE = 44;
const NAV_BUTTON_GAP = 12;
const NAV_HORIZONTAL_PADDING = 32;
const NAV_HEIGHT = 64;

export default function Shell({ isConnected }: ShellProps) {
  const { overlay: activeOverlay, openOverlay, closeOverlay } = useOverlayRouter();
  const overlayHost = useShellOverlayStore(state => state.overlayHost);
  const location = useLocation();
  const normalizedPathname = location.pathname.replace(/\/+$/, '') || '/';
  const isWorkspaceRoute = normalizedPathname === '/apps/workspace';
  const navButtonCount = isWorkspaceRoute ? 3 : 2;
  const navWidth = (navButtonCount * NAV_BUTTON_SIZE)
    + (Math.max(0, navButtonCount - 1) * NAV_BUTTON_GAP)
    + NAV_HORIZONTAL_PADDING;

  // Detect desktop breakpoint for draggable nav
  const [isDesktop, setIsDesktop] = useState(() =>
    typeof window !== 'undefined' && window.matchMedia(`(min-width: ${DESKTOP_BREAKPOINT}px)`).matches,
  );

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    const mql = window.matchMedia(`(min-width: ${DESKTOP_BREAKPOINT}px)`);
    const handleChange = (e: MediaQueryListEvent) => setIsDesktop(e.matches);

    mql.addEventListener('change', handleChange);
    return () => mql.removeEventListener('change', handleChange);
  }, []);

  // Compute centered-bottom default position for nav
  const getDefaultNavPosition = useCallback(() => {
    if (typeof window === 'undefined') {
      return { x: NAV_FLOATING_MARGIN, y: NAV_FLOATING_MARGIN };
    }
    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;
    // Center horizontally, position near bottom
    const x = Math.max(NAV_FLOATING_MARGIN, (viewportWidth - navWidth) / 2);
    const y = Math.max(NAV_FLOATING_MARGIN, viewportHeight - NAV_HEIGHT - NAV_FLOATING_MARGIN);
    return { x, y };
  }, [navWidth]);

  // Draggable nav for desktop
  const draggable = useDraggablePosition({
    isActive: isDesktop,
    storageKey: NAV_STORAGE_KEY,
    defaultPosition: getDefaultNavPosition,
    floatingMargin: NAV_FLOATING_MARGIN,
    onDragStart: closeOverlay,
  });

  const anyOverlayOpen = activeOverlay !== null;

  const overlayTarget = overlayHost && overlayHost.isConnected ? overlayHost : null;

  const mountOverlay = (node: JSX.Element) => (
    overlayTarget ? createPortal(node, overlayTarget) : node
  );

  useKeyboardScope({
    id: 'shell-overlay-escape',
    priority: 500,
    enabled: anyOverlayOpen,
    onKeyDown: (event) => {
      if (event.key !== 'Escape') {
        return false;
      }
      event.preventDefault();
      closeOverlay();
      return true;
    },
  });

  useKeyboardScope({
    id: 'shell-tab-switcher-shortcut',
    priority: 100,
    onKeyDown: (event) => {
      if (event.defaultPrevented || !isTabSwitcherShortcutEvent(event)) {
        return false;
      }

      event.preventDefault();
      event.stopPropagation();

      if (activeOverlay === 'tabs') {
        closeOverlay({ preserve: ['segment'] });
      } else {
        openOverlay('tabs');
      }
      return true;
    },
  });

  useEffect(() => {
    if (activeOverlay !== 'workspace' || isWorkspaceRoute) {
      return;
    }
    closeOverlay();
  }, [activeOverlay, closeOverlay, isWorkspaceRoute]);

  useEffect(() => {
    if (!anyOverlayOpen) {
      return;
    }

    const originalOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    return () => {
      document.body.style.overflow = originalOverflow;
    };
  }, [anyOverlayOpen]);

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

  const handleToggleWorkspace = useCallback(() => {
    if (!isWorkspaceRoute) {
      return;
    }
    if (activeOverlay === 'workspace') {
      closeOverlay();
    } else {
      openOverlay('workspace');
    }
  }, [activeOverlay, closeOverlay, isWorkspaceRoute, openOverlay]);

  return (
    <div className="shell">
      <div className="shell__content">
        <ErrorBoundary>
          <Outlet />
        </ErrorBoundary>
      </div>

      <nav
        ref={draggable.elementRef as React.RefObject<HTMLElement>}
        className={clsx(
          'shell__bottom-nav',
          isDesktop && 'shell__bottom-nav--draggable',
          isDesktop && draggable.isDragging && 'shell__bottom-nav--dragging',
        )}
        style={draggable.floatingStyle}
        onPointerDown={draggable.pointerHandlers.onPointerDown}
        onPointerMove={draggable.pointerHandlers.onPointerMove}
        onPointerUp={draggable.pointerHandlers.onPointerUp}
        onPointerCancel={draggable.pointerHandlers.onPointerCancel}
        onClickCapture={draggable.handleClickCapture}
        aria-label="App Monitor navigation"
      >
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

        {isWorkspaceRoute && (
          <button
            type="button"
            className={clsx('shell__nav-btn', activeOverlay === 'workspace' && 'shell__nav-btn--active')}
            onClick={handleToggleWorkspace}
            aria-pressed={activeOverlay === 'workspace'}
            aria-haspopup="dialog"
            aria-label="Workspace"
          >
            <span className="shell__nav-icon" aria-hidden>
              <Settings size={20} />
            </span>
            <span className="shell__nav-label">Workspace</span>
          </button>
        )}
      </nav>

      {activeOverlay === 'tabs' && mountOverlay(
        <ResponsiveDialog
          isOpen
          ariaLabel="Tab switcher"
          size="wide"
          className="shell__dialog shell__dialog--wide"
        >
          <ErrorBoundary fallback={SectionErrorFallback}>
            <TabSwitcherDialog />
          </ErrorBoundary>
        </ResponsiveDialog>,
      )}

      {activeOverlay === 'actions' && mountOverlay(
        <ResponsiveDialog
          isOpen
          ariaLabel="System actions"
          className="shell__dialog"
        >
          <ErrorBoundary fallback={SectionErrorFallback}>
            <ActionsDialog isConnected={isConnected} />
          </ErrorBoundary>
        </ResponsiveDialog>,
      )}

      {activeOverlay === 'workspace' && isWorkspaceRoute && mountOverlay(
        <ResponsiveDialog
          isOpen
          ariaLabel="Workspace manager"
          className="shell__dialog shell__dialog--workspace-floating"
          mode="floating"
          draggable
          dragHandleSelector=".workspace-manager__header"
          floatingStorageKey="app-monitor:workspace-manager-dialog-position"
          floatingDefaultPosition={{ x: 24, y: 96 }}
          floatingMargin={24}
        >
          <ErrorBoundary fallback={SectionErrorFallback}>
            <WorkspaceManagerDialog onClose={() => closeOverlay()} />
          </ErrorBoundary>
        </ResponsiveDialog>,
      )}
    </div>
  );
}
