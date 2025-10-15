import { useCallback, useEffect, useRef, useState } from 'react';
import type { ChangeEvent, KeyboardEvent as ReactKeyboardEvent, MouseEvent as ReactMouseEvent } from 'react';
import clsx from 'clsx';
import {
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
}: AppPreviewToolbarProps) => {
  const [lifecycleMenuOpen, setLifecycleMenuOpen] = useState(false);
  const [devMenuOpen, setDevMenuOpen] = useState(false);
  const lifecycleMenuRef = useRef<HTMLDivElement | null>(null);
  const devMenuRef = useRef<HTMLDivElement | null>(null);
  const lifecycleFirstItemRef = useRef<HTMLButtonElement | null>(null);
  const devFirstItemRef = useRef<HTMLButtonElement | null>(null);

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
    if (!lifecycleMenuOpen && !devMenuOpen) {
      return () => {};
    }

    const handlePointerDown = (event: globalThis.MouseEvent | globalThis.TouchEvent) => {
      const target = event.target as Node | null;
      if (lifecycleMenuOpen) {
        const menu = lifecycleMenuRef.current;
        if (menu && !menu.contains(target)) {
          setLifecycleMenuOpen(false);
        }
      }
      if (devMenuOpen) {
        const menu = devMenuRef.current;
        if (menu && !menu.contains(target)) {
          setDevMenuOpen(false);
        }
      }
    };

    const handleKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'Escape') {
        if (lifecycleMenuOpen || devMenuOpen) {
          event.stopPropagation();
          setLifecycleMenuOpen(false);
          setDevMenuOpen(false);
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
  }, []);

  const handleToggleLifecycleMenu = useCallback(() => {
    setLifecycleMenuOpen(prev => !prev);
    setDevMenuOpen(false);
  }, []);

  const handleToggleDevMenu = useCallback(() => {
    setDevMenuOpen(prev => !prev);
    setLifecycleMenuOpen(false);
  }, []);

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
      </div>
      <div className="preview-toolbar__title">
        <div
          className={clsx('preview-toolbar__url-wrapper', urlStatusClass)}
          title={urlStatusTitle}
        >
          <button
            type="button"
            className="preview-toolbar__url-action-btn"
            onClick={onOpenDetails}
            disabled={!hasCurrentApp}
            aria-label="Application details"
            title="Application details"
          >
            <Info aria-hidden size={16} />
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
          {lifecycleMenuOpen && (
            <div className="preview-toolbar__menu-popover" role="menu">
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
            </div>
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
            onClick={handleToggleDevMenu}
            disabled={!hasCurrentApp}
            aria-haspopup="menu"
            aria-expanded={devMenuOpen}
            aria-label="Developer actions"
            title="Developer actions"
          >
            <Wrench aria-hidden size={18} />
          </button>
          {devMenuOpen && (
            <div className="preview-toolbar__menu-popover" role="menu">
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
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default AppPreviewToolbar;
