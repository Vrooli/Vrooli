import { memo } from 'react';
import {
  ArrowLeft,
  ArrowRight,
  ExternalLink,
  Globe,
  Loader2,
  Maximize2,
  Minimize2,
  RefreshCw,
  Settings,
  Wand2,
  X,
} from 'lucide-react';

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
}: LandingPreviewToolbarProps) {
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

        {/* Close button */}
        <button
          type="button"
          className="landing-preview-toolbar__icon-btn landing-preview-toolbar__icon-btn--close"
          onClick={onClose}
          aria-label="Close preview"
          title="Close"
        >
          <X size={18} />
        </button>
      </div>
    </div>
  );
});

export default LandingPreviewToolbar;
