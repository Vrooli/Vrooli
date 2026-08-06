import { useCallback, useEffect, useState } from "react";

export function useCodeCopy(resetAfterMs = 1500) {
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!copied || typeof window === "undefined") return;
    const timer = window.setTimeout(() => setCopied(false), resetAfterMs);
    return () => window.clearTimeout(timer);
  }, [copied, resetAfterMs]);

  const copy = useCallback(async (value: string) => {
    try {
      await navigator.clipboard?.writeText(value);
      setCopied(true);
      return true;
    } catch {
      setCopied(false);
      return false;
    }
  }, []);

  return { copied, copy };
}
