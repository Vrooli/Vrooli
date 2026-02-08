import { useEffect, useRef } from 'react';
import type { BridgeComplianceResult } from '@/hooks/useIframeBridge';

interface UsePreviewBridgeComplianceCheckOptions {
  enabled: boolean;
  runComplianceCheck: () => Promise<BridgeComplianceResult>;
  onSuccess: (result: BridgeComplianceResult) => void;
  onError: (error: unknown) => void;
  runOnceWhileEnabled?: boolean;
  resetKey?: unknown;
}

export function usePreviewBridgeComplianceCheck({
  enabled,
  runComplianceCheck,
  onSuccess,
  onError,
  runOnceWhileEnabled = false,
  resetKey,
}: UsePreviewBridgeComplianceCheckOptions) {
  const hasRunRef = useRef(false);
  const previousResetKeyRef = useRef(resetKey);

  useEffect(() => {
    if (previousResetKeyRef.current !== resetKey) {
      previousResetKeyRef.current = resetKey;
      hasRunRef.current = false;
    }
  }, [resetKey]);

  useEffect(() => {
    if (!enabled) {
      hasRunRef.current = false;
      return;
    }

    if (runOnceWhileEnabled && hasRunRef.current) {
      return;
    }

    hasRunRef.current = true;
    let cancelled = false;
    runComplianceCheck()
      .then((result) => {
        if (!cancelled) {
          onSuccess(result);
        }
      })
      .catch((error) => {
        if (!cancelled) {
          onError(error);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [enabled, onError, onSuccess, runComplianceCheck, runOnceWhileEnabled]);
}

export default usePreviewBridgeComplianceCheck;
