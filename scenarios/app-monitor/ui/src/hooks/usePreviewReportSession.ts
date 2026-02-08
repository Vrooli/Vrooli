import { useCallback, useMemo, useState } from 'react';
import type { ReportElementCapture } from '@/components/report/reportTypes';
import { logger } from '@/services/logger';

interface PreviewBridgeStateSnapshot {
  isSupported: boolean;
  caps?: readonly string[] | null;
}

interface UsePreviewReportSessionOptions {
  activePreviewUrl: string;
  bridgeState: PreviewBridgeStateSnapshot;
  logPrefix?: string;
}

interface UsePreviewReportSessionResult {
  reportElementCaptures: ReportElementCapture[];
  hasPrimaryCaptureDraft: boolean;
  setHasPrimaryCaptureDraft: (next: boolean) => void;
  stagedCaptureCount: number;
  canCaptureScreenshot: boolean;
  bridgeSupportsScreenshot: boolean;
  isPreviewSameOrigin: boolean;
  handleInspectorCaptureAdded: (capture: ReportElementCapture) => void;
  handleElementCaptureNoteChange: (captureId: string, note: string) => void;
  handleRemoveElementCapture: (captureId: string) => void;
  resetElementCaptures: () => void;
  resetReportDraftState: () => void;
}

export function usePreviewReportSession({
  activePreviewUrl,
  bridgeState,
  logPrefix = '[preview-report]',
}: UsePreviewReportSessionOptions): UsePreviewReportSessionResult {
  const [reportElementCaptures, setReportElementCaptures] = useState<ReportElementCapture[]>([]);
  const [hasPrimaryCaptureDraft, setHasPrimaryCaptureDraft] = useState(false);

  const stagedCaptureCount = useMemo(
    () => reportElementCaptures.length + (hasPrimaryCaptureDraft ? 1 : 0),
    [hasPrimaryCaptureDraft, reportElementCaptures.length],
  );

  const canCaptureScreenshot = useMemo(
    () => Boolean(activePreviewUrl),
    [activePreviewUrl],
  );

  const bridgeSupportsScreenshot = useMemo(
    () => bridgeState.isSupported
      && Array.isArray(bridgeState.caps)
      && bridgeState.caps.includes('screenshot'),
    [bridgeState.caps, bridgeState.isSupported],
  );

  const isPreviewSameOrigin = useMemo(() => {
    if (typeof window === 'undefined' || !activePreviewUrl) {
      return false;
    }

    try {
      const targetOrigin = new URL(activePreviewUrl, window.location.href).origin;
      return targetOrigin === window.location.origin;
    } catch (error) {
      logger.warn(`${logPrefix} Failed to evaluate preview origin`, { activePreviewUrl, error });
      return false;
    }
  }, [activePreviewUrl, logPrefix]);

  const handleInspectorCaptureAdded = useCallback((capture: ReportElementCapture) => {
    setReportElementCaptures((previous) => [...previous, capture]);
  }, []);

  const handleElementCaptureNoteChange = useCallback((captureId: string, note: string) => {
    setReportElementCaptures((previous) => previous.map((capture) => (
      capture.id === captureId ? { ...capture, note } : capture
    )));
  }, []);

  const handleRemoveElementCapture = useCallback((captureId: string) => {
    setReportElementCaptures((previous) => previous.filter((capture) => capture.id !== captureId));
  }, []);

  const resetElementCaptures = useCallback(() => {
    setReportElementCaptures([]);
  }, []);

  const resetReportDraftState = useCallback(() => {
    setReportElementCaptures([]);
    setHasPrimaryCaptureDraft(false);
  }, []);

  return {
    reportElementCaptures,
    hasPrimaryCaptureDraft,
    setHasPrimaryCaptureDraft,
    stagedCaptureCount,
    canCaptureScreenshot,
    bridgeSupportsScreenshot,
    isPreviewSameOrigin,
    handleInspectorCaptureAdded,
    handleElementCaptureNoteChange,
    handleRemoveElementCapture,
    resetElementCaptures,
    resetReportDraftState,
  };
}

export default usePreviewReportSession;
