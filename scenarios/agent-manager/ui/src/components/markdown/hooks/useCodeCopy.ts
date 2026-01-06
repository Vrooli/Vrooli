import { useCallback, useState } from "react";

type CopyStatus = "idle" | "copied" | "error";

interface UseCodeCopyReturn {
  status: CopyStatus;
  copyCode: () => void;
}

export function useCodeCopy(code: string): UseCodeCopyReturn {
  const [status, setStatus] = useState<CopyStatus>("idle");

  const copyCode = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(code);
      setStatus("copied");
      setTimeout(() => setStatus("idle"), 2000);
    } catch {
      setStatus("error");
      setTimeout(() => setStatus("idle"), 2000);
    }
  }, [code]);

  return { status, copyCode };
}
