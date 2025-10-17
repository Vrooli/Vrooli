import { useEffect } from 'react';

interface Options {
  onLeft: () => void;
  onRight: () => void;
  onSkip: () => void;
  onUndo: () => void;
  enabled: boolean;
}

export const useKeyboardShortcuts = ({ onLeft, onRight, onSkip, onUndo, enabled }: Options) => {
  useEffect(() => {
    if (!enabled) return undefined;

    const handler = (event: KeyboardEvent) => {
      if (!enabled) return;

      if (event.key === 'ArrowLeft' || event.key.toLowerCase() === 'a') {
        event.preventDefault();
        onLeft();
      } else if (event.key === 'ArrowRight' || event.key.toLowerCase() === 'd') {
        event.preventDefault();
        onRight();
      } else if (event.code === 'Space') {
        event.preventDefault();
        onSkip();
      } else if (event.key.toLowerCase() === 'z' && (event.metaKey || event.ctrlKey)) {
        event.preventDefault();
        onUndo();
      }
    };

    window.addEventListener('keydown', handler, { passive: false });
    return () => window.removeEventListener('keydown', handler);
  }, [enabled, onLeft, onRight, onSkip, onUndo]);
};
