/**
 * @vrooliComponentSource react-component-library:markdown-renderer
 * @vrooliComponentVersion 0.3.2
 * @vrooliComponentAdoption 612450da-7d3d-4888-85a9-e9ecf63254a6
 * @vrooliComponentAppliedAt 2026-07-21T21:01:34Z
 * @vrooliComponentSourceSha256 46cc4ad7a664bfd7d4bea8891cd51f6e9525c5e145ceb21d6d649946bf9b96c8
 * @vrooliComponentDriftHash 46cc4ad7a664bfd7d4bea8891cd51f6e9525c5e145ceb21d6d649946bf9b96c8
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { useCallback, useEffect, useState } from "react";

export function useCodeCopy(resetAfterMs = 1500) {
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!copied) return;
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