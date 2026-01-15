import { useCallback, useEffect, useMemo } from 'react';
import type {
  ChangeEvent,
  KeyboardEvent as ReactKeyboardEvent,
  MouseEvent as ReactMouseEvent,
} from 'react';
import clsx from 'clsx';
import {
  AlertTriangle,
  ArrowLeft,
  ArrowRight,
  Bug,
  ExternalLink,
  Layers,
  Info,
  Loader2,
  Navigation2,
  Maximize2,
  Minimize2,
  MonitorSmartphone,
  Power,
  RefreshCw,
  RotateCcw,
  ScrollText,
  Wrench,
  Crosshair,
} from 'lucide-react';
import { useOverlayRouter } from '@/hooks/useOverlayRouter';
import { useDraggablePosition } from '@/hooks/useDraggablePosition';
import { useToolbarMenu, useMenuCoordinator, useMenuAutoFocus, useMenuOutsideClick } from '@/hooks/useToolbarMenu';
import { PREVIEW_UI } from './views/previewConstants';
import { AnchoredPopover } from './popover/AnchoredPopover';

import './AppPreviewToolbar.css';

export type AppPreviewToolbarPendingAction = 'start' | 'stop' | 'restart' | null;

export interface AppPreviewToolbarProps {
  canGoBack: boolean;
  canGoForward: boolean;
  onGoBack: () => void;
  onGoForward: () => void;
  onRefresh: () => void;
  isRefreshing: boolean;
  onOpenDetails: () => void;
  previewUrlInput: string;
  onPreviewUrlInputChange: (event: ChangeEvent<HTMLInputElement>) => void;
  onPreviewUrlInputBlur: () => void;
  onPreviewUrlInputKeyDown: (event: ReactKeyboardEvent<HTMLInputElement>) => void;
  onOpenInNewTab: (event: ReactMouseEvent<HTMLButtonElement>) => void;
  openPreviewTarget: string | null;
  urlStatusClass: string;
  urlStatusTitle: string;
  hasDetailsWarning: boolean;
  hasCurrentApp: boolean;
  isAppRunning: boolean;
  pendingAction: AppPreviewToolbarPendingAction;
  actionInProgress: boolean;
  toggleActionLabel: string;
  onToggleApp: () => void;
  restartActionLabel: string;
  onRestartApp: () => void;
  onToggleLogs: () => void;
  areLogsVisible: boolean;
  onReportIssue: () => void;
  appStatusLabel: string;
  isFullView: boolean;
  onToggleFullView: () => void;
  isDeviceEmulationActive: boolean;
  onToggleDeviceEmulation: () => void;
  canInspect: boolean;
  isInspecting: boolean;
  onToggleInspect: () => void;
  menuPortalContainer: HTMLElement | null;
  canOpenTabsOverlay: boolean;
  previewInteractionSignal: number;
  issueCaptureCount: number;
}

const AppPreviewToolbar = ({
  canGoBack,
  canGoForward,
  onGoBack,
  onGoForward,
  onRefresh,
  isRefreshing,
  onOpenDetails,
  previewUrlInput,
  onPreviewUrlInputChange,
  onPreviewUrlInputBlur,
  onPreviewUrlInputKeyDown,
  onOpenInNewTab,
  openPreviewTarget,
  urlStatusClass,
  urlStatusTitle,
  hasCurrentApp,
  isAppRunning,
  pendingAction,
  actionInProgress,
  toggleActionLabel,
  onToggleApp,
  restartActionLabel,
  onRestartApp,
  onToggleLogs,
  areLogsVisible,
  onReportIssue,
  appStatusLabel,
  hasDetailsWarning,
  isFullView,
  onToggleFullView,
  isDeviceEmulationActive,
  onToggleDeviceEmulation,
  canInspect,
  isInspecting,
  onToggleInspect,
  menuPortalContainer,
  canOpenTabsOverlay,
  previewInteractionSignal,
  issueCaptureCount,
}: AppPreviewToolbarProps) => {
  // Coordinate mutually-exclusive menus
  const { handleMenuOpenChange, closeAll: closeMenus, registerMenu } = useMenuCoordinator();

  // Create menu instances with consolidated hook
  const lifecycleMenu = useToolbarMenu({
    id: 'lifecycle',
    onOpenChange: handleMenuOpenChange,
  });

  const devMenu = useToolbarMenu({
    id: 'dev',
    onOpenChange: handleMenuOpenChange,
  });

  const navMenu = useToolbarMenu({
    id: 'nav',
    onOpenChange: handleMenuOpenChange,
  });

  useEffect(() => registerMenu('lifecycle', lifecycleMenu.close), [lifecycleMenu.close, registerMenu]);
  useEffect(() => registerMenu('dev', devMenu.close), [devMenu.close, registerMenu]);
  useEffect(() => registerMenu('nav', navMenu.close), [navMenu.close, registerMenu]);

  const { openOverlay } = useOverlayRouter();

  const captureBadgeCount = issueCaptureCount > 99 ? 99 : issueCaptureCount;
  const captureBadgeLabel = captureBadgeCount > 9 ? '9+' : captureBadgeCount.toString();
  const showCaptureBadge = issueCaptureCount > 0;
  const captureAriaLabel = showCaptureBadge
    ? `${issueCaptureCount} capture${issueCaptureCount === 1 ? '' : 's'} staged`
    : null;

  // Note: anchor measurement and flip logic live in the shared popover hook.

  const detailsButtonLabel = hasDetailsWarning
    ? 'Application details (localhost references detected)'
    : 'Application details';

  const fullscreenActionLabel = isFullView ? 'Exit full view' : 'Enter full view';
  const inspectModeDisabledReason = useMemo(() => {
    if (!hasCurrentApp) {
      return 'Select an application to inspect elements.';
    }
    if (!canInspect) {
      return 'Element inspection is unavailable for this preview.';
    }
    return null;
  }, [canInspect, hasCurrentApp]);

  const isBrowser = typeof document !== 'undefined';
  const portalHost = isBrowser ? (menuPortalContainer ?? document.body) : null;

  // Draggable toolbar positioning for fullscreen mode
  const floatingToolbar = useDraggablePosition({
    isActive: isFullView,
    storageKey: null, // No persistence for toolbar - resets each session
    defaultPosition: PREVIEW_UI.DEFAULT_FLOATING_POSITION,
    floatingMargin: PREVIEW_UI.FLOATING_MARGIN,
    onDragStart: closeMenus,
  });

  // Auto-focus first menu item when menus open (accessibility)
  useMenuAutoFocus(lifecycleMenu.isOpen, lifecycleMenu.firstItemRef);
  useMenuAutoFocus(devMenu.isOpen, devMenu.firstItemRef);
  useMenuAutoFocus(navMenu.isOpen, navMenu.firstItemRef);

  // Handle outside clicks to close all menus
  useMenuOutsideClick(
    [
      lifecycleMenu.menuRef,
      lifecycleMenu.popoverRef,
      lifecycleMenu.buttonRef,
      devMenu.menuRef,
      devMenu.popoverRef,
      devMenu.buttonRef,
      navMenu.menuRef,
      navMenu.popoverRef,
      navMenu.buttonRef,
    ],
    closeMenus,
    lifecycleMenu.isOpen || devMenu.isOpen || navMenu.isOpen,
  );

  useEffect(() => {
    if (!isFullView) {
      closeMenus();
    }
  }, [closeMenus, isFullView]);

  // Note: Old closeMenus callback removed - now provided by useMenuCoordinator

  useEffect(() => {
    if (previewInteractionSignal === 0) {
      return;
    }
    closeMenus();
  }, [closeMenus, previewInteractionSignal]);

  // Simplified toggle handlers - mutual exclusion handled by coordinator
  const handleToggleLifecycleMenu = useCallback(() => {
    lifecycleMenu.toggle();
  }, [lifecycleMenu]);

  const handleToggleDevMenu = useCallback(() => {
    devMenu.toggle();
  }, [devMenu]);

  const handleToggleNavMenu = useCallback(() => {
    navMenu.toggle();
  }, [navMenu]);

  const handleLifecycleAction = useCallback((action: 'toggle' | 'restart') => {
    if (action === 'toggle') {
      onToggleApp();
    } else {
      onRestartApp();
    }
    closeMenus();
  }, [closeMenus, onRestartApp, onToggleApp]);

  const handleToggleLogs = useCallback(() => {
    onToggleLogs();
    closeMenus();
  }, [closeMenus, onToggleLogs]);

  const handleReportIssue = useCallback(() => {
    onReportIssue();
    closeMenus();
  }, [closeMenus, onReportIssue]);

  const handleNavAction = useCallback((action: 'back' | 'forward' | 'refresh') => {
    if (action === 'back') {
      if (canGoBack) {
        onGoBack();
      }
    } else if (action === 'forward') {
      if (canGoForward) {
        onGoForward();
      }
    } else if (action === 'refresh') {
      if (!isRefreshing) {
        onRefresh();
      }
    }
    closeMenus();
  }, [canGoBack, canGoForward, closeMenus, isRefreshing, onGoBack, onGoForward, onRefresh]);

  const handleToggleFullscreen = useCallback(() => {
    onToggleFullView();
    closeMenus();
  }, [closeMenus, onToggleFullView]);

  const handleOpenTabsOverlay = useCallback(() => {
    closeMenus();
    openOverlay('tabs', {
      params: { segment: 'apps' },
    });
  }, [closeMenus, openOverlay]);

  return (
    <div
      ref={floatingToolbar.elementRef as React.RefObject<HTMLDivElement>}
      className={clsx(
        'preview-toolbar',
        isFullView && 'preview-toolbar--floating',
        isFullView && floatingToolbar.isDragging && 'preview-toolbar--dragging',
      )}
      style={floatingToolbar.floatingStyle}
      onPointerDown={floatingToolbar.pointerHandlers.onPointerDown}
      onPointerMove={floatingToolbar.pointerHandlers.onPointerMove}
      onPointerUp={floatingToolbar.pointerHandlers.onPointerUp}
      onPointerCancel={floatingToolbar.pointerHandlers.onPointerCancel}
      onClickCapture={floatingToolbar.handleClickCapture}
    >
      <div className="preview-toolbar__group preview-toolbar__group--left">
        {isFullView ? (
          <>
            <div
              className={clsx('preview-toolbar__menu', navMenu.isOpen && 'preview-toolbar__menu--open')}
              ref={navMenu.menuRef}
            >
              <button
                type="button"
                className={clsx(
                  'preview-toolbar__icon-btn',
                  navMenu.isOpen && 'preview-toolbar__icon-btn--active',
                )}
                ref={navMenu.buttonRef}
                onClick={handleToggleNavMenu}
                disabled={!canGoBack && !canGoForward && isRefreshing}
                aria-haspopup="menu"
                aria-expanded={navMenu.isOpen}
                aria-label="Navigation actions"
                title="Navigation actions"
              >
                <Navigation2 aria-hidden size={18} />
              </button>
            <AnchoredPopover
              isOpen={navMenu.isOpen}
              portalHost={portalHost}
              popoverRef={navMenu.popoverRef}
              style={navMenu.menuStyle}
              placement={navMenu.placement}
              className="preview-toolbar__menu-popover"
              role="menu"
            >
              <button
                type="button"
                role="menuitem"
                ref={navMenu.firstItemRef}
                className="preview-toolbar__menu-item"
                onClick={() => handleNavAction('back')}
                disabled={!canGoBack}
              >
                <ArrowLeft aria-hidden size={16} />
                <span>Go back</span>
              </button>
              <button
                type="button"
                role="menuitem"
                className="preview-toolbar__menu-item"
                onClick={() => handleNavAction('forward')}
                disabled={!canGoForward}
              >
                <ArrowRight aria-hidden size={16} />
                <span>Go forward</span>
              </button>
              <button
                type="button"
                role="menuitem"
                className="preview-toolbar__menu-item"
                onClick={() => handleNavAction('refresh')}
                disabled={isRefreshing}
              >
                <RefreshCw aria-hidden size={16} className={clsx({ spinning: isRefreshing })} />
                <span>Refresh</span>
              </button>
            </AnchoredPopover>
            </div>
            {isFullView && (
              <button
                type="button"
                className={clsx('preview-toolbar__icon-btn', 'preview-toolbar__icon-btn--secondary')}
                onClick={handleOpenTabsOverlay}
                disabled={!canOpenTabsOverlay}
                aria-label="Open tabs switcher"
                title="Open tabs switcher"
              >
                <Layers aria-hidden size={18} />
              </button>
            )}
          </>
        ) : (
          <>
            <button
              type="button"
              className="preview-toolbar__icon-btn"
              onClick={onGoBack}
              disabled={!canGoBack}
              aria-label={canGoBack ? 'Go back' : 'No previous page'}
              title={canGoBack ? 'Go back' : 'No previous page'}
            >
              <ArrowLeft aria-hidden size={18} />
            </button>
            <button
              type="button"
              className="preview-toolbar__icon-btn"
              onClick={onGoForward}
              disabled={!canGoForward}
              aria-label={canGoForward ? 'Go forward' : 'No forward page'}
              title={canGoForward ? 'Go forward' : 'No forward page'}
            >
              <ArrowRight aria-hidden size={18} />
            </button>
            <button
              type="button"
              className={clsx('preview-toolbar__icon-btn', 'preview-toolbar__icon-btn--refresh')}
              onClick={onRefresh}
              disabled={isRefreshing}
              aria-label={isRefreshing ? 'Refreshing application status' : 'Refresh application'}
              title={isRefreshing ? 'Refreshing...' : 'Refresh'}
            >
              <RefreshCw aria-hidden size={18} className={clsx({ spinning: isRefreshing })} />
            </button>
          </>
        )}
        <button
          type="button"
          className={clsx(
            'preview-toolbar__icon-btn',
            'preview-toolbar__details-btn--mobile',
            hasDetailsWarning && 'preview-toolbar__icon-btn--warning',
          )}
          onClick={onOpenDetails}
          disabled={!hasCurrentApp}
          aria-label={detailsButtonLabel}
          title={detailsButtonLabel}
        >
          {hasDetailsWarning ? (
            <AlertTriangle aria-hidden size={18} />
          ) : (
            <Info aria-hidden size={18} />
          )}
        </button>
      </div>
      <div className="preview-toolbar__title">
        <div
          className={clsx('preview-toolbar__url-wrapper', urlStatusClass)}
          title={urlStatusTitle}
        >
          <button
            type="button"
            className={clsx(
              'preview-toolbar__url-action-btn',
              hasDetailsWarning && 'preview-toolbar__url-action-btn--warning',
            )}
            onClick={onOpenDetails}
            disabled={!hasCurrentApp}
            aria-label={detailsButtonLabel}
            title={detailsButtonLabel}
          >
            {hasDetailsWarning ? (
              <AlertTriangle aria-hidden size={16} />
            ) : (
              <Info aria-hidden size={16} />
            )}
          </button>
          <input
            type="text"
            className="preview-toolbar__url-input"
            value={previewUrlInput}
            onChange={onPreviewUrlInputChange}
            onBlur={onPreviewUrlInputBlur}
            onKeyDown={onPreviewUrlInputKeyDown}
            placeholder="Enter preview URL"
            aria-label="Preview URL"
            autoComplete="off"
            spellCheck={false}
            inputMode="url"
          />
          <button
            type="button"
            className="preview-toolbar__url-action-btn"
            onClick={onOpenInNewTab}
            disabled={!openPreviewTarget}
            aria-label={openPreviewTarget ? 'Open preview in new tab' : 'Preview unavailable'}
            title={openPreviewTarget ? 'Open in new tab' : 'Preview unavailable'}
          >
            <ExternalLink aria-hidden size={16} />
          </button>
        </div>
      </div>
      <div className="preview-toolbar__group preview-toolbar__group--right">
        <div
          className={clsx('preview-toolbar__menu', lifecycleMenu.isOpen && 'preview-toolbar__menu--open')}
          ref={lifecycleMenu.menuRef}
        >
          <button
            type="button"
            className={clsx(
              'preview-toolbar__icon-btn',
              isAppRunning && 'preview-toolbar__icon-btn--danger',
              (pendingAction === 'start' || pendingAction === 'stop') && 'preview-toolbar__icon-btn--waiting',
              lifecycleMenu.isOpen && 'preview-toolbar__icon-btn--active',
            )}
            ref={lifecycleMenu.buttonRef}
            onClick={handleToggleLifecycleMenu}
            disabled={!hasCurrentApp || actionInProgress}
            aria-haspopup="menu"
            aria-expanded={lifecycleMenu.isOpen}
            aria-label={hasCurrentApp ? `Lifecycle actions (${appStatusLabel})` : 'Lifecycle actions unavailable'}
            title={hasCurrentApp ? `Lifecycle actions (${appStatusLabel})` : 'Lifecycle actions unavailable'}
          >
            {(pendingAction === 'start' || pendingAction === 'stop') ? (
              <Loader2 aria-hidden size={18} className="spinning" />
            ) : (
              <Power aria-hidden size={18} />
            )}
          </button>
          <AnchoredPopover
            isOpen={lifecycleMenu.isOpen}
            portalHost={portalHost}
            popoverRef={lifecycleMenu.popoverRef}
            style={lifecycleMenu.menuStyle}
            placement={lifecycleMenu.placement}
            className="preview-toolbar__menu-popover"
            role="menu"
          >
            <button
              type="button"
              role="menuitem"
              ref={lifecycleMenu.firstItemRef}
              className="preview-toolbar__menu-item"
              onClick={() => handleLifecycleAction('toggle')}
              disabled={!hasCurrentApp || actionInProgress}
            >
              <Power aria-hidden size={16} />
              <span>{toggleActionLabel}</span>
            </button>
            <button
              type="button"
              role="menuitem"
              className="preview-toolbar__menu-item"
              onClick={() => handleLifecycleAction('restart')}
              disabled={!hasCurrentApp || !isAppRunning || actionInProgress || pendingAction === 'restart'}
            >
              {pendingAction === 'restart' ? (
                <Loader2 aria-hidden size={16} className="spinning" />
              ) : (
                <RotateCcw aria-hidden size={16} />
              )}
              <span>{restartActionLabel}</span>
            </button>
          </AnchoredPopover>
        </div>
        <div
          className={clsx('preview-toolbar__menu', devMenu.isOpen && 'preview-toolbar__menu--open')}
          ref={devMenu.menuRef}
        >
          <button
            type="button"
            className={clsx(
              'preview-toolbar__icon-btn',
              'preview-toolbar__icon-btn--dev',
              devMenu.isOpen && 'preview-toolbar__icon-btn--active',
            )}
            ref={devMenu.buttonRef}
            onClick={handleToggleDevMenu}
            disabled={!hasCurrentApp}
            aria-haspopup="menu"
            aria-expanded={devMenu.isOpen}
            aria-label={captureAriaLabel ? `Developer actions (${captureAriaLabel})` : 'Developer actions'}
            title={captureAriaLabel ? `Developer actions (${captureAriaLabel})` : 'Developer actions'}
          >
            <Wrench aria-hidden size={18} />
            {showCaptureBadge && (
              <span className="preview-toolbar__badge" aria-hidden>{captureBadgeLabel}</span>
            )}
          </button>
          <AnchoredPopover
            isOpen={devMenu.isOpen}
            portalHost={portalHost}
            popoverRef={devMenu.popoverRef}
            style={devMenu.menuStyle}
            placement={devMenu.placement}
            className="preview-toolbar__menu-popover"
            role="menu"
          >
            <button
              type="button"
              role="menuitem"
              ref={devMenu.firstItemRef}
              className="preview-toolbar__menu-item"
              onClick={handleToggleFullscreen}
              disabled={!hasCurrentApp}
            >
              {isFullView ? (
                <Minimize2 aria-hidden size={16} />
              ) : (
                <Maximize2 aria-hidden size={16} />
              )}
              <span>{fullscreenActionLabel}</span>
            </button>
            <button
              type="button"
              role="menuitem"
              className="preview-toolbar__menu-item"
              onClick={onToggleDeviceEmulation}
              disabled={!hasCurrentApp}
            >
              <MonitorSmartphone aria-hidden size={16} />
              <span>{isDeviceEmulationActive ? 'Hide emulator' : 'Show emulator'}</span>
            </button>
            <button
              type="button"
              role="menuitem"
              className={clsx(
                'preview-toolbar__menu-item',
                isInspecting && 'preview-toolbar__menu-item--active',
                !hasCurrentApp || !canInspect ? 'preview-toolbar__menu-item--with-note' : undefined,
              )}
              onClick={() => {
                closeMenus();
                onToggleInspect();
              }}
              aria-pressed={isInspecting}
              aria-disabled={(!hasCurrentApp || !canInspect) || undefined}
              disabled={!hasCurrentApp || !canInspect}
              title={( !hasCurrentApp || !canInspect ) && inspectModeDisabledReason ? inspectModeDisabledReason : (isInspecting ? 'Exit inspect mode' : 'Inspect element')}
            >
              <span className="preview-toolbar__menu-item-label">
                <Crosshair aria-hidden size={16} />
                <span>{isInspecting ? 'Exit inspect mode' : 'Inspect element'}</span>
              </span>
              {(!hasCurrentApp || !canInspect) && inspectModeDisabledReason && (
                <span className="preview-toolbar__menu-item-note">{inspectModeDisabledReason}</span>
              )}
            </button>
            <button
              type="button"
              role="menuitem"
              className={clsx('preview-toolbar__menu-item', areLogsVisible && 'preview-toolbar__menu-item--active')}
              onClick={handleToggleLogs}
              aria-pressed={areLogsVisible}
              disabled={!hasCurrentApp}
            >
              <ScrollText aria-hidden size={16} />
              <span>{areLogsVisible ? 'Hide logs' : 'Show logs'}</span>
            </button>
            <button
              type="button"
              role="menuitem"
              className="preview-toolbar__menu-item"
              onClick={handleReportIssue}
              disabled={!hasCurrentApp}
            >
              <span className="preview-toolbar__menu-item-label">
                <Bug aria-hidden size={16} />
                <span>Report an issue</span>
              </span>
              {showCaptureBadge && (
                <span className="preview-toolbar__menu-item-badge" aria-hidden>{captureBadgeLabel}</span>
              )}
            </button>
          </AnchoredPopover>
        </div>
      </div>
    </div>
  );
};

export default AppPreviewToolbar;
