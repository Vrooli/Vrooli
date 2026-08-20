import { useEffect } from 'react';
import { emitShortcutIntent } from '@vrooli/iframe-bridge';

export function useEscapeKey(onEscape: () => void, enabled = true): void {
  useEffect(() => {
    if (!enabled) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onEscape();
        emitShortcutIntent({ action: 'dialog.close', outcome: 'handled', chord: 'Escape', source: 'keyboard' });
      }
    };
    document.addEventListener('keydown', handler);
    return () => { document.removeEventListener('keydown', handler); };
  }, [onEscape, enabled]);
}
