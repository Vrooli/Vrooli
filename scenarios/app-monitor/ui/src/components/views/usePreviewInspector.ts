import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, useId } from 'react';
import type { MutableRefObject } from 'react';
import type { ChangeEvent, MouseEvent, PointerEvent as ReactPointerEvent } from 'react';
import { ensureDataUrl } from '@/utils/dataUrl';
import { logger } from '@/services/logger';
import type { ReportElementCapture } from '../report/reportTypes';
import type { BridgeInspectState, UseIframeBridgeReturn } from '@/hooks/useIframeBridge';
import type { BridgeInspectHoverPayload, BridgeInspectResultPayload, BridgeInspectRect, BridgeInspectAncestorMeta } from '@vrooli/iframe-bridge';

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
  setInspectTargetIndex: (index: number) => boolean;
  shiftInspectTarget: (delta: number) => boolean;
  requestScreenshot: UseIframeBridgeReturn['requestScreenshot'];
  previewUrl: string | null;
  currentAppIdentifier: string | null;
  iframeRef: React.RefObject<HTMLIFrameElement | null>;
  previewViewRef: React.RefObject<HTMLDivElement | null>;
  previewViewNode: HTMLDivElement | null;
  onCaptureAdd: (capture: ReportElementCapture) => void;
  onViewReportRequest: () => void;
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

type InspectorReportStatus = {
  message: string;
  note: string;
  captureId: string;
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
  inspectActiveChipLabel: string | null;
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
  inspectAncestors: BridgeInspectAncestorMeta[];
  inspectAncestorIndex: number;
  inspectAncestorCanStepParent: boolean;
  inspectAncestorCanStepChild: boolean;
  handleInspectorAncestorStep: (delta: number) => void;
  handleInspectorAncestorSelect: (index: number) => void;
  inspectorScreenshot: InspectorScreenshot | null;
  isInspectorScreenshotCapturing: boolean;
  inspectorScreenshotExpanded: boolean;
  toggleInspectorScreenshotExpanded: () => void;
  inspectorCaptureNote: string;
  handleInspectorCaptureNoteChange: (event: ChangeEvent<HTMLInputElement>) => void;
  handleAddInspectorCaptureToReport: () => void;
  inspectorReportStatus: InspectorReportStatus | null;
  handleInspectorViewReport: () => void;
  handleInspectorInspectAnother: () => void;
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
  setInspectTargetIndex,
  shiftInspectTarget,
  requestScreenshot,
  previewUrl,
  currentAppIdentifier,
  iframeRef,
  previewViewRef,
  previewViewNode,
  onCaptureAdd,
  onViewReportRequest,
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
  const [inspectorReportStatus, setInspectorReportStatus] = useState<InspectorReportStatus | null>(null);
  const [inspectorAncestorIndex, setInspectorAncestorIndex] = useState(0);

  const inspectorDialogRef = useRef<HTMLDivElement | null>(null);
  const inspectorDefaultPositionAppliedRef = useRef(false);
  const inspectorCaptureTokenRef = useRef(0);
  const captureElementScreenshotRef = useRef<(() => Promise<void>) | null>(null);
  const lastCapturedResultSignatureRef = useRef<string | null>(null);
  const inspectorDragStateRef = useRef<DragState | null>(null);
  const inspectorSuppressClickRef = useRef(false);
  const inspectMessageLockRef = useRef<InspectMessageLock>({ locked: false, timer: null });
  const inspectorAutoOpenSuppressedRef = useRef(false);
  const lastInspectorScreenshotAtRef = useRef<number | null>(null);
  const inspectorAncestorSignatureRef = useRef<string | null>(null);
  const previousAncestorIndexRef = useRef<number>(0);

  const inspectorDialogTitleId = useId();
  const inspectorDetailsSectionId = useId();
  const inspectorScreenshotSectionId = useId();
  const inspectorReportNoteId = useId();
  const inspectorDetailsContentId = `${inspectorDetailsSectionId}-content`;
  const inspectorScreenshotContentId = `${inspectorScreenshotSectionId}-content`;
  const prefersCoarsePointer = useMemo(() => {
    if (typeof window === 'undefined') {
      return false;
    }
    try {
      if (typeof navigator !== 'undefined' && typeof navigator.maxTouchPoints === 'number' && navigator.maxTouchPoints > 0) {
        return true;
      }
      if (typeof window.matchMedia === 'function' && window.matchMedia('(pointer: coarse)').matches) {
        return true;
      }
    } catch (error) {
      logger.debug('Failed to determine pointer characteristics', error);
    }
    return false;
  }, []);

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

  const inspectAncestors = useMemo<BridgeInspectAncestorMeta[]>(() => {
    if (!Array.isArray(inspectTarget?.ancestors) || inspectTarget.ancestors.length === 0) {
      return [];
    }
    return inspectTarget.ancestors
      .filter((ancestor): ancestor is BridgeInspectAncestorMeta => Boolean(ancestor))
      .map((ancestor, depth) => ({
        ...ancestor,
        depth: typeof ancestor.depth === 'number' ? ancestor.depth : depth,
      }));
  }, [inspectTarget?.ancestors]);

  const inspectAncestorSignature = useMemo(() => {
    if (inspectAncestors.length === 0) {
      return null;
    }
    return inspectAncestors.map((ancestor, index) => {
      const selector = ancestor.selector ?? ancestor.tag ?? `node-${index}`;
      const rect = ancestor.documentRect ?? ancestor.rect ?? null;
      const rectPart = rect
        ? `${Math.round(rect.x)}:${Math.round(rect.y)}:${Math.round(rect.width)}:${Math.round(rect.height)}`
        : '0:0:0:0';
      return `${selector}@${rectPart}`;
    }).join('|');
  }, [inspectAncestors]);

  useEffect(() => {
    if (inspectAncestors.length === 0) {
      inspectorAncestorSignatureRef.current = null;
      if (inspectorAncestorIndex !== 0) {
        setInspectorAncestorIndex(0);
      }
      return;
    }

    const payloadIndex = inspectTarget?.selectedAncestorIndex ?? 0;
    const clamped = Math.min(Math.max(payloadIndex, 0), inspectAncestors.length - 1);
    const signature = inspectAncestorSignature;
    const signatureChanged = inspectorAncestorSignatureRef.current !== signature;
    inspectorAncestorSignatureRef.current = signature;

    if (signatureChanged || inspectorAncestorIndex !== clamped) {
      setInspectorAncestorIndex(clamped);
    }
  }, [inspectAncestors, inspectorAncestorIndex, inspectAncestorSignature, inspectTarget?.selectedAncestorIndex]);

  const effectiveAncestorIndex = inspectAncestors.length > 0
    ? Math.min(Math.max(inspectorAncestorIndex, 0), inspectAncestors.length - 1)
    : 0;

  const selectedAncestor = inspectAncestors.length > 0
    ? inspectAncestors[effectiveAncestorIndex]
    : null;

  const inspectMeta = selectedAncestor ?? inspectTarget?.meta ?? null;

  const inspectRect = selectedAncestor?.documentRect
    ?? inspectTarget?.documentRect
    ?? null;

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

  const inspectActiveChipLabel = useMemo(() => {
    if (!inspectState.active) {
      return null;
    }
    if (!inspectMeta) {
      return 'Move finger to highlight an element';
    }
    const preferred = [
      inspectMeta.label,
      inspectMeta.ariaLabel,
      inspectMeta.title,
    ].find(value => value && value.trim().length > 0);
    if (preferred) {
      return preferred.trim();
    }
    const textValue = inspectMeta.text?.trim();
    if (textValue) {
      return textValue.length > 160 ? `${textValue.slice(0, 160)}…` : textValue;
    }
    const selector = inspectMeta.selector?.trim();
    if (selector) {
      return selector.length > 160 ? `${selector.slice(0, 160)}…` : selector;
    }
    const tokens: string[] = [];
    if (inspectMeta.tag) {
      tokens.push(inspectMeta.tag.toLowerCase());
    }
    if (inspectMeta.id) {
      tokens.push(`#${inspectMeta.id}`);
    }
    if (Array.isArray(inspectMeta.classes) && inspectMeta.classes.length > 0) {
      tokens.push(`.${inspectMeta.classes[0]}`);
    }
    if (tokens.length > 0) {
      return tokens.join('');
    }
    return 'Element';
  }, [inspectMeta, inspectState.active]);

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

  const hasAncestorNavigation = inspectAncestors.length > 1;

  const inspectAncestorCanStepParent = inspectState.active
    && hasAncestorNavigation
    && effectiveAncestorIndex < inspectAncestors.length - 1;

  const inspectAncestorCanStepChild = inspectState.active
    && hasAncestorNavigation
    && effectiveAncestorIndex > 0;

  const handleInspectorAncestorSelect = useCallback((index: number) => {
    if (!inspectState.active) {
      setLockedInspectMessage('Start element inspection to adjust the selection.', 3200);
      return;
    }
    if (!Number.isFinite(index) || inspectAncestors.length === 0) {
      return;
    }
    if (!hasAncestorNavigation) {
      setLockedInspectMessage('Ancestor navigation is unavailable for this element.', 3200);
      return;
    }
    const clamped = Math.min(Math.max(Math.floor(index), 0), inspectAncestors.length - 1);
    if (clamped === effectiveAncestorIndex) {
      return;
    }
    setInspectorAncestorIndex(clamped);
    const ok = setInspectTargetIndex(clamped);
    if (!ok) {
      setInspectorAncestorIndex(effectiveAncestorIndex);
      setLockedInspectMessage('Unable to adjust selection.', 3200);
    }
  }, [
    effectiveAncestorIndex,
    hasAncestorNavigation,
    inspectAncestors.length,
    inspectState.active,
    setInspectTargetIndex,
    setLockedInspectMessage,
  ]);

  const handleInspectorAncestorStep = useCallback((delta: number) => {
    if (!inspectState.active) {
      setLockedInspectMessage('Start element inspection to adjust the selection.', 3200);
      return;
    }
    if (!Number.isFinite(delta) || delta === 0 || inspectAncestors.length === 0) {
      return;
    }
    if (!hasAncestorNavigation) {
      setLockedInspectMessage('Ancestor navigation is unavailable for this element.', 3200);
      return;
    }
    const normalized = delta > 0 ? 1 : -1;
    const nextIndex = Math.min(
      Math.max(effectiveAncestorIndex + normalized, 0),
      inspectAncestors.length - 1,
    );
    if (nextIndex === effectiveAncestorIndex) {
      return;
    }
    setInspectorAncestorIndex(nextIndex);
    const ok = shiftInspectTarget(normalized);
    if (!ok) {
      setInspectorAncestorIndex(effectiveAncestorIndex);
      setLockedInspectMessage('Unable to adjust selection.', 3200);
    }
  }, [
    effectiveAncestorIndex,
    hasAncestorNavigation,
    inspectAncestors.length,
    inspectState.active,
    setLockedInspectMessage,
    shiftInspectTarget,
  ]);

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

    const trimmedNote = inspectorCaptureNote.trim();

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
      note: trimmedNote,
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
        ...(inspectAncestors.length > 0
          ? {
              ancestorIndex: effectiveAncestorIndex,
              ancestorCount: inspectAncestors.length,
              ancestorTrail: inspectAncestors.map(ancestor => ancestor.selector ?? ancestor.tag ?? null),
            }
          : {}),
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
    setInspectorReportStatus({
      message: 'Element capture added to report.',
      note: trimmedNote,
      captureId,
    });
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
    inspectAncestors,
    effectiveAncestorIndex,
    onCaptureAdd,
    setLockedInspectMessage,
    setInspectorReportStatus,
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
    inspectorAutoOpenSuppressedRef.current = true;
    setIsInspectorDialogOpen(false);
    setIsInspectorDragging(false);
    inspectorDragStateRef.current = null;
    inspectorSuppressClickRef.current = false;
    stopInspect();
    clearInspectMessageLock();
    setInspectorReportStatus(null);
    setInspectorCaptureNote('');
    setInspectorAncestorIndex(0);
  }, [clearInspectMessageLock, stopInspect, setInspectorAncestorIndex]);

  const handleInspectorInspectAnother = useCallback(() => {
    setInspectorReportStatus(null);
    setInspectorCaptureNote('');
    if (inspectState.active) {
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
    inspectorAutoOpenSuppressedRef.current = false;
    setInspectorAncestorIndex(0);
    const started = startInspect();
    if (!started) {
      setLockedInspectMessage('Element inspector is unavailable for this preview.', 3600);
      return;
    }
    clearInspectMessageLock();
    setInspectStatusMessage(prefersCoarsePointer
      ? 'Drag on the preview to target an element and lift your finger to capture. Press Esc to cancel. Use Alt+↑/↓ to adjust the selection when available.'
      : 'Select an element in the preview. Press Esc to cancel. Use Alt+↑/↓ to adjust the selection when available.');
  }, [
    clearInspectMessageLock,
    inspectState.active,
    inspectState.supported,
    prefersCoarsePointer,
    previewUrl,
    setLockedInspectMessage,
    setInspectorAncestorIndex,
    startInspect,
  ]);

  const handleInspectorViewReport = useCallback(() => {
    handleInspectorDialogClose();
    onViewReportRequest();
  }, [handleInspectorDialogClose, onViewReportRequest]);

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
    if (inspectAncestors.length === 0) {
      previousAncestorIndexRef.current = 0;
      return;
    }
    const clamped = Math.min(Math.max(inspectorAncestorIndex, 0), inspectAncestors.length - 1);
    if (previousAncestorIndexRef.current !== clamped) {
      previousAncestorIndexRef.current = clamped;
      if (inspectorScreenshot) {
        setInspectorScreenshot(null);
      }
    }
  }, [inspectAncestors.length, inspectorAncestorIndex, inspectorScreenshot]);

  useEffect(() => {
    const capturedAt = inspectorScreenshot?.capturedAt ?? null;
    if (capturedAt === lastInspectorScreenshotAtRef.current) {
      return;
    }
    lastInspectorScreenshotAtRef.current = capturedAt;
    if (inspectorReportStatus !== null) {
      setInspectorReportStatus(null);
    }
  }, [inspectorReportStatus, inspectorScreenshot]);

  useEffect(() => {
    if (inspectState.active) {
      lastCapturedResultSignatureRef.current = null;
    }
  }, [inspectState.active]);

  useEffect(() => {
    if (!isInspectorDialogOpen || !inspectState.active) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)) {
        return;
      }

      if (event.altKey && !event.shiftKey && !event.ctrlKey && !event.metaKey) {
        if (event.key === 'ArrowUp') {
          event.preventDefault();
          handleInspectorAncestorStep(1);
        } else if (event.key === 'ArrowDown') {
          event.preventDefault();
          handleInspectorAncestorStep(-1);
        }
        return;
      }

      if (!event.altKey && !event.ctrlKey && !event.metaKey && !event.shiftKey) {
        if (event.key === '[') {
          event.preventDefault();
          handleInspectorAncestorStep(1);
        } else if (event.key === ']') {
          event.preventDefault();
          handleInspectorAncestorStep(-1);
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [handleInspectorAncestorStep, inspectState.active, isInspectorDialogOpen]);

  useEffect(() => {
    if (!inspectState.result || inspectState.lastReason !== 'complete') {
      return;
    }
    const { meta, documentRect, method, selectedAncestorIndex } = inspectState.result;
    const signatureParts = [
      meta?.selector ?? '',
      meta?.id ?? '',
      Array.isArray(meta?.classes) ? meta.classes.join('.') : '',
      documentRect ? Math.round(documentRect.x) : 0,
      documentRect ? Math.round(documentRect.y) : 0,
      documentRect ? Math.round(documentRect.width) : 0,
      documentRect ? Math.round(documentRect.height) : 0,
      method ?? '',
      typeof selectedAncestorIndex === 'number' ? selectedAncestorIndex.toString(10) : '',
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
    inspectorAutoOpenSuppressedRef.current = false;
    const started = startInspect();
    if (!started) {
      setLockedInspectMessage('Element inspector is unavailable for this preview.', 3600);
      return;
    }
    clearInspectMessageLock();
    setInspectStatusMessage(prefersCoarsePointer
      ? 'Drag on the preview to target an element and lift your finger to capture. Press Esc to cancel.'
      : 'Select an element in the preview. Press Esc to cancel.');
  }, [
    clearInspectMessageLock,
    inspectState.active,
    inspectState.supported,
    prefersCoarsePointer,
    previewUrl,
    setLockedInspectMessage,
    startInspect,
    stopInspect,
  ]);

  useEffect(() => {
    if (inspectState.active && !isInspectorDialogOpen) {
      inspectorAutoOpenSuppressedRef.current = false;
      setIsInspectorDialogOpen(true);
    }
  }, [inspectState.active, isInspectorDialogOpen]);

  useEffect(() => {
    if (inspectStatusMessage && !isInspectorDialogOpen && !inspectorAutoOpenSuppressedRef.current) {
      setIsInspectorDialogOpen(true);
    }
  }, [inspectStatusMessage, isInspectorDialogOpen]);

  useEffect(() => {
    if (isInspectorDialogOpen) {
      inspectorAutoOpenSuppressedRef.current = false;
    }
  }, [isInspectorDialogOpen]);

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
      setInspectStatusMessage(prefersCoarsePointer
        ? 'Drag on the preview to target an element and lift your finger to capture. Press Esc to cancel.'
        : 'Select an element in the preview. Press Esc to cancel.');
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
    prefersCoarsePointer,
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
    inspectActiveChipLabel,
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
    inspectAncestors,
    inspectAncestorIndex: effectiveAncestorIndex,
    inspectAncestorCanStepParent,
    inspectAncestorCanStepChild,
    handleInspectorAncestorStep,
    handleInspectorAncestorSelect,
    inspectorScreenshot,
    isInspectorScreenshotCapturing,
    inspectorScreenshotExpanded,
    toggleInspectorScreenshotExpanded,
    inspectorCaptureNote,
    handleInspectorCaptureNoteChange,
    handleAddInspectorCaptureToReport,
    inspectorReportStatus,
    handleInspectorViewReport,
    handleInspectorInspectAnother,
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
