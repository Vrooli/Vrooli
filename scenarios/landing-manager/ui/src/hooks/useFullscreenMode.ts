import { useCallback, useEffect, useState } from 'react';
import type { MutableRefObject } from 'react';

interface UseFullscreenModeOptions {
  previewViewRef: MutableRefObject<HTMLDivElement | null>;
}

export interface UseFullscreenModeReturn {
  isFullscreen: boolean;
  isLayoutFullscreen: boolean;
  setIsLayoutFullscreen: (value: boolean) => void;
  handleToggleFullscreen: () => void;
}

export const useFullscreenMode = ({
  previewViewRef,
}: UseFullscreenModeOptions): UseFullscreenModeReturn => {
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [isLayoutFullscreen, setIsLayoutFullscreen] = useState(false);

  // Monitor native fullscreen API
  useEffect(() => {
    if (typeof document === 'undefined') {
      return;
    }

    const handleFullscreenChange = () => {
      setIsFullscreen(document.fullscreenElement === previewViewRef.current);
    };

    document.addEventListener('fullscreenchange', handleFullscreenChange);
    handleFullscreenChange();

    return () => {
      document.removeEventListener('fullscreenchange', handleFullscreenChange);
    };
  }, [previewViewRef]);

  // Manage layout fullscreen body class
  useEffect(() => {
    if (typeof document === 'undefined') {
      return () => {};
    }

    const className = 'landing-preview-immersive';
    const { body } = document;
    if (isLayoutFullscreen) {
      body.classList.add(className);
    } else {
      body.classList.remove(className);
    }

    return () => {
      body.classList.remove(className);
    };
  }, [isLayoutFullscreen]);

  // Handle escape key for layout fullscreen
  useEffect(() => {
    if (!isLayoutFullscreen) {
      return () => {};
    }

    const handleKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'Escape') {
        setIsLayoutFullscreen(false);
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [isLayoutFullscreen]);

  const handleToggleFullscreen = useCallback(() => {
    if (typeof document === 'undefined') {
      return;
    }

    const container = previewViewRef.current;
    if (!container) {
      return;
    }

    if (document.fullscreenElement === container) {
      const exitFullscreen = typeof document.exitFullscreen === 'function'
        ? document.exitFullscreen.bind(document)
        : null;
      if (exitFullscreen) {
        exitFullscreen().catch(() => {
          // Fallback silently
        });
      }
      return;
    }

    if (isLayoutFullscreen) {
      setIsLayoutFullscreen(false);
      return;
    }

    const enterNativeFullscreen = () => {
      if (typeof container.requestFullscreen === 'function') {
        return container.requestFullscreen();
      }
      return Promise.reject(new Error('Fullscreen API unavailable'));
    };

    if (document.fullscreenElement && document.fullscreenElement !== container) {
      const exitFullscreen = typeof document.exitFullscreen === 'function'
        ? document.exitFullscreen.bind(document)
        : null;
      if (exitFullscreen) {
        exitFullscreen()
          .then(() => enterNativeFullscreen().catch(() => {
            setIsLayoutFullscreen(true);
          }))
          .catch(() => {
            setIsLayoutFullscreen(true);
          });
        return;
      }
      setIsLayoutFullscreen(true);
      return;
    }

    enterNativeFullscreen()
      .catch(() => {
        setIsLayoutFullscreen(true);
      });
  }, [isLayoutFullscreen, previewViewRef]);

  return {
    isFullscreen,
    isLayoutFullscreen,
    setIsLayoutFullscreen,
    handleToggleFullscreen,
  };
};
