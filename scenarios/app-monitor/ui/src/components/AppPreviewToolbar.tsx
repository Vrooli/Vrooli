import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import type {
  ChangeEvent,
  CSSProperties,
  KeyboardEvent as ReactKeyboardEvent,
  MouseEvent as ReactMouseEvent,
  PointerEvent as ReactPointerEvent,
  MutableRefObject,
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

import './AppPreviewToolbar.css';

const MENU_OFFSET = 8;
const FLOATING_MARGIN = 12;
const DEFAULT_FLOATING_POSITION: Readonly<{ x: number; y: number }> = { x: 16, y: 16 };

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
  const [lifecycleMenuOpen, setLifecycleMenuOpen] = useState(false);
  const [devMenuOpen, setDevMenuOpen] = useState(false);
  const [navMenuOpen, setNavMenuOpen] = useState(false);
  const [floatingPosition, setFloatingPosition] = useState<{ x: number; y: number }>({
    x: DEFAULT_FLOATING_POSITION.x,
    y: DEFAULT_FLOATING_POSITION.y,
  });
  const [isDragging, setIsDragging] = useState(false);
  const lifecycleMenuRef = useRef<HTMLDivElement | null>(null);
  const devMenuRef = useRef<HTMLDivElement | null>(null);
  const navMenuRef = useRef<HTMLDivElement | null>(null);
  const lifecycleButtonRef = useRef<HTMLButtonElement | null>(null);
  const devButtonRef = useRef<HTMLButtonElement | null>(null);
  const navButtonRef = useRef<HTMLButtonElement | null>(null);
  const lifecyclePopoverRef = useRef<HTMLDivElement | null>(null);
  const devPopoverRef = useRef<HTMLDivElement | null>(null);
  const navPopoverRef = useRef<HTMLDivElement | null>(null);
  const lifecycleFirstItemRef = useRef<HTMLButtonElement | null>(null);
  const devFirstItemRef = useRef<HTMLButtonElement | null>(null);
  const navFirstItemRef = useRef<HTMLButtonElement | null>(null);
  const toolbarRef = useRef<HTMLDivElement | null>(null);
  const dragStateRef = useRef<{
    pointerId: number;
    startX: number;
    startY: number;
    offsetX: number;
    offsetY: number;
    width: number;
    height: number;
    pointerCaptured: boolean;
    dragging: boolean;
  } | null>(null);
  const suppressClickRef = useRef(false);
  const [lifecycleAnchorRect, setLifecycleAnchorRect] = useState<DOMRect | null>(null);
  const [devAnchorRect, setDevAnchorRect] = useState<DOMRect | null>(null);
  const [navAnchorRect, setNavAnchorRect] = useState<DOMRect | null>(null);
  const [lifecycleMenuStyle, setLifecycleMenuStyle] = useState<CSSProperties | undefined>(undefined);
  const [devMenuStyle, setDevMenuStyle] = useState<CSSProperties | undefined>(undefined);
  const [navMenuStyle, setNavMenuStyle] = useState<CSSProperties | undefined>(undefined);
  const { openOverlay } = useOverlayRouter();

  const captureBadgeCount = issueCaptureCount > 99 ? 99 : issueCaptureCount;
  const captureBadgeLabel = captureBadgeCount > 9 ? '9+' : captureBadgeCount.toString();
  const showCaptureBadge = issueCaptureCount > 0;
  const captureAriaLabel = showCaptureBadge
    ? `${issueCaptureCount} capture${issueCaptureCount === 1 ? '' : 's'} staged`
    : null;

  const updateLifecycleAnchor = useCallback(() => {
    const button = lifecycleButtonRef.current;
    if (!button) {
      setLifecycleAnchorRect(null);
      return null;
    }
    const rect = button.getBoundingClientRect();
    setLifecycleAnchorRect(rect);
    return rect;
  }, []);

  const updateDevAnchor = useCallback(() => {
    const button = devButtonRef.current;
    if (!button) {
      setDevAnchorRect(null);
      return null;
    }
    const rect = button.getBoundingClientRect();
    setDevAnchorRect(rect);
    return rect;
  }, []);

  const updateNavAnchor = useCallback(() => {
    const button = navButtonRef.current;
    if (!button) {
      setNavAnchorRect(null);
      return null;
    }
    const rect = button.getBoundingClientRect();
    setNavAnchorRect(rect);
    return rect;
  }, []);

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

  const computeMenuStyle = useCallback((anchorRect: DOMRect | null, popover: HTMLDivElement | null) => {
    if (!anchorRect) {
      return undefined;
    }

    const viewportWidth = typeof window !== 'undefined' ? window.innerWidth : undefined;
    const viewportHeight = typeof window !== 'undefined' ? window.innerHeight : undefined;
    const popRect = popover?.getBoundingClientRect();
    const popHeight = popRect?.height ?? 0;
    const popWidth = popRect?.width ?? 0;

    let top = anchorRect.bottom + MENU_OFFSET;
    if (typeof viewportHeight === 'number') {
      const spaceBelow = viewportHeight - anchorRect.bottom - MENU_OFFSET;
      const shouldPlaceBelow = popHeight === 0
        ? anchorRect.top + anchorRect.height / 2 < viewportHeight / 2
        : spaceBelow >= popHeight + FLOATING_MARGIN;
      if (!shouldPlaceBelow) {
        top = anchorRect.top - MENU_OFFSET - popHeight;
      }
      const maxTop = viewportHeight - FLOATING_MARGIN - popHeight;
      const minTop = FLOATING_MARGIN;
      top = Math.min(Math.max(top, minTop), maxTop);
    }

    let left = anchorRect.right;
    if (typeof viewportWidth === 'number') {
      const maxLeft = viewportWidth - FLOATING_MARGIN;
      const minLeft = popWidth > 0 ? popWidth + FLOATING_MARGIN : FLOATING_MARGIN;
      left = Math.min(Math.max(left, minLeft), maxLeft);
    }

    return {
      top: `${Math.round(top)}px`,
      left: `${Math.round(left)}px`,
      transform: 'translateX(-100%)',
    } satisfies CSSProperties;
  }, []);

  const scheduleMenuStyleUpdate = useCallback((
    anchorRect: DOMRect | null,
    popoverRef: MutableRefObject<HTMLDivElement | null>,
    setter: React.Dispatch<React.SetStateAction<CSSProperties | undefined>>,
  ) => {
    if (!anchorRect) {
      setter(undefined);
      return;
    }

    const apply = (popover: HTMLDivElement | null) => {
      setter(computeMenuStyle(anchorRect, popover));
    };

    apply(popoverRef.current);

    if (!popoverRef.current && typeof window !== 'undefined') {
      requestAnimationFrame(() => {
        apply(popoverRef.current);
      });
    }
  }, [computeMenuStyle]);

  const clampPosition = useCallback((x: number, y: number, size: { width: number; height: number }) => {
    if (typeof window === 'undefined') {
      return { x, y };
    }

    const { innerWidth, innerHeight } = window;
    const maxX = Math.max(FLOATING_MARGIN, innerWidth - size.width - FLOATING_MARGIN);
    const maxY = Math.max(FLOATING_MARGIN, innerHeight - size.height - FLOATING_MARGIN);
    const clampedX = Math.min(Math.max(x, FLOATING_MARGIN), maxX);
    const clampedY = Math.min(Math.max(y, FLOATING_MARGIN), maxY);
    return { x: clampedX, y: clampedY };
  }, []);

  const computeBottomRightPosition = useCallback(() => {
    if (typeof window === 'undefined') {
      return null;
    }

    const toolbar = toolbarRef.current;
    if (!toolbar) {
      return null;
    }

    const rect = toolbar.getBoundingClientRect();
    const desiredX = window.innerWidth - rect.width - FLOATING_MARGIN;
    const desiredY = window.innerHeight - rect.height - FLOATING_MARGIN;
    return clampPosition(desiredX, desiredY, { width: rect.width, height: rect.height });
  }, [clampPosition]);

  useEffect(() => {
    if (lifecycleMenuOpen && lifecycleFirstItemRef.current) {
      lifecycleFirstItemRef.current.focus();
    }
  }, [lifecycleMenuOpen]);

  useEffect(() => {
    if (devMenuOpen && devFirstItemRef.current) {
      devFirstItemRef.current.focus();
    }
  }, [devMenuOpen]);

  useEffect(() => {
    if (navMenuOpen && navFirstItemRef.current) {
      navFirstItemRef.current.focus();
    }
  }, [navMenuOpen]);

  useEffect(() => {
    if (!lifecycleMenuOpen) {
      setLifecycleMenuStyle(undefined);
      return;
    }

    const rect = updateLifecycleAnchor();
    scheduleMenuStyleUpdate(rect ?? lifecycleAnchorRect, lifecyclePopoverRef, setLifecycleMenuStyle);

    const handleResizeOrScroll = () => {
      const nextRect = updateLifecycleAnchor();
      scheduleMenuStyleUpdate(nextRect ?? lifecycleAnchorRect, lifecyclePopoverRef, setLifecycleMenuStyle);
    };

    window.addEventListener('resize', handleResizeOrScroll);
    window.addEventListener('scroll', handleResizeOrScroll, true);

    return () => {
      window.removeEventListener('resize', handleResizeOrScroll);
      window.removeEventListener('scroll', handleResizeOrScroll, true);
    };
  }, [lifecycleAnchorRect, lifecycleMenuOpen, scheduleMenuStyleUpdate, updateLifecycleAnchor]);

  useEffect(() => {
    if (!devMenuOpen) {
      setDevMenuStyle(undefined);
      return;
    }

    const rect = updateDevAnchor();
    scheduleMenuStyleUpdate(rect ?? devAnchorRect, devPopoverRef, setDevMenuStyle);

    const handleResizeOrScroll = () => {
      const nextRect = updateDevAnchor();
      scheduleMenuStyleUpdate(nextRect ?? devAnchorRect, devPopoverRef, setDevMenuStyle);
    };

    window.addEventListener('resize', handleResizeOrScroll);
    window.addEventListener('scroll', handleResizeOrScroll, true);

    return () => {
      window.removeEventListener('resize', handleResizeOrScroll);
      window.removeEventListener('scroll', handleResizeOrScroll, true);
    };
  }, [devAnchorRect, devMenuOpen, scheduleMenuStyleUpdate, updateDevAnchor]);

  useEffect(() => {
    if (!navMenuOpen) {
      setNavMenuStyle(undefined);
      return;
    }

    const rect = updateNavAnchor();
    scheduleMenuStyleUpdate(rect ?? navAnchorRect, navPopoverRef, setNavMenuStyle);

    const handleResizeOrScroll = () => {
      const nextRect = updateNavAnchor();
      scheduleMenuStyleUpdate(nextRect ?? navAnchorRect, navPopoverRef, setNavMenuStyle);
    };

    window.addEventListener('resize', handleResizeOrScroll);
    window.addEventListener('scroll', handleResizeOrScroll, true);

    return () => {
      window.removeEventListener('resize', handleResizeOrScroll);
      window.removeEventListener('scroll', handleResizeOrScroll, true);
    };
  }, [navAnchorRect, navMenuOpen, scheduleMenuStyleUpdate, updateNavAnchor]);

  useEffect(() => {
    if (!lifecycleMenuOpen && !devMenuOpen && !navMenuOpen) {
      return () => {};
    }

    const handlePointerDown = (event: globalThis.MouseEvent | globalThis.TouchEvent) => {
      const target = event.target as Node | null;
      if (lifecycleMenuOpen) {
        const menuContainer = lifecycleMenuRef.current;
        const popover = lifecyclePopoverRef.current;
        const button = lifecycleButtonRef.current;
        const isInside = Boolean(
          (menuContainer && target && menuContainer.contains(target)) ||
          (popover && target && popover.contains(target)) ||
          (button && target && button.contains(target))
        );
        if (!isInside) {
          setLifecycleMenuOpen(false);
          setLifecycleAnchorRect(null);
          setLifecycleMenuStyle(undefined);
        }
      }
      if (devMenuOpen) {
        const menuContainer = devMenuRef.current;
        const popover = devPopoverRef.current;
        const button = devButtonRef.current;
        const isInside = Boolean(
          (menuContainer && target && menuContainer.contains(target)) ||
          (popover && target && popover.contains(target)) ||
          (button && target && button.contains(target))
        );
        if (!isInside) {
          setDevMenuOpen(false);
          setDevAnchorRect(null);
          setDevMenuStyle(undefined);
        }
      }
      if (navMenuOpen) {
        const menuContainer = navMenuRef.current;
        const popover = navPopoverRef.current;
        const button = navButtonRef.current;
        const isInside = Boolean(
          (menuContainer && target && menuContainer.contains(target)) ||
          (popover && target && popover.contains(target)) ||
          (button && target && button.contains(target))
        );
        if (!isInside) {
          setNavMenuOpen(false);
          setNavAnchorRect(null);
          setNavMenuStyle(undefined);
        }
      }
    };

    const handleKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'Escape') {
        if (lifecycleMenuOpen || devMenuOpen || navMenuOpen) {
          event.stopPropagation();
          setLifecycleMenuOpen(false);
          setLifecycleAnchorRect(null);
          setLifecycleMenuStyle(undefined);
          setDevMenuOpen(false);
          setDevAnchorRect(null);
          setDevMenuStyle(undefined);
          setNavMenuOpen(false);
          setNavAnchorRect(null);
          setNavMenuStyle(undefined);
        }
      }
    };

    document.addEventListener('mousedown', handlePointerDown);
    document.addEventListener('touchstart', handlePointerDown);
    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('mousedown', handlePointerDown);
      document.removeEventListener('touchstart', handlePointerDown);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [devMenuOpen, lifecycleMenuOpen, navMenuOpen]);

  useEffect(() => {
    if (!isFullView) {
      setFloatingPosition({
        x: DEFAULT_FLOATING_POSITION.x,
        y: DEFAULT_FLOATING_POSITION.y,
      });
      setIsDragging(false);
      dragStateRef.current = null;
      setLifecycleMenuOpen(false);
      setDevMenuOpen(false);
      setNavMenuOpen(false);
      setLifecycleAnchorRect(null);
      setDevAnchorRect(null);
      setNavAnchorRect(null);
      setLifecycleMenuStyle(undefined);
      setDevMenuStyle(undefined);
      setNavMenuStyle(undefined);
      return;
    }

    const initialPosition = computeBottomRightPosition();
    if (initialPosition) {
      setFloatingPosition(initialPosition);
    }

    const handleResize = () => {
      const toolbar = toolbarRef.current;
      if (!toolbar) {
        return;
      }
      const rect = toolbar.getBoundingClientRect();
      setFloatingPosition(prev => {
        const next = clampPosition(prev.x, prev.y, { width: rect.width, height: rect.height });
        if (next.x === prev.x && next.y === prev.y) {
          return prev;
        }
        return next;
      });
    };

    handleResize();
    window.addEventListener('resize', handleResize);
    window.addEventListener('orientationchange', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      window.removeEventListener('orientationchange', handleResize);
    };
  }, [clampPosition, computeBottomRightPosition, isFullView]);

  const closeMenus = useCallback(() => {
    setLifecycleMenuOpen(false);
    setDevMenuOpen(false);
    setNavMenuOpen(false);
    setLifecycleAnchorRect(null);
    setDevAnchorRect(null);
    setNavAnchorRect(null);
    setLifecycleMenuStyle(undefined);
    setDevMenuStyle(undefined);
    setNavMenuStyle(undefined);
  }, []);

  useEffect(() => {
    if (previewInteractionSignal === 0) {
      return;
    }
    closeMenus();
  }, [closeMenus, previewInteractionSignal]);

  const handleToggleLifecycleMenu = useCallback(() => {
    const next = !lifecycleMenuOpen;
    setLifecycleMenuOpen(next);
    if (next) {
      const rect = updateLifecycleAnchor();
      scheduleMenuStyleUpdate(rect ?? lifecycleAnchorRect, lifecyclePopoverRef, setLifecycleMenuStyle);
    } else {
      setLifecycleAnchorRect(null);
      setLifecycleMenuStyle(undefined);
    }
    if (devMenuOpen) {
      setDevMenuOpen(false);
      setDevAnchorRect(null);
      setDevMenuStyle(undefined);
    }
    if (navMenuOpen) {
      setNavMenuOpen(false);
      setNavAnchorRect(null);
      setNavMenuStyle(undefined);
    }
  }, [devMenuOpen, lifecycleMenuOpen, navMenuOpen, lifecycleAnchorRect, scheduleMenuStyleUpdate, updateLifecycleAnchor]);

  const handleToggleDevMenu = useCallback(() => {
    const next = !devMenuOpen;
    setDevMenuOpen(next);
    if (next) {
      const rect = updateDevAnchor();
      scheduleMenuStyleUpdate(rect ?? devAnchorRect, devPopoverRef, setDevMenuStyle);
    } else {
      setDevAnchorRect(null);
      setDevMenuStyle(undefined);
    }
    if (lifecycleMenuOpen) {
      setLifecycleMenuOpen(false);
      setLifecycleAnchorRect(null);
      setLifecycleMenuStyle(undefined);
    }
    if (navMenuOpen) {
      setNavMenuOpen(false);
      setNavAnchorRect(null);
      setNavMenuStyle(undefined);
    }
  }, [devAnchorRect, devMenuOpen, lifecycleMenuOpen, navMenuOpen, scheduleMenuStyleUpdate, updateDevAnchor]);

  const handleToggleNavMenu = useCallback(() => {
    const next = !navMenuOpen;
    setNavMenuOpen(next);
    if (next) {
      const rect = updateNavAnchor();
      scheduleMenuStyleUpdate(rect ?? navAnchorRect, navPopoverRef, setNavMenuStyle);
    } else {
      setNavAnchorRect(null);
      setNavMenuStyle(undefined);
    }
    if (lifecycleMenuOpen) {
      setLifecycleMenuOpen(false);
      setLifecycleAnchorRect(null);
      setLifecycleMenuStyle(undefined);
    }
    if (devMenuOpen) {
      setDevMenuOpen(false);
      setDevAnchorRect(null);
      setDevMenuStyle(undefined);
    }
  }, [devMenuOpen, lifecycleMenuOpen, navAnchorRect, navMenuOpen, scheduleMenuStyleUpdate, updateNavAnchor]);

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

  const handlePointerDown = useCallback((event: ReactPointerEvent<HTMLDivElement>) => {
    if (!isFullView) {
      return;
    }

    if (event.pointerType === 'mouse' && event.button !== 0) {
      return;
    }

    const toolbar = toolbarRef.current;
    if (!toolbar) {
      return;
    }

    const rect = toolbar.getBoundingClientRect();
    dragStateRef.current = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      offsetX: event.clientX - rect.left,
      offsetY: event.clientY - rect.top,
      width: rect.width,
      height: rect.height,
      pointerCaptured: false,
      dragging: false,
    };
    setIsDragging(false);
  }, [isFullView]);

  const handlePointerMove = useCallback((event: ReactPointerEvent<HTMLDivElement>) => {
    const state = dragStateRef.current;
    if (!state || state.pointerId !== event.pointerId) {
      return;
    }

    const toolbar = toolbarRef.current;
    if (!toolbar) {
      return;
    }

    const deltaX = event.clientX - state.startX;
    const deltaY = event.clientY - state.startY;
    if (!state.dragging) {
      if (Math.abs(deltaX) + Math.abs(deltaY) < 6) {
        return;
      }
      state.dragging = true;
      setIsDragging(true);
      closeMenus();
      if (!state.pointerCaptured) {
        try {
          toolbar.setPointerCapture(event.pointerId);
          state.pointerCaptured = true;
        } catch (error) {
          state.pointerCaptured = false;
        }
      }
    }

    if (!state.dragging) {
      return;
    }

    event.preventDefault();
    const next = clampPosition(
      event.clientX - state.offsetX,
      event.clientY - state.offsetY,
      { width: state.width, height: state.height },
    );
    setFloatingPosition(prev => (
      prev.x === next.x && prev.y === next.y ? prev : next
    ));
  }, [clampPosition, closeMenus]);

  const endDrag = useCallback((event: ReactPointerEvent<HTMLDivElement>) => {
    const state = dragStateRef.current;
    if (!state || state.pointerId !== event.pointerId) {
      return;
    }

    const toolbar = toolbarRef.current;
    if (state.pointerCaptured && toolbar) {
      try {
        toolbar.releasePointerCapture(event.pointerId);
      } catch (error) {
        // noop - releasePointerCapture may throw if pointer already released
      }
    }

    if (state.dragging) {
      event.preventDefault();
      suppressClickRef.current = true;
      if (typeof window !== 'undefined') {
        window.setTimeout(() => {
          suppressClickRef.current = false;
        }, 0);
      } else {
        suppressClickRef.current = false;
      }
    }

    dragStateRef.current = null;
    setIsDragging(false);
  }, []);

  const handleClickCapture = useCallback((event: ReactMouseEvent<HTMLDivElement>) => {
    if (suppressClickRef.current) {
      event.preventDefault();
      event.stopPropagation();
      suppressClickRef.current = false;
    }
  }, []);

  const floatingStyle = useMemo<CSSProperties | undefined>(() => {
    if (!isFullView) {
      return undefined;
    }
    return {
      transform: `translate3d(${Math.round(floatingPosition.x)}px, ${Math.round(floatingPosition.y)}px, 0)`,
    };
  }, [floatingPosition, isFullView]);

  return (
    <div
      ref={toolbarRef}
      className={clsx(
        'preview-toolbar',
        isFullView && 'preview-toolbar--floating',
        isFullView && isDragging && 'preview-toolbar--dragging',
      )}
      style={floatingStyle}
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={endDrag}
      onPointerCancel={endDrag}
      onClickCapture={handleClickCapture}
    >
      <div className="preview-toolbar__group preview-toolbar__group--left">
        {isFullView ? (
          <>
            <div
              className={clsx('preview-toolbar__menu', navMenuOpen && 'preview-toolbar__menu--open')}
              ref={navMenuRef}
            >
              <button
                type="button"
                className={clsx(
                  'preview-toolbar__icon-btn',
                  navMenuOpen && 'preview-toolbar__icon-btn--active',
                )}
                ref={navButtonRef}
                onClick={handleToggleNavMenu}
                disabled={!canGoBack && !canGoForward && isRefreshing}
                aria-haspopup="menu"
                aria-expanded={navMenuOpen}
                aria-label="Navigation actions"
                title="Navigation actions"
              >
                <Navigation2 aria-hidden size={18} />
              </button>
            {portalHost && navMenuOpen && navMenuStyle && createPortal(
                <div
                  className="preview-toolbar__menu-popover"
                  role="menu"
                  ref={navPopoverRef}
                  style={navMenuStyle}
                >
                  <button
                    type="button"
                    role="menuitem"
                    ref={navFirstItemRef}
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
                </div>,
                portalHost,
              )}
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
          className={clsx('preview-toolbar__menu', lifecycleMenuOpen && 'preview-toolbar__menu--open')}
          ref={lifecycleMenuRef}
        >
          <button
            type="button"
            className={clsx(
              'preview-toolbar__icon-btn',
              isAppRunning && 'preview-toolbar__icon-btn--danger',
              (pendingAction === 'start' || pendingAction === 'stop') && 'preview-toolbar__icon-btn--waiting',
              lifecycleMenuOpen && 'preview-toolbar__icon-btn--active',
            )}
            ref={lifecycleButtonRef}
            onClick={handleToggleLifecycleMenu}
            disabled={!hasCurrentApp || actionInProgress}
            aria-haspopup="menu"
            aria-expanded={lifecycleMenuOpen}
            aria-label={hasCurrentApp ? `Lifecycle actions (${appStatusLabel})` : 'Lifecycle actions unavailable'}
            title={hasCurrentApp ? `Lifecycle actions (${appStatusLabel})` : 'Lifecycle actions unavailable'}
          >
            {(pendingAction === 'start' || pendingAction === 'stop') ? (
              <Loader2 aria-hidden size={18} className="spinning" />
            ) : (
              <Power aria-hidden size={18} />
            )}
          </button>
          {portalHost && lifecycleMenuOpen && lifecycleMenuStyle && createPortal(
            <div
              className="preview-toolbar__menu-popover"
              role="menu"
              ref={lifecyclePopoverRef}
              style={lifecycleMenuStyle}
            >
              <button
                type="button"
                role="menuitem"
                ref={lifecycleFirstItemRef}
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
            </div>,
            portalHost,
          )}
        </div>
        <div
          className={clsx('preview-toolbar__menu', devMenuOpen && 'preview-toolbar__menu--open')}
          ref={devMenuRef}
        >
          <button
            type="button"
            className={clsx(
              'preview-toolbar__icon-btn',
              'preview-toolbar__icon-btn--dev',
              devMenuOpen && 'preview-toolbar__icon-btn--active',
            )}
            ref={devButtonRef}
            onClick={handleToggleDevMenu}
            disabled={!hasCurrentApp}
            aria-haspopup="menu"
            aria-expanded={devMenuOpen}
            aria-label={captureAriaLabel ? `Developer actions (${captureAriaLabel})` : 'Developer actions'}
            title={captureAriaLabel ? `Developer actions (${captureAriaLabel})` : 'Developer actions'}
          >
            <Wrench aria-hidden size={18} />
            {showCaptureBadge && (
              <span className="preview-toolbar__badge" aria-hidden>{captureBadgeLabel}</span>
            )}
          </button>
          {portalHost && devMenuOpen && devMenuStyle && createPortal(
            <div
              className="preview-toolbar__menu-popover"
              role="menu"
              ref={devPopoverRef}
              style={devMenuStyle}
            >
              <button
                type="button"
                role="menuitem"
                ref={devFirstItemRef}
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
            </div>,
            portalHost,
          )}
        </div>
      </div>
    </div>
  );
};

export default AppPreviewToolbar;
