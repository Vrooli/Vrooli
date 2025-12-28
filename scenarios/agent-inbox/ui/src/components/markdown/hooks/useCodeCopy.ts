import { useCallback, useState } from "react";
import { useToast } from "../../ui/toast";

interface UseCodeCopyReturn {
  /** Whether the code was recently copied */
  copied: boolean;
  /** Copy the code to clipboard */
  copyCode: () => void;
}

/**
 * Hook for copying code to clipboard with toast feedback.
 */
export function useCodeCopy(code: string): UseCodeCopyReturn {
  const [copied, setCopied] = useState(false);
  const { addToast } = useToast();

  const copyCode = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      addToast("Copied to clipboard", "success");

      // Reset copied state after 2 seconds
      setTimeout(() => setCopied(false), 2000);
    } catch {
      addToast("Failed to copy", "error");
    }
  }, [code, addToast]);

  return { copied, copyCode };
}
