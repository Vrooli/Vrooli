import { useCallback, useRef, useState } from 'react';

const DEFAULT_RESET_MS = 1500;

export function useCodeCopy(text: string, resetMs = DEFAULT_RESET_MS) {
  const [copied, setCopied] = useState(false);
  const timeoutRef = useRef<number | null>(null);

  const copyCode = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);

      if (timeoutRef.current !== null) {
        window.clearTimeout(timeoutRef.current);
      }

      timeoutRef.current = window.setTimeout(() => {
        setCopied(false);
        timeoutRef.current = null;
      }, resetMs);
    } catch (error) {
      console.error('Copy failed:', error);
      setCopied(false);
    }
  }, [resetMs, text]);

  return { copied, copyCode };
}
