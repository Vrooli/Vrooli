/**
 * Shared components for recording and execution preview
 *
 * These components provide unified presentation and playback functionality
 * across both recording mode (live preview) and execution mode (replay/slideshow).
 */

// Presentation
export { PresentationWrapper } from './PresentationWrapper';
export type { PresentationWrapperProps } from './PresentationWrapper';

// Unified preview container
export { PreviewContainer } from './PreviewContainer';
export type { PreviewContainerProps, PreviewContainerContext } from './PreviewContainer';

// Playback controls
export { PlaybackControls } from './PlaybackControls';
export type { PlaybackControlsProps, PlaybackContentType } from './PlaybackControls';

// Slideshow
export { ScreenshotSlideshow } from './ScreenshotSlideshow';
export type { ScreenshotSlideshowProps, Screenshot } from './ScreenshotSlideshow';

// Video player
export { VideoPlayer } from './VideoPlayer';
export type { VideoPlayerRef, VideoPlayerProps } from './VideoPlayer';

// Hooks
export { useSlideshowPlayback } from './useSlideshowPlayback';
export type {
  SlideshowPlaybackState,
  SlideshowPlaybackActions,
  UseSlideshowPlaybackOptions,
} from './useSlideshowPlayback';

// Execution completion actions
export { ExecutionCompletionActions } from './ExecutionCompletionActions';
export type { ExecutionCompletionActionsProps } from './ExecutionCompletionActions';
