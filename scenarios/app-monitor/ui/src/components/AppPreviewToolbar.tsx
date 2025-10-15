import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import type { ChangeEvent, CSSProperties, KeyboardEvent as ReactKeyboardEvent, MouseEvent as ReactMouseEvent } from 'react';
import clsx from 'clsx';
import {
  AlertTriangle,
  ArrowLeft,
  ArrowRight,
  Bug,
  ExternalLink,
  Info,
  Loader2,
  Power,
  RefreshCw,
  RotateCcw,
  ScrollText,
  Wrench,
} from 'lucide-react';

import './AppPreviewToolbar.css';

const MENU_OFFSET = 8;

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
  onViewLogs: () => void;
  onReportIssue: () => void;
  appStatusLabel: string;
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
  onViewLogs,
  onReportIssue,
  appStatusLabel,
  hasDetailsWarning,
}: AppPreviewToolbarProps) => {
  const [lifecycleMenuOpen, setLifecycleMenuOpen] = useState(false);
  const [devMenuOpen, setDevMenuOpen] = useState(false);
  const lifecycleMenuRef = useRef<HTMLDivElement | null>(null);
  const devMenuRef = useRef<HTMLDivElement | null>(null);
  const lifecycleButtonRef = useRef<HTMLButtonElement | null>(null);
  const devButtonRef = useRef<HTMLButtonElement | null>(null);
  const lifecyclePopoverRef = useRef<HTMLDivElement | null>(null);
  const devPopoverRef = useRef<HTMLDivElement | null>(null);
  const lifecycleFirstItemRef = useRef<HTMLButtonElement | null>(null);
  const devFirstItemRef = useRef<HTMLButtonElement | null>(null);
  const [lifecycleAnchorRect, setLifecycleAnchorRect] = useState<DOMRect | null>(null);
  const [devAnchorRect, setDevAnchorRect] = useState<DOMRect | null>(null);

  const updateLifecycleAnchor = useCallback(() => {
    const button = lifecycleButtonRef.current;
    if (!button) {
      setLifecycleAnchorRect(null);
      return;
    }
    setLifecycleAnchorRect(button.getBoundingClientRect());
  }, []);

  const updateDevAnchor = useCallback(() => {
    const button = devButtonRef.current;
    if (!button) {
      setDevAnchorRect(null);
      return;
    }
    setDevAnchorRect(button.getBoundingClientRect());
  }, []);

  const lifecycleMenuStyle = useMemo<CSSProperties | undefined>(() => {
    if (!lifecycleAnchorRect) {
      return undefined;
    }

    return {
      top: `${Math.round(lifecycleAnchorRect.bottom + MENU_OFFSET)}px`,
      left: `${Math.round(lifecycleAnchorRect.right)}px`,
      transform: 'translateX(-100%)',
    };
  }, [lifecycleAnchorRect]);

  const devMenuStyle = useMemo<CSSProperties | undefined>(() => {
    if (!devAnchorRect) {
      return undefined;
    }

    return {
      top: `${Math.round(devAnchorRect.bottom + MENU_OFFSET)}px`,
      left: `${Math.round(devAnchorRect.right)}px`,
      transform: 'translateX(-100%)',
    };
  }, [devAnchorRect]);

  const detailsButtonLabel = hasDetailsWarning
    ? 'Application details (localhost references detected)'
    : 'Application details';

  const isBrowser = typeof document !== 'undefined';

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
    if (!lifecycleMenuOpen) {
      return;
    }

    updateLifecycleAnchor();

    const handleResizeOrScroll = () => {
      updateLifecycleAnchor();
    };

    window.addEventListener('resize', handleResizeOrScroll);
    window.addEventListener('scroll', handleResizeOrScroll, true);

    return () => {
      window.removeEventListener('resize', handleResizeOrScroll);
      window.removeEventListener('scroll', handleResizeOrScroll, true);
    };
  }, [lifecycleMenuOpen, updateLifecycleAnchor]);

  useEffect(() => {
    if (!devMenuOpen) {
      return;
    }

    updateDevAnchor();

    const handleResizeOrScroll = () => {
      updateDevAnchor();
    };

    window.addEventListener('resize', handleResizeOrScroll);
    window.addEventListener('scroll', handleResizeOrScroll, true);

    return () => {
      window.removeEventListener('resize', handleResizeOrScroll);
      window.removeEventListener('scroll', handleResizeOrScroll, true);
    };
  }, [devMenuOpen, updateDevAnchor]);

  useEffect(() => {
    if (!lifecycleMenuOpen && !devMenuOpen) {
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
        }
      }
    };

    const handleKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'Escape') {
        if (lifecycleMenuOpen || devMenuOpen) {
          event.stopPropagation();
          setLifecycleMenuOpen(false);
          setLifecycleAnchorRect(null);
          setDevMenuOpen(false);
          setDevAnchorRect(null);
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
  }, [devMenuOpen, lifecycleMenuOpen]);

  const closeMenus = useCallback(() => {
    setLifecycleMenuOpen(false);
    setDevMenuOpen(false);
    setLifecycleAnchorRect(null);
    setDevAnchorRect(null);
  }, []);

  const handleToggleLifecycleMenu = useCallback(() => {
    const next = !lifecycleMenuOpen;
    setLifecycleMenuOpen(next);
    if (next) {
      updateLifecycleAnchor();
    } else {
      setLifecycleAnchorRect(null);
    }
    if (devMenuOpen) {
      setDevMenuOpen(false);
      setDevAnchorRect(null);
    }
  }, [devMenuOpen, lifecycleMenuOpen, updateLifecycleAnchor]);

  const handleToggleDevMenu = useCallback(() => {
    const next = !devMenuOpen;
    setDevMenuOpen(next);
    if (next) {
      updateDevAnchor();
    } else {
      setDevAnchorRect(null);
    }
    if (lifecycleMenuOpen) {
      setLifecycleMenuOpen(false);
      setLifecycleAnchorRect(null);
    }
  }, [devMenuOpen, lifecycleMenuOpen, updateDevAnchor]);

  const handleLifecycleAction = useCallback((action: 'toggle' | 'restart') => {
    if (action === 'toggle') {
      onToggleApp();
    } else {
      onRestartApp();
    }
    closeMenus();
  }, [closeMenus, onRestartApp, onToggleApp]);

  const handleViewLogs = useCallback(() => {
    onViewLogs();
    closeMenus();
  }, [closeMenus, onViewLogs]);

  const handleReportIssue = useCallback(() => {
    onReportIssue();
    closeMenus();
  }, [closeMenus, onReportIssue]);

  return (
    <div className="preview-toolbar">
      <div className="preview-toolbar__group preview-toolbar__group--left">
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
          className="preview-toolbar__icon-btn"
          onClick={onRefresh}
          disabled={isRefreshing}
          aria-label={isRefreshing ? 'Refreshing application status' : 'Refresh application'}
          title={isRefreshing ? 'Refreshing...' : 'Refresh'}
        >
          <RefreshCw aria-hidden size={18} className={clsx({ spinning: isRefreshing })} />
        </button>
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
          {isBrowser && lifecycleMenuOpen && lifecycleMenuStyle && createPortal(
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
            document.body,
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
            aria-label="Developer actions"
            title="Developer actions"
          >
            <Wrench aria-hidden size={18} />
          </button>
          {isBrowser && devMenuOpen && devMenuStyle && createPortal(
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
                onClick={handleViewLogs}
                disabled={!hasCurrentApp}
              >
                <ScrollText aria-hidden size={16} />
                <span>View logs</span>
              </button>
              <button
                type="button"
                role="menuitem"
                className="preview-toolbar__menu-item"
                onClick={handleReportIssue}
                disabled={!hasCurrentApp}
              >
                <Bug aria-hidden size={16} />
                <span>Report an issue</span>
              </button>
            </div>,
            document.body,
          )}
        </div>
      </div>
    </div>
  );
};

export default AppPreviewToolbar;
