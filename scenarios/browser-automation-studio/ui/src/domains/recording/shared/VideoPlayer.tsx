/**
 * VideoPlayer - Video player wrapper with hidden native controls
 *
 * Designed to be controlled by PlaybackControls. Exposes a ref for
 * the parent to control playback programmatically.
 */

import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from 'react';
import { AlertCircle, Loader2 } from 'lucide-react';
import clsx from 'clsx';

export interface VideoPlayerRef {
  /** Play the video */
  play: () => Promise<void>;
  /** Pause the video */
  pause: () => void;
  /** Toggle play/pause */
  toggle: () => void;
  /** Seek to a specific time in seconds */
  seekTo: (time: number) => void;
  /** Seek to a position (0 to 1) */
  seekToPosition: (position: number) => void;
  /** Get current time in seconds */
  getCurrentTime: () => number;
  /** Get duration in seconds */
  getDuration: () => number;
  /** Get progress as 0 to 1 */
  getProgress: () => number;
  /** Check if video is playing */
  isPlaying: () => boolean;
  /** Get the underlying video element */
  getVideoElement: () => HTMLVideoElement | null;
}

export interface VideoPlayerProps {
  /** URL of the video to play */
  src: string;
  /** Optional poster image */
  poster?: string;
  /** Whether to start playing automatically */
  autoPlay?: boolean;
  /** Whether to loop the video */
  loop?: boolean;
  /** Whether to mute the video */
  muted?: boolean;
  /** Object-fit style for the video */
  objectFit?: 'contain' | 'cover' | 'fill';
  /** Additional class name for the container */
  className?: string;
  /** Callback when video starts playing */
  onPlay?: () => void;
  /** Callback when video pauses */
  onPause?: () => void;
  /** Callback when video ends */
  onEnded?: () => void;
  /** Callback when time updates (fires frequently) */
  onTimeUpdate?: (currentTime: number, duration: number) => void;
  /** Callback when video is ready to play */
  onCanPlay?: () => void;
  /** Callback when video has an error */
  onError?: (error: Error) => void;
}

export const VideoPlayer = forwardRef<VideoPlayerRef, VideoPlayerProps>(
  function VideoPlayer(
    {
      src,
      poster,
      autoPlay = false,
      loop = false,
      muted = true,
      objectFit = 'contain',
      className,
      onPlay,
      onPause,
      onEnded,
      onTimeUpdate,
      onCanPlay,
      onError,
    },
    ref
  ) {
    const videoRef = useRef<HTMLVideoElement>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [isPlaying, setIsPlaying] = useState(false);

    // Expose control methods via ref
    useImperativeHandle(ref, () => ({
      play: async () => {
        if (videoRef.current) {
          try {
            await videoRef.current.play();
          } catch (e) {
            console.error('Failed to play video:', e);
          }
        }
      },
      pause: () => {
        videoRef.current?.pause();
      },
      toggle: () => {
        if (videoRef.current) {
          if (videoRef.current.paused) {
            videoRef.current.play().catch(console.error);
          } else {
            videoRef.current.pause();
          }
        }
      },
      seekTo: (time: number) => {
        if (videoRef.current) {
          videoRef.current.currentTime = time;
        }
      },
      seekToPosition: (position: number) => {
        if (videoRef.current && videoRef.current.duration) {
          videoRef.current.currentTime = position * videoRef.current.duration;
        }
      },
      getCurrentTime: () => videoRef.current?.currentTime ?? 0,
      getDuration: () => videoRef.current?.duration ?? 0,
      getProgress: () => {
        const video = videoRef.current;
        if (!video || !video.duration) return 0;
        return video.currentTime / video.duration;
      },
      isPlaying: () => isPlaying,
      getVideoElement: () => videoRef.current,
    }));

    // Handle video events
    const handleLoadStart = useCallback(() => {
      setIsLoading(true);
      setError(null);
    }, []);

    const handleCanPlay = useCallback(() => {
      setIsLoading(false);
      onCanPlay?.();
    }, [onCanPlay]);

    const handlePlay = useCallback(() => {
      setIsPlaying(true);
      onPlay?.();
    }, [onPlay]);

    const handlePause = useCallback(() => {
      setIsPlaying(false);
      onPause?.();
    }, [onPause]);

    const handleEnded = useCallback(() => {
      setIsPlaying(false);
      onEnded?.();
    }, [onEnded]);

    const handleTimeUpdate = useCallback(() => {
      if (videoRef.current && onTimeUpdate) {
        onTimeUpdate(videoRef.current.currentTime, videoRef.current.duration);
      }
    }, [onTimeUpdate]);

    const handleError = useCallback(() => {
      const video = videoRef.current;
      let errorMessage = 'Failed to load video';

      if (video?.error) {
        switch (video.error.code) {
          case MediaError.MEDIA_ERR_ABORTED:
            errorMessage = 'Video playback aborted';
            break;
          case MediaError.MEDIA_ERR_NETWORK:
            errorMessage = 'Network error while loading video';
            break;
          case MediaError.MEDIA_ERR_DECODE:
            errorMessage = 'Video decode error';
            break;
          case MediaError.MEDIA_ERR_SRC_NOT_SUPPORTED:
            errorMessage = 'Video format not supported';
            break;
        }
      }

      setError(errorMessage);
      setIsLoading(false);
      onError?.(new Error(errorMessage));
    }, [onError]);

    // Reset state when src changes
    useEffect(() => {
      setIsLoading(true);
      setError(null);
      setIsPlaying(false);
    }, [src]);

    // Error state
    if (error) {
      return (
        <div
          className={clsx(
            'h-full w-full flex flex-col items-center justify-center bg-gray-900 text-gray-400',
            className
          )}
        >
          <AlertCircle className="w-12 h-12 mb-3 text-red-500" />
          <p className="text-sm text-red-400">{error}</p>
        </div>
      );
    }

    return (
      <div className={clsx('h-full w-full relative bg-gray-900', className)}>
        {/* Loading overlay */}
        {isLoading && (
          <div className="absolute inset-0 flex items-center justify-center bg-gray-900/80 z-10">
            <Loader2 className="w-8 h-8 text-flow-accent animate-spin" />
          </div>
        )}

        {/* Video element - native controls hidden */}
        <video
          ref={videoRef}
          src={src}
          poster={poster}
          autoPlay={autoPlay}
          loop={loop}
          muted={muted}
          playsInline
          className={clsx(
            'w-full h-full',
            objectFit === 'contain' && 'object-contain',
            objectFit === 'cover' && 'object-cover',
            objectFit === 'fill' && 'object-fill'
          )}
          onLoadStart={handleLoadStart}
          onCanPlay={handleCanPlay}
          onPlay={handlePlay}
          onPause={handlePause}
          onEnded={handleEnded}
          onTimeUpdate={handleTimeUpdate}
          onError={handleError}
        />
      </div>
    );
  }
);
