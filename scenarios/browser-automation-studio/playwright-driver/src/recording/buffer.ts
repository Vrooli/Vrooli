import type { RecordedAction } from './types';

// In-memory action buffers keyed by session ID for record-mode endpoints.
const actionBuffers = new Map<string, RecordedAction[]>();

export function initRecordingBuffer(sessionId: string): void {
  actionBuffers.set(sessionId, []);
}

export function bufferRecordedAction(sessionId: string, action: RecordedAction): void {
  const buffer = actionBuffers.get(sessionId);
  if (buffer) {
    buffer.push(action);
    return;
  }

  actionBuffers.set(sessionId, [action]);
}

export function getRecordedActions(sessionId: string): RecordedAction[] {
  return actionBuffers.get(sessionId) || [];
}

export function getRecordedActionCount(sessionId: string): number {
  return getRecordedActions(sessionId).length;
}

export function clearRecordedActions(sessionId: string): void {
  actionBuffers.set(sessionId, []);
}

export function removeRecordedActions(sessionId: string): void {
  actionBuffers.delete(sessionId);
}
