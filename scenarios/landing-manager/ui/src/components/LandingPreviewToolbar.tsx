import { memo, useEffect, useRef, useState } from 'react';
import {
  ArrowLeft,
  ArrowRight,
  ExternalLink,
  Globe,
  Home,
  Info,
  Loader2,
  Maximize2,
  Minimize2,
  Play,
  RefreshCw,
  RotateCw,
  Square,
  Settings,
  SlidersHorizontal,
  Wand2,
} from 'lucide-react';
import { type LifecycleControlConfig } from './types';

export type PreviewViewType = 'public' | 'admin';

export interface LandingPreviewToolbarProps {
  scenarioName: string;
  viewType: PreviewViewType;
  onViewTypeChange: (viewType: PreviewViewType) => void;
  onRefresh: () => void;
  isRefreshing: boolean;
  onCustomize: () => void;
  onOpenExternal: () => void;
  externalUrl: string | null;
  onClose: () => void;
  isFullscreen: boolean;
  onToggleFullscreen: () => void;
  canGoBack?: boolean;
  canGoForward?: boolean;
  onGoBack?: () => void;
  onGoForward?: () => void;
  showInfoButton?: boolean;
  infoPanelOpen?: boolean;
  onToggleInfoPanel?: () => void;
  lifecycleControls?: LifecycleControlConfig | null;
}

export const LandingPreviewToolbar = memo(function LandingPreviewToolbar({
  scenarioName,
  viewType,
  onViewTypeChange,
  onRefresh,
  isRefreshing,
  onCustomize,
  onOpenExternal,
  externalUrl,
  onClose,
  isFullscreen,
  onToggleFullscreen,
  canGoBack = false,
  canGoForward = false,
  onGoBack,
  onGoForward,
  showInfoButton = false,
  infoPanelOpen = false,
  onToggleInfoPanel,
  lifecycleControls,
}: LandingPreviewToolbarProps) {
  const [actionsOpen, setActionsOpen] = useState(false);
  const actionsRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!actionsOpen) return;

    const handleClick = (event: MouseEvent) => {
      if (!actionsRef.current?.contains(event.target as Node)) {
        setActionsOpen(false);
      }
    };

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setActionsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClick);
    document.addEventListener('keydown', handleEscape);
    return () => {
      document.removeEventListener('mousedown', handleClick);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [actionsOpen]);

  const toggleActionsMenu = () => {
    if (!lifecycleControls) return;
    setActionsOpen((prev) => !prev);
  };

  const handleStart = () => {
    lifecycleControls?.onStart();
    setActionsOpen(false);
  };

  const handleStop = () => {
    lifecycleControls?.onStop?.();
    setActionsOpen(false);
  };

  const handleRestart = () => {
    lifecycleControls?.onRestart?.();
    setActionsOpen(false);
  };

  const lifecycleButtonDisabled = lifecycleControls?.loading;

  return (
    <div className="landing-preview-toolbar">
      <div className="landing-preview-toolbar__left">
        {/* Navigation buttons */}
        <button
          type="button"
          className="landing-preview-toolbar__icon-btn"
          onClick={onGoBack}
          disabled={!canGoBack}
          aria-label="Go back"
          title="Go back"
        >
          <ArrowLeft size={18} />
        </button>
        <button
          type="button"
          className="landing-preview-toolbar__icon-btn"
          onClick={onGoForward}
          disabled={!canGoForward}
          aria-label="Go forward"
          title="Go forward"
        >
          <ArrowRight size={18} />
        </button>
        <button
          type="button"
          className="landing-preview-toolbar__icon-btn"
          onClick={onRefresh}
          disabled={isRefreshing}
          aria-label="Refresh preview"
          title="Refresh"
        >
          <RefreshCw size={18} className={isRefreshing ? 'animate-spin' : ''} />
        </button>

        {/* View type toggle */}
        <div className="landing-preview-toolbar__view-toggle">
          <button
            type="button"
            className={`landing-preview-toolbar__view-btn ${viewType === 'public' ? 'landing-preview-toolbar__view-btn--active' : ''}`}
            onClick={() => onViewTypeChange('public')}
            aria-pressed={viewType === 'public'}
            title="View public landing page"
          >
            <Globe size={14} />
            <span>Public</span>
          </button>
          <button
            type="button"
            className={`landing-preview-toolbar__view-btn ${viewType === 'admin' ? 'landing-preview-toolbar__view-btn--active' : ''}`}
            onClick={() => onViewTypeChange('admin')}
            aria-pressed={viewType === 'admin'}
            title="View admin dashboard"
          >
            <Settings size={14} />
            <span>Admin</span>
          </button>
        </div>
      </div>

      <div className="landing-preview-toolbar__center">
        <span className="landing-preview-toolbar__title">{scenarioName}</span>
        {isRefreshing && (
          <Loader2 size={14} className="animate-spin text-emerald-400" />
        )}
      </div>

      <div className="landing-preview-toolbar__right">
        <button
          type="button"
          className="landing-preview-toolbar__action-btn landing-preview-toolbar__action-btn--back"
          onClick={onClose}
          title="Back to factory"
          data-testid="preview-back-button"
        >
          <Home size={16} />
          <span>Back to Factory</span>
        </button>

        {showInfoButton && (
          <button
            type="button"
            className={`landing-preview-toolbar__icon-btn ${infoPanelOpen ? 'landing-preview-toolbar__icon-btn--active' : ''}`}
            onClick={onToggleInfoPanel}
            aria-pressed={infoPanelOpen}
            aria-label="Toggle scenario information"
            title="Scenario info"
            data-testid="preview-info-toggle"
          >
            <Info size={18} />
          </button>
        )}

        {lifecycleControls && (
          <div className="landing-preview-toolbar__actions" ref={actionsRef}>
            <button
              type="button"
              className={`landing-preview-toolbar__icon-btn ${actionsOpen ? 'landing-preview-toolbar__icon-btn--active' : ''}`}
              onClick={toggleActionsMenu}
              aria-haspopup="true"
              aria-expanded={actionsOpen}
              aria-label="Lifecycle actions"
              title="Lifecycle actions"
            >
              <SlidersHorizontal size={18} />
            </button>
            {actionsOpen && (
              <div className="landing-preview-toolbar__actions-menu" role="menu">
                <button
                  type="button"
                  onClick={handleStart}
                  className="landing-preview-toolbar__actions-menu-btn"
                  disabled={lifecycleButtonDisabled || lifecycleControls.running}
                  role="menuitem"
                >
                  <Play size={14} />
                  Start
                </button>
                <button
                  type="button"
                  onClick={handleStop}
                  className="landing-preview-toolbar__actions-menu-btn"
                  disabled={lifecycleButtonDisabled || !lifecycleControls.onStop || !lifecycleControls.running}
                  role="menuitem"
                >
                  <Square size={14} />
                  Stop
                </button>
                <button
                  type="button"
                  onClick={handleRestart}
                  className="landing-preview-toolbar__actions-menu-btn"
                  disabled={lifecycleButtonDisabled || !lifecycleControls.onRestart || !lifecycleControls.running}
                  role="menuitem"
                >
                  <RotateCw size={14} />
                  Restart
                </button>
              </div>
            )}
          </div>
        )}

        {/* AI Customize button */}
        <button
          type="button"
          className="landing-preview-toolbar__action-btn landing-preview-toolbar__action-btn--customize"
          onClick={onCustomize}
          title="Customize with AI"
        >
          <Wand2 size={16} />
          <span>Customize</span>
        </button>

        {/* Open in new tab */}
        <button
          type="button"
          className="landing-preview-toolbar__icon-btn"
          onClick={onOpenExternal}
          disabled={!externalUrl}
          aria-label="Open in new tab"
          title="Open in new tab"
        >
          <ExternalLink size={18} />
        </button>

        {/* Fullscreen toggle */}
        <button
          type="button"
          className="landing-preview-toolbar__icon-btn"
          onClick={onToggleFullscreen}
          aria-label={isFullscreen ? 'Exit fullscreen' : 'Enter fullscreen'}
          title={isFullscreen ? 'Exit fullscreen' : 'Fullscreen'}
        >
          {isFullscreen ? <Minimize2 size={18} /> : <Maximize2 size={18} />}
        </button>
      </div>
    </div>
  );
});

export default LandingPreviewToolbar;
