import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, useId } from 'react';
import type { MutableRefObject } from 'react';
import type { ChangeEvent, MouseEvent, PointerEvent as ReactPointerEvent } from 'react';
import { ensureDataUrl } from '@/utils/dataUrl';
import { logger } from '@/services/logger';
import type { ReportElementCapture } from '../report/reportTypes';
import type { BridgeInspectState, UseIframeBridgeReturn } from '@/hooks/useIframeBridge';
import type { BridgeInspectHoverPayload, BridgeInspectResultPayload, BridgeInspectRect } from '@vrooli/iframe-bridge';

const INSPECTOR_FLOATING_MARGIN = 12;
type InspectorPosition = { x: number; y: number };
const DEFAULT_INSPECTOR_POSITION: InspectorPosition = { x: 24, y: INSPECTOR_FLOATING_MARGIN };
const DRAG_THRESHOLD_PX = 3;

export type InspectorScreenshot = {
  dataUrl: string;
  width: number;
  height: number;
  filename: string;
  capturedAt: number;
  note?: string | null;
  mode?: string | null;
  clip?: { x: number; y: number; width: number; height: number } | null;
};

type UsePreviewInspectorParams = {
  inspectState: BridgeInspectState;
  startInspect: () => boolean;
  stopInspect: () => boolean;
  requestScreenshot: UseIframeBridgeReturn['requestScreenshot'];
  previewUrl: string | null;
  currentAppIdentifier: string | null;
  iframeRef: React.RefObject<HTMLIFrameElement | null>;
  previewViewRef: React.RefObject<HTMLDivElement | null>;
  previewViewNode: HTMLDivElement | null;
  onCaptureAdd: (capture: ReportElementCapture) => void;
};

type DragState = {
  pointerId: number;
  offsetX: number;
  offsetY: number;
  startClientX: number;
  startClientY: number;
  pointerCaptured: boolean;
  dragging: boolean;
  width: number;
  height: number;
  containerRect: DOMRect;
};

type InspectMessageLock = {
  locked: boolean;
  timer: number | null;
};

export type PreviewInspectorState = {
  shouldRenderInspectorDialog: boolean;
  inspectorDialogRef: MutableRefObject<HTMLDivElement | null>;
  inspectorDialogPosition: InspectorPosition;
  isInspectorDragging: boolean;
  inspectorDialogTitleId: string;
  inspectorDetailsSectionId: string;
  inspectorDetailsContentId: string;
  inspectorScreenshotSectionId: string;
  inspectorScreenshotContentId: string;
  inspectorReportNoteId: string;
  inspectorDetailsExpanded: boolean;
  toggleInspectorDetails: () => void;
  inspectStatusMessage: string | null;
  inspectCopyFeedback: 'selector' | 'text' | null;
  inspectTarget: BridgeInspectHoverPayload | BridgeInspectResultPayload | null;
  inspectMeta: (BridgeInspectHoverPayload | BridgeInspectResultPayload)['meta'] | null;
  inspectRect: BridgeInspectRect | null;
  inspectSizeLabel: string | null;
  inspectPositionLabel: string | null;
  inspectClassTokens: string[];
  inspectSelectorValue: string | null;
  inspectLabelValue: string | null;
  inspectTextValue: string | null;
  inspectTextPreview: string | null;
  inspectAriaDescription: string | null;
  inspectTitleValue: string | null;
  inspectMethodLabel: string | null;
  inspectorScreenshot: InspectorScreenshot | null;
  isInspectorScreenshotCapturing: boolean;
  inspectorScreenshotExpanded: boolean;
  toggleInspectorScreenshotExpanded: () => void;
  inspectorCaptureNote: string;
  handleInspectorCaptureNoteChange: (event: ChangeEvent<HTMLInputElement>) => void;
  handleAddInspectorCaptureToReport: () => void;
  handleCopySelector: () => void;
  handleCopyText: () => void;
  handleDownloadInspectorScreenshot: () => void;
  handleToggleInspectMode: () => void;
  handleInspectorDialogClose: () => void;
  handleInspectorPointerDown: (event: ReactPointerEvent<HTMLDivElement>) => void;
  handleInspectorPointerMove: (event: ReactPointerEvent<HTMLDivElement>) => void;
  handleInspectorPointerUp: (event: ReactPointerEvent<HTMLDivElement>) => void;
  handleInspectorClickCapture: (event: MouseEvent<HTMLDivElement>) => void;
};

export const usePreviewInspector = ({
  inspectState,
  startInspect,
  stopInspect,
  requestScreenshot,
  previewUrl,
  currentAppIdentifier,
  iframeRef,
  previewViewRef,
  previewViewNode,
  onCaptureAdd,
}: UsePreviewInspectorParams): PreviewInspectorState => {
  const [inspectStatusMessage, setInspectStatusMessage] = useState<string | null>(null);
  const [inspectCopyFeedback, setInspectCopyFeedback] = useState<'selector' | 'text' | null>(null);
  const [isInspectorDialogOpen, setIsInspectorDialogOpen] = useState(false);
  const [inspectorDetailsExpanded, setInspectorDetailsExpanded] = useState(false);
  const [inspectorScreenshot, setInspectorScreenshot] = useState<InspectorScreenshot | null>(null);
  const [isInspectorScreenshotCapturing, setIsInspectorScreenshotCapturing] = useState(false);
  const [inspectorScreenshotExpanded, setInspectorScreenshotExpanded] = useState(false);
  const [inspectorDialogPosition, setInspectorDialogPosition] = useState<InspectorPosition>(() => ({ ...DEFAULT_INSPECTOR_POSITION }));
  const [isInspectorDragging, setIsInspectorDragging] = useState(false);
  const [inspectorCaptureNote, setInspectorCaptureNote] = useState('');

  const inspectorDialogRef = useRef<HTMLDivElement | null>(null);
  const inspectorDefaultPositionAppliedRef = useRef(false);
  const inspectorCaptureTokenRef = useRef(0);
  const captureElementScreenshotRef = useRef<(() => Promise<void>) | null>(null);
  const lastCapturedResultSignatureRef = useRef<string | null>(null);
  const inspectorDragStateRef = useRef<DragState | null>(null);
  const inspectorSuppressClickRef = useRef(false);
  const inspectMessageLockRef = useRef<InspectMessageLock>({ locked: false, timer: null });

  const inspectorDialogTitleId = useId();
  const inspectorDetailsSectionId = useId();
  const inspectorScreenshotSectionId = useId();
  const inspectorReportNoteId = useId();
  const inspectorDetailsContentId = `${inspectorDetailsSectionId}-content`;
  const inspectorScreenshotContentId = `${inspectorScreenshotSectionId}-content`;

  useEffect(() => {
    if (!inspectCopyFeedback) {
      return;
    }
    if (typeof window === 'undefined') {
      return;
    }
    const timeoutId = window.setTimeout(() => setInspectCopyFeedback(null), 1800);
    return () => window.clearTimeout(timeoutId);
  }, [inspectCopyFeedback]);

  useEffect(() => {
    if (!isInspectorDialogOpen) {
      return;
    }
    setInspectorDetailsExpanded(false);
    setInspectorScreenshotExpanded(false);
  }, [isInspectorDialogOpen]);

  const inspectTarget = useMemo(() => inspectState.hover ?? inspectState.result, [inspectState.hover, inspectState.result]);
  const inspectMeta = inspectTarget?.meta ?? null;
  const inspectRect = inspectTarget?.documentRect ?? null;
  const inspectSizeLabel = inspectRect
    ? `${Math.round(inspectRect.width)} × ${Math.round(inspectRect.height)} px`
    : null;
  const inspectPositionLabel = inspectRect
    ? `x ${Math.round(inspectRect.x)}, y ${Math.round(inspectRect.y)}`
    : null;
  const inspectClassTokens = useMemo<string[]>(
    () => (inspectMeta?.classes ? inspectMeta.classes.slice(0, 3) : []),
    [inspectMeta?.classes],
  );
  const inspectSelectorValue = inspectMeta?.selector ?? null;
  const inspectLabelValue = inspectMeta?.ariaLabel ?? inspectMeta?.label ?? null;
  const inspectTextValue = inspectMeta?.text ?? null;
  const inspectTextPreview = inspectTextValue && inspectTextValue.length > 220
    ? `${inspectTextValue.slice(0, 220)}…`
    : inspectTextValue;
  const inspectAriaDescription = inspectMeta?.ariaDescription ?? null;
  const inspectTitleValue = inspectMeta?.title ?? null;
  const inspectResultMethod = inspectState.result?.method ?? null;
  let inspectMethodLabel: string | null = null;
  if (inspectResultMethod === 'keyboard') {
    inspectMethodLabel = 'Keyboard selection';
  } else if (inspectResultMethod === 'pointer') {
    inspectMethodLabel = 'Pointer selection';
  } else if (inspectState.active && inspectTarget?.pointerType) {
    inspectMethodLabel = `${inspectTarget.pointerType} pointer`;
  }

  const resolvePreviewBackgroundColor = useCallback(() => {
    if (typeof window === 'undefined') {
      return undefined;
    }

    const iframe = iframeRef.current;
    if (iframe) {
      try {
        const iframeWindow = iframe.contentWindow;
        const iframeDocument = iframe.contentDocument ?? iframeWindow?.document ?? null;
        if (iframeDocument && iframeWindow) {
          const candidates: Element[] = [];
          if (iframeDocument.body) {
            candidates.push(iframeDocument.body);
          }
          if (iframeDocument.documentElement && iframeDocument.documentElement !== iframeDocument.body) {
            candidates.push(iframeDocument.documentElement);
          }
          for (const element of candidates) {
            const style = iframeWindow.getComputedStyle(element);
            if (!style) {
              continue;
            }
            const color = style.backgroundColor;
            if (color && color !== 'rgba(0, 0, 0, 0)' && color.toLowerCase() !== 'transparent') {
              return color;
            }
          }
        }
      } catch (error) {
        logger.debug('Unable to inspect iframe background color', error);
      }
    }

    if (previewViewRef.current) {
      try {
        const style = window.getComputedStyle(previewViewRef.current);
        const color = style.backgroundColor;
        if (color && color !== 'rgba(0, 0, 0, 0)' && color.toLowerCase() !== 'transparent') {
          return color;
        }
      } catch (error) {
        logger.debug('Unable to inspect preview container background color', error);
      }
    }

    return undefined;
  }, [iframeRef, previewViewRef]);

  const clampInspectorPosition = useCallback((
    x: number,
    y: number,
    dimensions?: { width?: number; height?: number; containerRect?: DOMRect },
  ): InspectorPosition => {
    const container = previewViewRef.current;
    const containerRect = dimensions?.containerRect ?? container?.getBoundingClientRect();
    const width = dimensions?.width ?? inspectorDialogRef.current?.offsetWidth ?? 360;
    const height = dimensions?.height ?? inspectorDialogRef.current?.offsetHeight ?? 360;
    if (!containerRect) {
      return { x, y };
    }
    const maxX = Math.max(INSPECTOR_FLOATING_MARGIN, containerRect.width - width - INSPECTOR_FLOATING_MARGIN);
    const maxY = Math.max(INSPECTOR_FLOATING_MARGIN, containerRect.height - height - INSPECTOR_FLOATING_MARGIN);
    const clampedX = Math.min(Math.max(x, INSPECTOR_FLOATING_MARGIN), maxX);
    const clampedY = Math.min(Math.max(y, INSPECTOR_FLOATING_MARGIN), maxY);
    return { x: clampedX, y: clampedY };
  }, [previewViewRef]);

  const clearInspectMessageLock = useCallback(() => {
    if (inspectMessageLockRef.current.timer !== null && typeof window !== 'undefined') {
      window.clearTimeout(inspectMessageLockRef.current.timer);
    }
    inspectMessageLockRef.current.timer = null;
    inspectMessageLockRef.current.locked = false;
  }, []);

  const setLockedInspectMessage = useCallback((message: string, durationMs = 3200) => {
    clearInspectMessageLock();
    setInspectStatusMessage(message);
    if (typeof window === 'undefined') {
      return;
    }
    inspectMessageLockRef.current.locked = true;
    inspectMessageLockRef.current.timer = window.setTimeout(() => {
      inspectMessageLockRef.current.locked = false;
      inspectMessageLockRef.current.timer = null;
      setInspectStatusMessage(null);
    }, durationMs);
  }, [clearInspectMessageLock]);

  const handleInspectorCaptureNoteChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    setInspectorCaptureNote(event.target.value);
  }, []);

  const handleAddInspectorCaptureToReport = useCallback(() => {
    if (!inspectorScreenshot) {
      setLockedInspectMessage('Capture an element before adding it to the report.', 3200);
      return;
    }

    const base64Payload = inspectorScreenshot.dataUrl.includes(',')
      ? inspectorScreenshot.dataUrl.split(',')[1]
      : inspectorScreenshot.dataUrl;

    if (!base64Payload) {
      setLockedInspectMessage('Unable to read captured image data.', 3200);
      return;
    }

    const captureId = typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
      ? crypto.randomUUID()
      : `capture-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;

    const metadataClasses = Array.isArray(inspectMeta?.classes)
      ? inspectMeta.classes.slice(0, 6)
      : [];

    const capture: ReportElementCapture = {
      id: captureId,
      type: 'element',
      width: inspectorScreenshot.width,
      height: inspectorScreenshot.height,
      data: base64Payload,
      createdAt: inspectorScreenshot.capturedAt ?? Date.now(),
      filename: inspectorScreenshot.filename ?? null,
      clip: inspectorScreenshot.clip ?? null,
      mode: inspectorScreenshot.mode ?? null,
      note: inspectorCaptureNote.trim(),
      metadata: {
        selector: inspectSelectorValue,
        tagName: inspectMeta?.tag ?? null,
        elementId: inspectMeta?.id ?? null,
        classes: metadataClasses,
        label: inspectLabelValue,
        ariaDescription: inspectAriaDescription,
        title: inspectTitleValue,
        role: inspectMeta?.role ?? null,
        text: inspectTextValue ?? null,
        boundingBox: inspectRect
          ? {
              x: inspectRect.x,
              y: inspectRect.y,
              width: inspectRect.width,
              height: inspectRect.height,
            }
          : null,
      },
    };

    onCaptureAdd(capture);
    setInspectorCaptureNote('');
    setLockedInspectMessage('Element capture added to report.', 3200);
  }, [
    inspectorScreenshot,
    inspectorCaptureNote,
    inspectMeta,
    inspectSelectorValue,
    inspectLabelValue,
    inspectAriaDescription,
    inspectTitleValue,
    inspectTextValue,
    inspectRect,
    onCaptureAdd,
    setLockedInspectMessage,
  ]);

  const handleInspectorPointerDown = useCallback((event: ReactPointerEvent<HTMLDivElement>) => {
    if (!isInspectorDialogOpen) {
      return;
    }
    if (event.button !== 0) {
      return;
    }
    const target = event.target as HTMLElement | null;
    if (!target) {
      return;
    }
    const handleNode = target.closest('.preview-inspector__handle');
    if (!handleNode) {
      return;
    }
    if (target.closest('button, a, input, textarea, select')) {
      return;
    }
    const dialog = inspectorDialogRef.current;
    const container = previewViewRef.current;
    if (!dialog || !container) {
      return;
    }
    const containerRect = container.getBoundingClientRect();
    const dialogRect = dialog.getBoundingClientRect();
    inspectorDragStateRef.current = {
      pointerId: event.pointerId,
      offsetX: event.clientX - dialogRect.left,
      offsetY: event.clientY - dialogRect.top,
      startClientX: event.clientX,
      startClientY: event.clientY,
      pointerCaptured: false,
      dragging: false,
      width: dialogRect.width,
      height: dialogRect.height,
      containerRect,
    };
    inspectorSuppressClickRef.current = false;
  }, [isInspectorDialogOpen, previewViewRef]);

  const handleInspectorPointerMove = useCallback((event: ReactPointerEvent<HTMLDivElement>) => {
    const state = inspectorDragStateRef.current;
    if (!state || state.pointerId !== event.pointerId) {
      return;
    }
    if (!state.pointerCaptured && typeof event.currentTarget.setPointerCapture === 'function') {
      event.currentTarget.setPointerCapture(event.pointerId);
      state.pointerCaptured = true;
    }
    const deltaX = Math.abs(event.clientX - state.startClientX);
    const deltaY = Math.abs(event.clientY - state.startClientY);
    if (!state.dragging && (deltaX > DRAG_THRESHOLD_PX || deltaY > DRAG_THRESHOLD_PX)) {
      state.dragging = true;
      setIsInspectorDragging(true);
      inspectorSuppressClickRef.current = true;
    }
    if (!state.dragging) {
      return;
    }
    const rawX = event.clientX - state.offsetX - state.containerRect.left;
    const rawY = event.clientY - state.offsetY - state.containerRect.top;
    const next = clampInspectorPosition(rawX, rawY, state);
    setInspectorDialogPosition(prev => (prev.x === next.x && prev.y === next.y ? prev : next));
  }, [clampInspectorPosition]);

  const handleInspectorPointerUp = useCallback((event: ReactPointerEvent<HTMLDivElement>) => {
    const state = inspectorDragStateRef.current;
    if (!state || state.pointerId !== event.pointerId) {
      return;
    }
    inspectorDragStateRef.current = null;
    if (state.pointerCaptured && typeof event.currentTarget.releasePointerCapture === 'function') {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
    setTimeout(() => {
      inspectorSuppressClickRef.current = false;
    }, 0);
    setIsInspectorDragging(false);
  }, []);

  const handleInspectorClickCapture = useCallback((event: MouseEvent<HTMLDivElement>) => {
    if (inspectorSuppressClickRef.current) {
      event.stopPropagation();
      event.preventDefault();
      inspectorSuppressClickRef.current = false;
    }
  }, []);

  const handleInspectorDialogClose = useCallback(() => {
    setIsInspectorDialogOpen(false);
    setIsInspectorDragging(false);
    inspectorDragStateRef.current = null;
    inspectorSuppressClickRef.current = false;
    stopInspect();
    clearInspectMessageLock();
  }, [clearInspectMessageLock, stopInspect]);

  const handleCopyValue = useCallback(async (value: string, kind: 'selector' | 'text') => {
    try {
      if (typeof navigator !== 'undefined' && typeof navigator.clipboard?.writeText === 'function') {
        await navigator.clipboard.writeText(value);
      } else if (typeof document !== 'undefined') {
        const textarea = document.createElement('textarea');
        textarea.value = value;
        textarea.setAttribute('readonly', '');
        textarea.style.position = 'absolute';
        textarea.style.left = '-9999px';
        document.body.appendChild(textarea);
        textarea.select();
        const successful = document.execCommand('copy');
        document.body.removeChild(textarea);
        if (!successful) {
          throw new Error('execCommand failed');
        }
      } else {
        throw new Error('clipboard-unavailable');
      }
      setInspectCopyFeedback(kind);
      setLockedInspectMessage(kind === 'selector' ? 'Selector copied to clipboard.' : 'Text snippet copied to clipboard.', 2200);
    } catch (error) {
      logger.warn('Failed to copy inspector value', error);
      setLockedInspectMessage('Unable to copy to clipboard.', 3200);
    }
  }, [setLockedInspectMessage]);

  const handleCopySelector = useCallback(() => {
    if (!inspectSelectorValue) {
      setLockedInspectMessage('Selector unavailable for this element.', 2600);
      return;
    }
    void handleCopyValue(inspectSelectorValue, 'selector');
  }, [handleCopyValue, inspectSelectorValue, setLockedInspectMessage]);

  const handleCopyText = useCallback(() => {
    if (!inspectTextValue) {
      setLockedInspectMessage('Text snippet unavailable for this element.', 2600);
      return;
    }
    void handleCopyValue(inspectTextValue, 'text');
  }, [handleCopyValue, inspectTextValue, setLockedInspectMessage]);

  const handleCaptureElementScreenshot = useCallback(async () => {
    if (!inspectRect) {
      setLockedInspectMessage('Element bounds unavailable for screenshots.', 3000);
      return;
    }
    const captureToken = Date.now();
    inspectorCaptureTokenRef.current = captureToken;
    setIsInspectorScreenshotCapturing(true);
    try {
      const scale = typeof window !== 'undefined' ? window.devicePixelRatio || 1 : 1;
      const result = await requestScreenshot({
        mode: 'clip',
        clip: inspectRect,
        selector: inspectSelectorValue ?? undefined,
        backgroundColor: resolvePreviewBackgroundColor(),
        scale,
      });
      const sanitizedSelector = inspectSelectorValue
        ? inspectSelectorValue.replace(/[^a-z0-9-_]+/gi, '-').slice(0, 40)
        : 'element';
      const filename = `${(currentAppIdentifier ?? 'preview').replace(/[^a-z0-9-_]+/gi, '-').slice(0, 24)}-${sanitizedSelector || 'element'}-${Date.now()}.png`;
      const dataUrl = ensureDataUrl(result.data);
      if (!dataUrl) {
        setLockedInspectMessage('Failed to capture element screenshot.', 3400);
        return;
      }
      const capturedAt = Date.now();
      setInspectorScreenshot({
        dataUrl,
        width: result.width,
        height: result.height,
        filename,
        capturedAt,
        note: result.note ?? null,
        mode: result.mode ?? null,
        clip: result.clip ?? null,
      });
      setInspectorScreenshotExpanded(true);
      setLockedInspectMessage('Element screenshot captured. Use the download button to save it.', 3600);
    } catch (error) {
      logger.error('Failed to capture element screenshot', error);
      setLockedInspectMessage('Failed to capture element screenshot.', 3400);
    } finally {
      if (inspectorCaptureTokenRef.current === captureToken) {
        setIsInspectorScreenshotCapturing(false);
      }
    }
  }, [
    currentAppIdentifier,
    inspectRect,
    inspectSelectorValue,
    requestScreenshot,
    resolvePreviewBackgroundColor,
    setLockedInspectMessage,
  ]);

  const handleDownloadInspectorScreenshot = useCallback(() => {
    if (!inspectorScreenshot) {
      setLockedInspectMessage('Capture an element before downloading.', 3200);
      return;
    }
    if (typeof document === 'undefined') {
      return;
    }
    try {
      const anchor = document.createElement('a');
      anchor.href = inspectorScreenshot.dataUrl;
      anchor.download = inspectorScreenshot.filename;
      document.body.appendChild(anchor);
      anchor.click();
      document.body.removeChild(anchor);
      setLockedInspectMessage('Element screenshot downloaded.', 3200);
    } catch (error) {
      logger.error('Failed to download element screenshot', error);
      setLockedInspectMessage('Unable to download the screenshot.', 3200);
    }
  }, [inspectorScreenshot, setLockedInspectMessage]);

  useEffect(() => {
    captureElementScreenshotRef.current = handleCaptureElementScreenshot;
  }, [handleCaptureElementScreenshot]);

  useEffect(() => {
    if (inspectState.active) {
      lastCapturedResultSignatureRef.current = null;
    }
  }, [inspectState.active]);

  useEffect(() => {
    if (!inspectState.result || inspectState.lastReason !== 'complete') {
      return;
    }
    const { meta, documentRect, method } = inspectState.result;
    const signatureParts = [
      meta?.selector ?? '',
      meta?.id ?? '',
      Array.isArray(meta?.classes) ? meta.classes.join('.') : '',
      documentRect ? Math.round(documentRect.x) : 0,
      documentRect ? Math.round(documentRect.y) : 0,
      documentRect ? Math.round(documentRect.width) : 0,
      documentRect ? Math.round(documentRect.height) : 0,
      method ?? '',
    ];
    const signature = signatureParts.join('|');
    if (lastCapturedResultSignatureRef.current === signature) {
      return;
    }
    lastCapturedResultSignatureRef.current = signature;
    const capture = captureElementScreenshotRef.current;
    if (capture) {
      void capture();
    }
  }, [inspectState.lastReason, inspectState.result]);

  const handleToggleInspectMode = useCallback(() => {
    if (inspectState.active) {
      const stopped = stopInspect();
      if (!stopped) {
        setLockedInspectMessage('Unable to exit inspect mode.', 3200);
      }
      return;
    }
    if (!inspectState.supported) {
      setLockedInspectMessage('Element inspector requires the latest bridge in the previewed app.', 3600);
      return;
    }
    if (!previewUrl) {
      setLockedInspectMessage('Load the preview before inspecting elements.', 3200);
      return;
    }
    const started = startInspect();
    if (!started) {
      setLockedInspectMessage('Element inspector is unavailable for this preview.', 3600);
      return;
    }
    clearInspectMessageLock();
    setInspectStatusMessage('Select an element in the preview. Press Esc to cancel.');
  }, [
    clearInspectMessageLock,
    inspectState.active,
    inspectState.supported,
    previewUrl,
    setLockedInspectMessage,
    startInspect,
    stopInspect,
  ]);

  useEffect(() => {
    if (inspectState.active && !isInspectorDialogOpen) {
      setIsInspectorDialogOpen(true);
    }
  }, [inspectState.active, isInspectorDialogOpen]);

  useEffect(() => {
    if (inspectStatusMessage && !isInspectorDialogOpen) {
      setIsInspectorDialogOpen(true);
    }
  }, [inspectStatusMessage, isInspectorDialogOpen]);

  useEffect(() => {
    if (!isInspectorDialogOpen) {
      inspectorDragStateRef.current = null;
      setIsInspectorDragging(false);
      return;
    }

    setInspectorDialogPosition(prev => clampInspectorPosition(prev.x, prev.y));

    if (typeof ResizeObserver === 'undefined') {
      return;
    }

    const container = previewViewRef.current;
    if (!container) {
      return;
    }

    const observer = new ResizeObserver(() => {
      setInspectorDialogPosition(prev => clampInspectorPosition(prev.x, prev.y));
    });
    observer.observe(container);
    return () => {
      observer.disconnect();
    };
  }, [clampInspectorPosition, isInspectorDialogOpen, previewViewRef]);

  useLayoutEffect(() => {
    if (!isInspectorDialogOpen || inspectorDefaultPositionAppliedRef.current) {
      return;
    }
    const container = previewViewRef.current;
    const dialog = inspectorDialogRef.current;
    if (!container || !dialog) {
      return;
    }
    const containerRect = container.getBoundingClientRect();
    const width = dialog.offsetWidth;
    const height = dialog.offsetHeight;
    const target = clampInspectorPosition(
      containerRect.width - width - INSPECTOR_FLOATING_MARGIN,
      INSPECTOR_FLOATING_MARGIN,
      { width, height, containerRect },
    );
    inspectorDefaultPositionAppliedRef.current = true;
    setInspectorDialogPosition(target);
  }, [clampInspectorPosition, isInspectorDialogOpen, previewViewRef]);

  useEffect(() => {
    if (!previewViewNode) {
      return;
    }
    setInspectorDialogPosition(prev => clampInspectorPosition(prev.x, prev.y));
  }, [clampInspectorPosition, previewViewNode]);

  useEffect(() => {
    if (!inspectState.supported) {
      return;
    }
    if (inspectMessageLockRef.current.locked) {
      return;
    }
    if (inspectState.error) {
      setLockedInspectMessage(`Element inspector error: ${inspectState.error}`, 3600);
      return;
    }
    if (inspectState.active) {
      clearInspectMessageLock();
      setInspectStatusMessage('Select an element in the preview. Press Esc to cancel.');
      return;
    }
    if (inspectState.lastReason === 'complete') {
      setLockedInspectMessage('Element captured. Use “Inspect element” to capture another.', 3400);
      return;
    }
    if (inspectState.lastReason === 'cancel') {
      setLockedInspectMessage('Selection cancelled.', 2400);
      return;
    }
    if (!inspectTarget) {
      clearInspectMessageLock();
      setInspectStatusMessage(null);
    }
  }, [
    clearInspectMessageLock,
    inspectState.active,
    inspectState.error,
    inspectState.lastReason,
    inspectState.supported,
    inspectTarget,
    setLockedInspectMessage,
  ]);

  useEffect(() => {
    if (!previewUrl && inspectState.active) {
      stopInspect();
    }
  }, [inspectState.active, previewUrl, stopInspect]);

  useEffect(() => {
    if (!inspectState.active || typeof window === 'undefined') {
      return () => {};
    }

    const handleKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        stopInspect();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [inspectState.active, stopInspect]);

  useEffect(() => {
    return () => {
      stopInspect();
      clearInspectMessageLock();
    };
  }, [clearInspectMessageLock, stopInspect]);

  const toggleInspectorDetails = useCallback(() => {
    setInspectorDetailsExpanded(prev => !prev);
  }, []);

  const toggleInspectorScreenshotExpanded = useCallback(() => {
    setInspectorScreenshotExpanded(prev => !prev);
  }, []);

  return {
    shouldRenderInspectorDialog: isInspectorDialogOpen,
    inspectorDialogRef,
    inspectorDialogPosition,
    isInspectorDragging,
    inspectorDialogTitleId,
    inspectorDetailsSectionId,
    inspectorDetailsContentId,
    inspectorScreenshotSectionId,
    inspectorScreenshotContentId,
    inspectorReportNoteId,
    inspectorDetailsExpanded,
    toggleInspectorDetails,
    inspectStatusMessage,
    inspectCopyFeedback,
    inspectTarget,
    inspectMeta,
    inspectRect,
    inspectSizeLabel,
    inspectPositionLabel,
    inspectClassTokens,
    inspectSelectorValue,
    inspectLabelValue,
    inspectTextValue,
    inspectTextPreview,
    inspectAriaDescription,
    inspectTitleValue,
    inspectMethodLabel,
    inspectorScreenshot,
    isInspectorScreenshotCapturing,
    inspectorScreenshotExpanded,
    toggleInspectorScreenshotExpanded,
    inspectorCaptureNote,
    handleInspectorCaptureNoteChange,
    handleAddInspectorCaptureToReport,
    handleCopySelector,
    handleCopyText,
    handleDownloadInspectorScreenshot,
    handleToggleInspectMode,
    handleInspectorDialogClose,
    handleInspectorPointerDown,
    handleInspectorPointerMove,
    handleInspectorPointerUp,
    handleInspectorClickCapture,
  };
};

export default usePreviewInspector;
