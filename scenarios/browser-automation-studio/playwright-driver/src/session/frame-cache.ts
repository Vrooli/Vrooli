export interface CapturedFrame {
  key: string;
  hash: string;
  base64DataUri: string;
  width: number;
  height: number;
  capturedAt: number;
}

export interface FrameCacheSlot {
  frame?: CapturedFrame;
  pending?: { key: string; result: Promise<CapturedFrame | null> };
}

const frameCache = new Map<string, FrameCacheSlot>();

export function getFrameCacheSlot(sessionId: string): FrameCacheSlot {
  const slot = frameCache.get(sessionId) ?? {};
  frameCache.set(sessionId, slot);
  return slot;
}

export function ownsFrameCacheSlot(sessionId: string, slot: FrameCacheSlot): boolean {
  return frameCache.get(sessionId) === slot;
}

/** Clear one session's cached frame and retire any pending capture. */
export function clearFrameCache(sessionId: string): void {
  frameCache.delete(sessionId);
}

/** Clear every cached frame during driver shutdown. */
export function clearAllFrameCaches(): void {
  frameCache.clear();
}
