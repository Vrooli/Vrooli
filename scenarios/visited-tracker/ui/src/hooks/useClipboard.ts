import { useState, useCallback } from 'react';

export function useClipboard(timeout = 2000) {
  const [isCopied, setIsCopied] = useState(false);

  const copy = useCallback(
    async (text: string) => {
      if (!navigator?.clipboard) {
        console.warn('Clipboard not supported');
        return false;
      }

      try {
        await navigator.clipboard.writeText(text);
        setIsCopied(true);
        setTimeout(() => setIsCopied(false), timeout);
        return true;
      } catch (error) {
        console.warn('Failed to copy text', error);
        setIsCopied(false);
        return false;
      }
    },
    [timeout]
  );

  return { isCopied, copy };
}
