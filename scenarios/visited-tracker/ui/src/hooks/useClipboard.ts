import { useState, useCallback, useEffect, useRef } from 'react';

export function useClipboard(timeout = 2000) {
  const [isCopied, setIsCopied] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const mounted = useRef(false);
  const request = useRef(0);

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      request.current += 1;
      if (timer.current !== null) clearTimeout(timer.current);
    };
  }, []);

  const copy = useCallback(async (text: string) => {
    const current = ++request.current;
    if (timer.current !== null) {
      clearTimeout(timer.current);
      timer.current = null;
    }
    if (mounted.current) {
      setIsCopied(false);
      setError(null);
    }
    if (typeof navigator === 'undefined' || !navigator.clipboard?.writeText) {
      if (mounted.current) setError('Clipboard is unavailable. Select the text to copy it manually.');
      return false;
    }
    try {
      await navigator.clipboard.writeText(text);
      // The browser write cannot be cancelled. Only the latest request on a
      // mounted component may change feedback or schedule a reset timer.
      if (mounted.current && request.current === current) {
        setIsCopied(true);
        timer.current = setTimeout(() => {
          timer.current = null;
          setIsCopied(false);
        }, timeout);
      }
      return true;
    } catch {
      if (mounted.current && request.current === current) {
        setIsCopied(false);
        setError('Copy failed. Select the text to copy it manually.');
      }
      return false;
    }
  }, [timeout]);

  return { isCopied, error, copy };
}
