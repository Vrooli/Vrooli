import { useCallback, useState } from "react";

interface UseCodeCopyReturn {
  copied: boolean;
  copyCode: () => void;
}

/** Copy code to clipboard with a 2-second "copied" feedback state. */
export function useCodeCopy(code: string): UseCodeCopyReturn {
  const [copied, setCopied] = useState(false);

  const copyCode = useCallback(() => {
    void navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [code]);

  return { copied, copyCode };
}
