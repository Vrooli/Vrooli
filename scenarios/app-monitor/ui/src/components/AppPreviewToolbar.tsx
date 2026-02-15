import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type {
  ChangeEvent,
  KeyboardEvent as ReactKeyboardEvent,
  MouseEvent as ReactMouseEvent,
  ReactNode,
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
import { useAnchoredPopover } from './popover/useAnchoredPopover';
import { formatPreviewUrlForDisplay } from '@/utils/previewUrl';
import type { PreviewSuggestionSection } from '@/utils/workspaceDiscovery';

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
  showDetailsButton?: boolean;
  showLifecycleMenu?: boolean;
  showDevMenu?: boolean;
  rightInlineActions?: ReactNode;
  buildUrlSuggestionSections?: (query: string) => PreviewSuggestionSection[];
  onSelectUrlSuggestion?: (value: string) => void;
  onOpenScenarioSelector?: () => void;
  scenarioSelectorLabel?: string;
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
  showDetailsButton = true,
  showLifecycleMenu = true,
  showDevMenu = true,
  rightInlineActions,
  buildUrlSuggestionSections,
  onSelectUrlSuggestion,
  onOpenScenarioSelector,
  scenarioSelectorLabel = 'Open scenario selector',
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
  const [isSmallScreen, setIsSmallScreen] = useState(() => (
    typeof window !== 'undefined' && typeof window.matchMedia === 'function'
      ? window.matchMedia('(max-width: 640px)').matches
      : false
  ));
  const shouldUseCompactNavigation = isFullView || isSmallScreen;
  const shouldShowLifecycleMenu = showLifecycleMenu && !isSmallScreen;

  const isBrowser = typeof document !== 'undefined';
  const portalHost = isBrowser ? (menuPortalContainer ?? document.body) : null;
  const urlWrapperRef = useRef<HTMLDivElement | null>(null);
  const urlSuggestionsPopoverRef = useRef<HTMLDivElement | null>(null);
  const closeUrlSuggestionsTimerRef = useRef<number | null>(null);
  const suggestionSelectionGuardRef = useRef(false);
  const urlSuggestionItemRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const [isUrlSuggestionsOpen, setIsUrlSuggestionsOpen] = useState(false);
  const [activeUrlSuggestionIndex, setActiveUrlSuggestionIndex] = useState(-1);
  const [isUrlInputEditing, setIsUrlInputEditing] = useState(false);
  const urlSuggestionSections = useMemo<PreviewSuggestionSection[]>(() => {
    if (!buildUrlSuggestionSections) {
      return [];
    }
    return buildUrlSuggestionSections(previewUrlInput);
  }, [buildUrlSuggestionSections, previewUrlInput]);
  const flattenedUrlSuggestions = useMemo(() => (
    urlSuggestionSections.flatMap((section) => section.items)
  ), [urlSuggestionSections]);
  const urlSuggestionIndexById = useMemo(() => {
    const entries = flattenedUrlSuggestions.map((item, index) => [item.id, index] as const);
    return new Map(entries);
  }, [flattenedUrlSuggestions]);
  const hasUrlSuggestionsContent = flattenedUrlSuggestions.length > 0 || Boolean(onOpenScenarioSelector);
  const urlSuggestionsCount = flattenedUrlSuggestions.length + (onOpenScenarioSelector ? 1 : 0);
  const displayedPreviewUrlInput = useMemo(() => {
    if (isUrlInputEditing) {
      return previewUrlInput;
    }
    return formatPreviewUrlForDisplay(previewUrlInput);
  }, [isUrlInputEditing, previewUrlInput]);

  useEffect(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
      return;
    }
    const mediaQuery = window.matchMedia('(max-width: 640px)');
    const handleChange = (event: MediaQueryListEvent) => {
      setIsSmallScreen(event.matches);
    };
    setIsSmallScreen(mediaQuery.matches);
    mediaQuery.addEventListener('change', handleChange);
    return () => {
      mediaQuery.removeEventListener('change', handleChange);
    };
  }, []);
  const urlSuggestionsPopover = useAnchoredPopover({
    isOpen: isUrlSuggestionsOpen && hasUrlSuggestionsContent,
    anchorRef: urlWrapperRef,
    popoverRef: urlSuggestionsPopoverRef,
    placement: 'bottom-end',
  });

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
      urlWrapperRef,
      urlSuggestionsPopoverRef,
    ],
    () => {
      closeMenus();
      setIsUrlSuggestionsOpen(false);
    },
    lifecycleMenu.isOpen || devMenu.isOpen || navMenu.isOpen || (isUrlSuggestionsOpen && hasUrlSuggestionsContent),
  );

  useEffect(() => {
    if (!isFullView) {
      closeMenus();
      setIsUrlSuggestionsOpen(false);
    }
  }, [closeMenus, isFullView]);

  // Note: Old closeMenus callback removed - now provided by useMenuCoordinator

  useEffect(() => {
    if (previewInteractionSignal === 0) {
      return;
    }
    closeMenus();
    setIsUrlSuggestionsOpen(false);
  }, [closeMenus, previewInteractionSignal]);

  useEffect(() => {
    if (lifecycleMenu.isOpen || devMenu.isOpen || navMenu.isOpen) {
      setIsUrlSuggestionsOpen(false);
    }
  }, [devMenu.isOpen, lifecycleMenu.isOpen, navMenu.isOpen]);

  useEffect(() => {
    if (!isUrlSuggestionsOpen || urlSuggestionsCount === 0) {
      setActiveUrlSuggestionIndex(-1);
      return;
    }
    setActiveUrlSuggestionIndex((current) => {
      if (current >= urlSuggestionsCount) {
        return urlSuggestionsCount - 1;
      }
      return current;
    });
  }, [isUrlSuggestionsOpen, urlSuggestionsCount]);

  useEffect(() => {
    if (!isUrlSuggestionsOpen || activeUrlSuggestionIndex < 0) {
      return;
    }
    const activeNode = urlSuggestionItemRefs.current[activeUrlSuggestionIndex];
    activeNode?.scrollIntoView({ block: 'nearest' });
  }, [activeUrlSuggestionIndex, isUrlSuggestionsOpen]);

  useEffect(() => {
    if (urlSuggestionItemRefs.current.length > urlSuggestionsCount) {
      urlSuggestionItemRefs.current = urlSuggestionItemRefs.current.slice(0, urlSuggestionsCount);
    }
  }, [urlSuggestionsCount]);

  useEffect(() => {
    return () => {
      if (closeUrlSuggestionsTimerRef.current !== null) {
        window.clearTimeout(closeUrlSuggestionsTimerRef.current);
      }
    };
  }, []);

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
      onRefresh();
    }
    closeMenus();
  }, [canGoBack, canGoForward, closeMenus, onGoBack, onGoForward, onRefresh]);

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

  const handleUrlInputFocus = useCallback(() => {
    suggestionSelectionGuardRef.current = false;
    setIsUrlInputEditing(true);
    if (!hasUrlSuggestionsContent) {
      return;
    }
    if (closeUrlSuggestionsTimerRef.current !== null) {
      window.clearTimeout(closeUrlSuggestionsTimerRef.current);
      closeUrlSuggestionsTimerRef.current = null;
    }
    setIsUrlSuggestionsOpen(true);
    setActiveUrlSuggestionIndex(-1);
  }, [hasUrlSuggestionsContent]);

  const handleUrlInputBlur = useCallback(() => {
    const shouldSkipBlurCommit = suggestionSelectionGuardRef.current;
    if (shouldSkipBlurCommit) {
      suggestionSelectionGuardRef.current = false;
    } else {
      onPreviewUrlInputBlur();
    }
    setIsUrlInputEditing(false);
    if (closeUrlSuggestionsTimerRef.current !== null) {
      window.clearTimeout(closeUrlSuggestionsTimerRef.current);
    }
    closeUrlSuggestionsTimerRef.current = window.setTimeout(() => {
      setIsUrlSuggestionsOpen(false);
      setActiveUrlSuggestionIndex(-1);
      closeUrlSuggestionsTimerRef.current = null;
    }, 110);
  }, [onPreviewUrlInputBlur]);

  const markSuggestionSelectionGuard = useCallback(() => {
    suggestionSelectionGuardRef.current = true;
  }, []);

  const handleUrlSuggestionSelect = useCallback((value: string) => {
    markSuggestionSelectionGuard();
    setIsUrlSuggestionsOpen(false);
    setActiveUrlSuggestionIndex(-1);
    onSelectUrlSuggestion?.(value);
  }, [markSuggestionSelectionGuard, onSelectUrlSuggestion]);

  const handleOpenScenarioSelectorClick = useCallback(() => {
    markSuggestionSelectionGuard();
    setIsUrlSuggestionsOpen(false);
    setActiveUrlSuggestionIndex(-1);
    onOpenScenarioSelector?.();
  }, [markSuggestionSelectionGuard, onOpenScenarioSelector]);

  const handleUrlSuggestionPointerDown = useCallback((event: React.PointerEvent<HTMLButtonElement>) => {
    event.preventDefault();
    markSuggestionSelectionGuard();
  }, [markSuggestionSelectionGuard]);

  const handleUrlSuggestionSelectByIndex = useCallback((index: number) => {
    if (index < 0 || index >= urlSuggestionsCount) {
      return;
    }
    const suggestion = flattenedUrlSuggestions[index];
    if (suggestion?.value) {
      handleUrlSuggestionSelect(suggestion.value);
      return;
    }
    if (onOpenScenarioSelector && index === flattenedUrlSuggestions.length) {
      handleOpenScenarioSelectorClick();
    }
  }, [
    flattenedUrlSuggestions,
    handleOpenScenarioSelectorClick,
    handleUrlSuggestionSelect,
    onOpenScenarioSelector,
    urlSuggestionsCount,
  ]);

  const isExplicitUrlInput = useCallback((value: string): boolean => {
    const trimmed = value.trim();
    if (!trimmed) {
      return false;
    }
    if (trimmed.startsWith('/') || trimmed.startsWith('//')) {
      return true;
    }
    if (/^[a-z][a-z0-9+.-]*:/i.test(trimmed)) {
      return true;
    }
    if (/^(localhost|127(?:\.\d{1,3}){3}|0\.0\.0\.0)(:\d+)?(?:[/?#].*)?$/i.test(trimmed)) {
      return true;
    }
    if (/^[a-z0-9.-]+:\d+(?:[/?#].*)?$/i.test(trimmed)) {
      return true;
    }
    return /^[^\s/]+\.[^\s/]+(?:[/?#].*)?$/i.test(trimmed);
  }, []);

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
        {shouldUseCompactNavigation ? (
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
              aria-label={isRefreshing ? 'Refreshing application status' : 'Refresh application'}
              title={isRefreshing ? 'Refreshing...' : 'Refresh'}
            >
              <RefreshCw aria-hidden size={18} className={clsx({ spinning: isRefreshing })} />
            </button>
          </>
        )}
        {showDetailsButton && (
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
        )}
      </div>
      <div className="preview-toolbar__title">
        <div
          className={clsx('preview-toolbar__url-wrapper', urlStatusClass)}
          title={urlStatusTitle}
          ref={urlWrapperRef}
        >
          {showDetailsButton && (
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
          )}
          <input
            type="text"
            className="preview-toolbar__url-input"
            value={displayedPreviewUrlInput}
            onChange={(event) => {
              onPreviewUrlInputChange(event);
              setActiveUrlSuggestionIndex(-1);
              if (hasUrlSuggestionsContent) {
                setIsUrlSuggestionsOpen(true);
              }
            }}
            onBlur={handleUrlInputBlur}
            onFocus={handleUrlInputFocus}
            onClick={handleUrlInputFocus}
            onKeyDown={(event) => {
              if (event.key === 'Escape') {
                if (isUrlSuggestionsOpen) {
                  event.preventDefault();
                  setIsUrlSuggestionsOpen(false);
                  setActiveUrlSuggestionIndex(-1);
                  return;
                }
              }
              if (hasUrlSuggestionsContent && (event.key === 'ArrowDown' || event.key === 'ArrowUp')) {
                event.preventDefault();
                setIsUrlSuggestionsOpen(true);
                setActiveUrlSuggestionIndex((current) => {
                  const startIndex = current < 0 ? (event.key === 'ArrowDown' ? -1 : 0) : current;
                  if (event.key === 'ArrowDown') {
                    return (startIndex + 1 + urlSuggestionsCount) % urlSuggestionsCount;
                  }
                  return (startIndex - 1 + urlSuggestionsCount) % urlSuggestionsCount;
                });
                return;
              }
              if (event.key === 'Enter' && isUrlSuggestionsOpen && activeUrlSuggestionIndex >= 0) {
                event.preventDefault();
                handleUrlSuggestionSelectByIndex(activeUrlSuggestionIndex);
                return;
              }
              if (event.key === 'Enter' && activeUrlSuggestionIndex < 0 && hasUrlSuggestionsContent) {
                const shouldPreferSuggestion = !isExplicitUrlInput(previewUrlInput);
                if (shouldPreferSuggestion) {
                  event.preventDefault();
                  if (flattenedUrlSuggestions.length > 0) {
                    handleUrlSuggestionSelectByIndex(0);
                    return;
                  }
                  if (onOpenScenarioSelector) {
                    handleOpenScenarioSelectorClick();
                    return;
                  }
                }
              }
              onPreviewUrlInputKeyDown(event);
              if (event.defaultPrevented) {
                setIsUrlSuggestionsOpen(false);
                setActiveUrlSuggestionIndex(-1);
                return;
              }
            }}
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
        <AnchoredPopover
          isOpen={isUrlSuggestionsOpen && hasUrlSuggestionsContent}
          portalHost={portalHost}
          popoverRef={urlSuggestionsPopoverRef}
          style={urlSuggestionsPopover.style}
          placement={urlSuggestionsPopover.placement}
          className="preview-toolbar__url-suggestions"
          role="listbox"
        >
          {urlSuggestionSections.map((section) => (
            <div key={section.id} className="preview-toolbar__url-section">
              <div className="preview-toolbar__url-section-heading">{section.label}</div>
              {section.items.map((suggestion) => {
                const optionIndex = urlSuggestionIndexById.get(suggestion.id) ?? -1;
                return (
                  <button
                    key={suggestion.id}
                    type="button"
                    className={clsx(
                      'preview-toolbar__url-suggestion',
                      activeUrlSuggestionIndex === optionIndex && 'preview-toolbar__url-suggestion--active',
                    )}
                    onPointerDown={handleUrlSuggestionPointerDown}
                    onMouseEnter={() => setActiveUrlSuggestionIndex(optionIndex)}
                    onClick={() => handleUrlSuggestionSelect(suggestion.value)}
                    ref={(node) => {
                      urlSuggestionItemRefs.current[optionIndex] = node;
                    }}
                    title={suggestion.label}
                  >
                    <span>{formatPreviewUrlForDisplay(suggestion.label)}</span>
                    {suggestion.detail && (
                      <span className="preview-toolbar__url-suggestion-detail">{suggestion.detail}</span>
                    )}
                  </button>
                );
              })}
            </div>
          ))}
          {onOpenScenarioSelector && (
            <button
              type="button"
              className={clsx(
                'preview-toolbar__url-selector',
                activeUrlSuggestionIndex === flattenedUrlSuggestions.length && 'preview-toolbar__url-suggestion--active',
              )}
              onPointerDown={handleUrlSuggestionPointerDown}
              onMouseEnter={() => setActiveUrlSuggestionIndex(flattenedUrlSuggestions.length)}
              onClick={handleOpenScenarioSelectorClick}
              ref={(node) => {
                urlSuggestionItemRefs.current[flattenedUrlSuggestions.length] = node;
              }}
            >
              <Layers aria-hidden size={14} />
              <span>{scenarioSelectorLabel}</span>
            </button>
          )}
        </AnchoredPopover>
      </div>
      <div className="preview-toolbar__group preview-toolbar__group--right">
        {shouldShowLifecycleMenu && (
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
        )}
        {showDevMenu && (
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
        )}
        {rightInlineActions}
      </div>
    </div>
  );
};

export default AppPreviewToolbar;
