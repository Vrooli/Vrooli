/** Recording and execution previews share one bounded WebSocket decoder. */
import { useContext, useEffect, useRef, useState } from 'react';
import { getConfig } from '@/config';
import { WebSocketContext } from '@/contexts/WebSocketContext';
import { useFrameStats, type FrameStats } from '../hooks/useFrameStats';
import { useSessionStore } from '../stores';

interface FrameDimensions { width: number; height: number; capturedAt: string }
interface FrameIdentity { session_id: string; page_id: string; captured_at: string }
export interface PageMetadata { title: string; url: string }
export interface ExecutionFrame { data: string; mediaType: string; width: number; height: number; capturedAt: string }
export interface StreamConnectionStatus {
  isConnected: boolean;
  isWebSocket: boolean;
  lastFrameTime?: string;
}
export interface UseFrameStreamOptions {
  sessionId: string | null;
  /** Execution previews use the shared hub socket and decoder. */
  executionId?: string | null;
  enabled?: boolean;
  onExecutionFrame?: (frame: ExecutionFrame) => void;
  /** Null is an empty workspace; omission follows the active browser page. */
  pageId?: string | null;
  refreshToken?: number;
  onStreamError?: (message: string) => void;
  onStatsUpdate?: (stats: FrameStats) => void;
  onPageMetadataChange?: (metadata: PageMetadata) => void;
  onConnectionStatusChange?: (status: StreamConnectionStatus) => void;
  enableTimestampState?: boolean;
}
interface ViewState {
  hasFrame: boolean;
  displayDimensions: {width: number; height: number} | null;
  displayedTimestamp: string | null;
  error: string | null;
  isFetching: boolean;
  isWsFrameActive: boolean;
  isPageSwitching: boolean;
  frameCount: number;
}
export interface UseFrameStreamResult extends ViewState {
  canvasRef: React.RefObject<HTMLCanvasElement>;
  frameDimensionsRef: React.RefObject<FrameDimensions | null>;
  frameStats: FrameStats;
  frameUrl: string | null;
}
const EMPTY_VIEW: ViewState = {
  hasFrame: false, displayDimensions: null, displayedTimestamp: null, error: null,
  isFetching: false, isWsFrameActive: false, isPageSwitching: false, frameCount: 0,
};
// Keep a bounded visual cadence when decoding is slower than admission. A
// frame that is only a few admissions behind can still be useful; a much older
// decode is discarded so a burst cannot paint stale pixels after a newer frame
// has already been admitted.
const MAX_ADMISSION_LAG = 3;

interface FrameJob {
  id: number;
  blob: Blob;
  timestamp: string;
  socket: boolean;
  metadata?: PageMetadata;
  execution?: ExecutionFrame;
  owns: () => boolean;
  deliver: (frame: FrameJob, bitmap: ImageBitmap) => void;
  fail: (message: string) => void;
}
interface Decoder { active: boolean; pending: FrameJob | null }

// This decoder survives effect replacement, so a slow old-session decode cannot
// multiply active work as the user switches tabs. Only the newest input waits.
async function decodeLatest(decoder: Decoder): Promise<void> {
  if (decoder.active) return;
  decoder.active = true;
  try {
    while (decoder.pending) {
      const frame = decoder.pending;
      decoder.pending = null;
      if (!frame.owns()) continue;
      try {
        const bitmap = await createImageBitmap(frame.blob);
        if (frame.owns()) frame.deliver(frame, bitmap);
        else bitmap.close();
      } catch {
        if (frame.owns()) frame.fail('Failed to decode live frame');
      }
    }
  } finally {
    decoder.active = false;
  }
}

function parseIdentity(value: unknown): FrameIdentity {
  if (!value || typeof value !== 'object') throw new Error('Invalid frame identity');
  const identity = value as Partial<FrameIdentity>;
  if (typeof identity.session_id !== 'string' || !identity.session_id ||
      typeof identity.page_id !== 'string' || !identity.page_id ||
      typeof identity.captured_at !== 'string' || !Number.isFinite(Date.parse(identity.captured_at))) {
    throw new Error('Invalid frame identity');
  }
  return identity as FrameIdentity;
}

function executionFrameBlob(data: string, mediaType: string): Blob {
  const bytes = Uint8Array.from(atob(data), character => character.charCodeAt(0));
  return new Blob([bytes], {type: mediaType || 'image/jpeg'});
}

// The API publishes canonical identity; producer lease credentials stay server-side.
function binaryFrame(data: ArrayBuffer): FrameIdentity & {blob: Blob; timestamp: string; metadata?: PageMetadata} {
  if (data.byteLength < 4) throw new Error('Invalid binary frame');
  const length = new DataView(data).getUint32(0);
  if (!length || length > 16 * 1024 || length > data.byteLength - 6) throw new Error('Invalid binary frame');
  const header: unknown = JSON.parse(new TextDecoder().decode(new Uint8Array(data, 4, length)));
  const identity = parseIdentity(header);
  if ((header as {version?: unknown}).version !== 1) throw new Error('Invalid frame version');
  const jpeg = new Uint8Array(data, 4 + length);
  if (jpeg[0] !== 255 || jpeg[1] !== 216) throw new Error('Invalid JPEG frame');
  const metadata = header as {page_title?: unknown; page_url?: unknown};
  return {...identity, blob: new Blob([jpeg], {type:'image/jpeg'}), timestamp: identity.captured_at,
    metadata: typeof metadata.page_title === 'string' && typeof metadata.page_url === 'string'
      ? {title:metadata.page_title,url:metadata.page_url} : undefined};
}

export function useFrameStream(options: UseFrameStreamOptions): UseFrameStreamResult {
  const {sessionId: suppliedSessionId, executionId = null, enabled = true, pageId, refreshToken} = options;
  const webSocket = useContext(WebSocketContext);
  const storedSessionId = useSessionStore(state => state.sessionId);
  const validated = useSessionStore(state => state.isValidated);
  const sessionId = executionId ? null : validated ? storedSessionId : suppliedSessionId;
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const frameDimensionsRef = useRef<FrameDimensions | null>(null);
  const decoder = useRef<Decoder>({active:false,pending:null});
  const callbacks = useRef(options);
  callbacks.current = options;
  const executionMessageHandler = useRef<((message: unknown) => void) | null>(null);
  const previousPage = useRef(pageId);
  const [view, setView] = useState<ViewState>(EMPTY_VIEW);
  const [frameUrl, setFrameUrl] = useState<string | null>(null);
  const {stats: frameStats, recordFrame, reset: resetStats} = useFrameStats();

  useEffect(() => {callbacks.current.onStatsUpdate?.(frameStats);}, [frameStats]);
  useEffect(() => webSocket?.subscribeToMessages(message => executionMessageHandler.current?.(message)),
    [webSocket?.subscribeToMessages]);
  useEffect(() => {
    callbacks.current.onConnectionStatusChange?.({isConnected:view.hasFrame,
      isWebSocket:view.isWsFrameActive,lastFrameTime:view.displayedTimestamp ?? undefined});
  }, [view.hasFrame, view.isWsFrameActive, view.displayedTimestamp]);

  useEffect(() => {
    const currentDecoder = decoder.current;
    let disposed = false;
    let socket: WebSocket | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let sequence = 0;
    let newestAdmission = 0;
    let newestAdmissionTimestamp = '';
    let lastPainted = 0;
    let reconnectAttempts = 0;
    let lastMetadata: PageMetadata | undefined;
    let lastTimestampUpdate = -Infinity;
    let painted = false;
    let currentView = {...EMPTY_VIEW, isPageSwitching: Boolean(previousPage.current && pageId && previousPage.current !== pageId)};
    previousPage.current = pageId;
    setView(currentView);
    setFrameUrl(null);
    frameDimensionsRef.current = null;
    useSessionStore.getState().setFrameDimensions(null);
    useSessionStore.getState().setDisplayDimensions(null);
    resetStats();
    const canvas = canvasRef.current;
    canvas?.getContext('2d')?.clearRect(0,0,canvas.width,canvas.height);
    const owns = () => !disposed;
    const update = (change: Partial<ViewState>) => {
      if (disposed || Object.entries(change).every(([key,value]) => currentView[key as keyof ViewState] === value)) return;
      currentView = {...currentView,...change};
      setView(currentView);
    };
    const fail = (message: string) => {
      if (disposed) return;
      update({error:message});
      callbacks.current.onStreamError?.(message);
    };
    const draw = (frame: FrameJob, bitmap: ImageBitmap) => {
      try {
        if (disposed || frame.id <= lastPainted) return;
        const target = canvasRef.current;
        const context = target?.getContext('2d', {alpha:false});
        if ((!target || !context) && !frame.execution) return;
        // Draw as soon as decoding completes. Deferring through another
        // animation frame adds a full display interval to input feedback.
        if (target && context) {
          if (target.width !== bitmap.width) target.width = bitmap.width;
          if (target.height !== bitmap.height) target.height = bitmap.height;
          context.drawImage(bitmap,0,0);
        }
        const dimensions = {width:bitmap.width,height:bitmap.height};
        const previous = frameDimensionsRef.current;
        frameDimensionsRef.current = {...dimensions,capturedAt:frame.timestamp};
        const change: Partial<ViewState> = {hasFrame:true,isFetching:false,isPageSwitching:false,error:null,isWsFrameActive:frame.socket};
        if (frame.execution) change.frameCount = currentView.frameCount + 1;
        if (!previous || previous.width !== bitmap.width || previous.height !== bitmap.height) {
          change.displayDimensions = dimensions;
          useSessionStore.getState().setFrameDimensions(dimensions);
          useSessionStore.getState().setDisplayDimensions(dimensions);
        }
        if (callbacks.current.enableTimestampState !== false && performance.now() - lastTimestampUpdate >= 1000) {
          change.displayedTimestamp = frame.timestamp;
          lastTimestampUpdate = performance.now();
        }
        lastPainted = frame.id;
        painted = true;
        update(change);
        if (frame.execution) callbacks.current.onExecutionFrame?.(frame.execution);
        if (frame.execution) setFrameUrl(`data:${frame.execution.mediaType};base64,${frame.execution.data}`);
        recordFrame(frame.blob.size);
        if (frame.metadata && (frame.metadata.url !== lastMetadata?.url || frame.metadata.title !== lastMetadata?.title)) {
          lastMetadata = frame.metadata;
          callbacks.current.onPageMetadataChange?.(frame.metadata);
        }
      } catch {
        fail('Failed to render live frame');
      } finally {
        bitmap.close();
      }
    };
    const deliver = (frame: FrameJob, bitmap: ImageBitmap) => {
      // A newer frame may arrive while this decode is in flight. Do not put
      // stale pixels into the paint queue; close the bitmap at the ownership
      // boundary and let the latest pending frame advance instead.
      const frameTime = Date.parse(frame.timestamp);
      const newestTime = Date.parse(newestAdmissionTimestamp);
      const adjacentTimestampStale = frame.id < newestAdmission &&
        Number.isFinite(frameTime) && Number.isFinite(newestTime) &&
        newestTime > frameTime && newestTime - frameTime <= 1;
      if (newestAdmission - frame.id > MAX_ADMISSION_LAG || adjacentTimestampStale) {
        bitmap.close();
        return;
      }
      draw(frame,bitmap);
    };
    const enqueue = (frame: Omit<FrameJob,'owns'|'deliver'|'fail'>) => {
      if (disposed || document.hidden || frame.id < newestAdmission) return;
      newestAdmission = frame.id;
      newestAdmissionTimestamp = frame.timestamp;
      currentDecoder.pending = {...frame,owns,deliver,fail};
      void decodeLatest(currentDecoder);
    };
    const onExecutionMessage = (value: unknown) => {
      if (!executionId || !enabled || !value || typeof value !== 'object') return;
      const message = value as {
        type?: unknown; execution_id?: unknown; data?: unknown; media_type?: unknown;
        width?: unknown; height?: unknown; captured_at?: unknown;
      };
      if (message.execution_id !== executionId) return;
      if (message.type === 'execution_frame_subscribed') {
        update({isWsFrameActive:true});
        return;
      }
      if (message.type !== 'execution_frame') return;
      if (typeof message.data !== 'string' || typeof message.captured_at !== 'string' ||
          !Number.isFinite(Date.parse(message.captured_at)) || typeof message.width !== 'number' ||
          message.width <= 0 || typeof message.height !== 'number' || message.height <= 0) {
        fail('Invalid execution frame');
        return;
      }
      try {
        const mediaType = typeof message.media_type === 'string' ? message.media_type : 'image/jpeg';
        enqueue({id:++sequence,blob:executionFrameBlob(message.data,mediaType),
          timestamp:message.captured_at,socket:true,
          execution:{data:message.data,mediaType,width:message.width,height:message.height,capturedAt:message.captured_at}});
      } catch {
        fail('Invalid execution frame');
      }
    };
    const matchesSource = (frame: FrameIdentity) => frame.session_id === sessionId &&
      (pageId === undefined || frame.page_id === pageId);
    const start = async () => {
      const config = await getConfig();
      if (disposed || !sessionId) return;
      const connect = () => {
        if (disposed) return;
        const connection = new WebSocket(config.WS_URL);
        socket = connection;
        connection.binaryType = 'arraybuffer';
        connection.onopen = () => {
          if (disposed || socket !== connection) return;
          reconnectAttempts = 0;
          update({isFetching:!painted});
          connection.send(JSON.stringify({type:'subscribe_recording',session_id:sessionId}));
        };
        connection.onmessage = event => {
          if (disposed || socket !== connection || !(event.data instanceof ArrayBuffer)) return;
          try {
            const frame = binaryFrame(event.data);
            if (matchesSource(frame)) enqueue({...frame,id:++sequence,socket:true});
          }
          catch {fail('Invalid binary frame');}
        };
        connection.onerror = () => { /* onclose owns retry. */ };
        connection.onclose = () => {
          if (disposed || socket !== connection) return;
          socket = null;
          update({isWsFrameActive:false,isFetching:false});
          if (reconnectAttempts < 10) {
            const delay = Math.min(250 * 2 ** reconnectAttempts++,15000);
            reconnectTimer = setTimeout(connect,delay);
          }
        };
      };
      connect();
    };
    let subscribedExecution = false;
    if (executionId) {
      executionMessageHandler.current = onExecutionMessage;
      if (enabled && webSocket?.isConnected) {
        subscribedExecution = webSocket.send({type:'subscribe_execution_frames',execution_id:executionId});
        if (!subscribedExecution) fail('Failed to subscribe to execution frames');
      }
    } else if (sessionId && pageId !== null && enabled) {
      void start().catch(error => fail(error instanceof Error ? error.message : 'Failed to connect live viewer'));
    }
    return () => {
      disposed = true;
      if (executionMessageHandler.current === onExecutionMessage) executionMessageHandler.current = null;
      if (subscribedExecution) webSocket?.send({type:'unsubscribe_execution_frames'});
      socket?.close();
      if (reconnectTimer !== null) clearTimeout(reconnectTimer);
      if (currentDecoder.pending?.owns === owns) currentDecoder.pending = null;
    };
  }, [sessionId,executionId,enabled,webSocket?.isConnected,webSocket?.send,pageId,refreshToken,recordFrame,resetStats]);

  return {...view,canvasRef,frameDimensionsRef,frameStats,frameUrl};
}
