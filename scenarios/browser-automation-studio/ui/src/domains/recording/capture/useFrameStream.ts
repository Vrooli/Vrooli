/** A session-owned API stream and HTTP fallback share one bounded decoder. */
import { useEffect, useRef, useState } from 'react';
import { getConfig } from '@/config';
import { useFrameStats, type FrameStats } from '../hooks/useFrameStats';
import { useSessionStore } from '../stores';

interface FrameDimensions { width: number; height: number; capturedAt: string }
interface FrameIdentity { session_id: string; page_id: string; captured_at: string }
interface FramePayload extends FrameIdentity {
  image: string;
  width: number;
  height: number;
  page_title?: string;
  page_url?: string;
}
export interface PageMetadata { title: string; url: string }
export interface StreamConnectionStatus {
  isConnected: boolean;
  isWebSocket: boolean;
  lastFrameTime?: string;
}
export interface UseFrameStreamOptions {
  sessionId: string | null;
  /** Null is an empty workspace; omission follows the active browser page. */
  pageId?: string | null;
  quality?: number;
  fps?: number;
  useWebSocketFrames?: boolean;
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
}
export interface UseFrameStreamResult extends ViewState {
  canvasRef: React.RefObject<HTMLCanvasElement>;
  frameDimensionsRef: React.RefObject<FrameDimensions | null>;
  frameStats: FrameStats;
}
const EMPTY_VIEW: ViewState = {
  hasFrame: false, displayDimensions: null, displayedTimestamp: null, error: null,
  isFetching: false, isWsFrameActive: false, isPageSwitching: false,
};
const STREAM_STALE_MS = 1000;
// Keep one missed-RAF frame pair without allowing delayed viewers to grow memory use.
const MAX_PENDING_PAINT_FRAMES = 2;
const MAX_PENDING_PAINT_BYTES = 16 * 1024 * 1024;

interface FrameJob {
  id: number;
  blob: Blob;
  timestamp: string;
  socket: boolean;
  etag?: string | null;
  metadata?: PageMetadata;
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

function parseFrame(value: unknown): FramePayload {
  parseIdentity(value);
  const frame = value as Partial<FramePayload>;
  if (typeof frame.image !== 'string' || !frame.image || typeof frame.width !== 'number' ||
      frame.width <= 0 || typeof frame.height !== 'number' || frame.height <= 0) throw new Error('Invalid frame payload');
  return frame as FramePayload;
}

function frameBlob(image: string): Blob {
  const encoded = image.includes(',') ? image.slice(image.indexOf(',') + 1) : image;
  const bytes = Uint8Array.from(atob(encoded), character => character.charCodeAt(0));
  return new Blob([bytes], {type: 'image/jpeg'});
}

// The API publishes canonical identity; producer lease credentials stay server-side.
function binaryFrame(data: ArrayBuffer): FrameIdentity & {blob: Blob; timestamp: string} {
  if (data.byteLength < 4) throw new Error('Invalid binary frame');
  const length = new DataView(data).getUint32(0);
  if (!length || length > 16 * 1024 || length > data.byteLength - 6) throw new Error('Invalid binary frame');
  const header: unknown = JSON.parse(new TextDecoder().decode(new Uint8Array(data, 4, length)));
  const identity = parseIdentity(header);
  if ((header as {version?: unknown}).version !== 1) throw new Error('Invalid frame version');
  const jpeg = new Uint8Array(data, 4 + length);
  if (jpeg[0] !== 255 || jpeg[1] !== 216) throw new Error('Invalid JPEG frame');
  return {...identity, blob: new Blob([jpeg], {type:'image/jpeg'}), timestamp: identity.captured_at};
}

export function useFrameStream(options: UseFrameStreamOptions): UseFrameStreamResult {
  const {sessionId: suppliedSessionId, pageId, quality = 65, fps = 30,
    useWebSocketFrames = true, refreshToken} = options;
  const storedSessionId = useSessionStore(state => state.sessionId);
  const validated = useSessionStore(state => state.isValidated);
  const sessionId = validated ? storedSessionId : suppliedSessionId;
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const frameDimensionsRef = useRef<FrameDimensions | null>(null);
  const decoder = useRef<Decoder>({active:false,pending:null});
  const callbacks = useRef(options);
  callbacks.current = options;
  const previousPage = useRef(pageId);
  const [view, setView] = useState<ViewState>(EMPTY_VIEW);
  const {stats: frameStats, recordFrame, reset: resetStats} = useFrameStats();

  useEffect(() => {callbacks.current.onStatsUpdate?.(frameStats);}, [frameStats]);
  useEffect(() => {
    callbacks.current.onConnectionStatusChange?.({isConnected:view.hasFrame,
      isWebSocket:view.isWsFrameActive,lastFrameTime:view.displayedTimestamp ?? undefined});
  }, [view.hasFrame, view.isWsFrameActive, view.displayedTimestamp]);

  useEffect(() => {
    const currentDecoder = decoder.current;
    let disposed = false;
    let socket: WebSocket | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let pollTimer: ReturnType<typeof setTimeout> | null = null;
    let request: AbortController | null = null;
    let raf: number | null = null;
    let pendingPaint: Array<{frame: FrameJob; bitmap: ImageBitmap; byteSize: number}> = [];
    let pendingPaintBytes = 0;
    let sequence = 0;
    let newestAdmission = 0;
    let lastPainted = 0;
    let lastSocketPaint = -Infinity;
    let reconnectAttempts = 0;
    let etag: string | null = null;
    let lastMetadata: PageMetadata | undefined;
    let lastTimestampUpdate = -Infinity;
    let painted = false;
    let currentView = {...EMPTY_VIEW, isPageSwitching: Boolean(previousPage.current && pageId && previousPage.current !== pageId)};
    previousPage.current = pageId;
    setView(currentView);
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
    const draw = () => {
      raf = null;
      const ready = pendingPaint.shift();
      if (!ready) return;
      pendingPaintBytes -= ready.byteSize;
      const {frame,bitmap} = ready;
      try {
        if (disposed || frame.id <= lastPainted) return;
        const target = canvasRef.current;
        const context = target?.getContext('2d', {alpha:false});
        if (!target || !context) return;
        // Resize and draw in one synchronous animation callback, before paint.
        if (target.width !== bitmap.width) target.width = bitmap.width;
        if (target.height !== bitmap.height) target.height = bitmap.height;
        context.drawImage(bitmap,0,0);
        const dimensions = {width:bitmap.width,height:bitmap.height};
        const previous = frameDimensionsRef.current;
        frameDimensionsRef.current = {...dimensions,capturedAt:frame.timestamp};
        const change: Partial<ViewState> = {hasFrame:true,isFetching:false,isPageSwitching:false,error:null,isWsFrameActive:frame.socket};
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
        if (frame.socket) lastSocketPaint = performance.now();
        else if (frame.etag) etag = frame.etag;
        update(change);
        recordFrame(frame.blob.size);
        if (frame.metadata && (frame.metadata.url !== lastMetadata?.url || frame.metadata.title !== lastMetadata?.title)) {
          lastMetadata = frame.metadata;
          callbacks.current.onPageMetadataChange?.(frame.metadata);
        }
      } catch {
        fail('Failed to render live frame');
      } finally {
        bitmap.close();
        if (!disposed && pendingPaint.length > 0 && raf === null) raf = requestAnimationFrame(draw);
      }
    };
    const deliver = (frame: FrameJob, bitmap: ImageBitmap) => {
      const byteSize = bitmap.width * bitmap.height * 4;
      // A single oversize image remains displayable; never retain another beside it.
      while (pendingPaint.length > 0 &&
        (pendingPaint.length >= MAX_PENDING_PAINT_FRAMES || pendingPaintBytes + byteSize > MAX_PENDING_PAINT_BYTES)) {
        const discarded = pendingPaint.shift();
        if (!discarded) break;
        pendingPaintBytes -= discarded.byteSize;
        discarded.bitmap.close();
      }
      pendingPaint.push({frame,bitmap,byteSize});
      pendingPaintBytes += byteSize;
      if (raf === null) raf = requestAnimationFrame(draw);
    };
    const enqueue = (frame: Omit<FrameJob,'owns'|'deliver'|'fail'>) => {
      if (disposed || document.hidden || frame.id < newestAdmission) return;
      newestAdmission = frame.id;
      currentDecoder.pending = {...frame,owns,deliver,fail};
      void decodeLatest(currentDecoder);
    };
    const matchesSource = (frame: FrameIdentity) => frame.session_id === sessionId &&
      (pageId === undefined || frame.page_id === pageId);
    const pollInterval = Math.max(300,Math.floor(1000 / Math.max(1,fps)));
    const start = async () => {
      const config = await getConfig();
      if (disposed || !sessionId) return;
      const poll = async () => {
        if (disposed) return;
        if (performance.now() - lastSocketPaint >= STREAM_STALE_MS) {
          update({isWsFrameActive:false});
          if (!document.hidden && !request) {
            const id = ++sequence;
            request = new AbortController();
            const timeout = setTimeout(() => request?.abort(),10000);
            update({isFetching:!painted});
            try {
              const query = new URLSearchParams({quality:String(quality)});
              if (pageId) query.set('page_id',pageId);
              const response = await fetch(`${config.API_URL}/recordings/live/${sessionId}/frame?${query}`, {
                signal:request.signal, headers:etag ? {'If-None-Match':etag} : {},
              });
              if (disposed || id < newestAdmission || response.status === 304) return;
              if (!response.ok) throw new Error(`Frame fetch failed (${response.status})`);
              const frame = parseFrame(await response.json());
              if (disposed || id < newestAdmission || !matchesSource(frame)) return;
              enqueue({id,blob:frameBlob(frame.image),timestamp:frame.captured_at,socket:false,etag:response.headers.get('ETag'),
                metadata: {title:frame.page_title ?? '',url:frame.page_url ?? ''}});
            } catch (error) {
              if (!disposed && id >= newestAdmission) fail(error instanceof Error ? error.message : 'Failed to fetch live frame');
            } finally {
              clearTimeout(timeout);
              request = null;
              update({isFetching:false});
              if (!disposed) pollTimer = setTimeout(() => {void poll();},pollInterval);
            }
            return;
          }
        }
        pollTimer = setTimeout(() => {void poll();},pollInterval);
      };
      const connect = () => {
        if (disposed || !useWebSocketFrames) return;
        const connection = new WebSocket(config.WS_URL);
        socket = connection;
        connection.binaryType = 'arraybuffer';
        connection.onopen = () => {
          if (disposed || socket !== connection) return;
          reconnectAttempts = 0;
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
        connection.onerror = () => { /* onclose owns retry and fallback. */ };
        connection.onclose = () => {
          if (disposed || socket !== connection) return;
          socket = null;
          lastSocketPaint = -Infinity;
          update({isWsFrameActive:false});
          if (reconnectAttempts < 10) {
            const delay = Math.min(250 * 2 ** reconnectAttempts++,15000);
            reconnectTimer = setTimeout(connect,delay);
          }
        };
      };
      void poll();
      connect();
    };
    if (sessionId && pageId !== null) void start().catch(error => fail(error instanceof Error ? error.message : 'Failed to connect live viewer'));
    return () => {
      disposed = true;
      socket?.close();
      request?.abort();
      if (reconnectTimer !== null) clearTimeout(reconnectTimer);
      if (pollTimer !== null) clearTimeout(pollTimer);
      if (raf !== null) cancelAnimationFrame(raf);
      for (const pending of pendingPaint) pending.bitmap.close();
      pendingPaint = [];
      pendingPaintBytes = 0;
      if (currentDecoder.pending?.owns === owns) currentDecoder.pending = null;
    };
  }, [sessionId,pageId,quality,fps,useWebSocketFrames,refreshToken,recordFrame,resetStats]);

  return {...view,canvasRef,frameDimensionsRef,frameStats};
}
