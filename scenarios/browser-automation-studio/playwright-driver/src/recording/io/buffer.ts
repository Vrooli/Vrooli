import { timelineEntryToJson, type TimelineEntry } from '../../proto/recording';
import { MAX_RECORDING_BUFFER_SIZE } from '../../utils';
import { registerSessionCleanup } from '../../infra';

type BufferedEntry = {
  entry: TimelineEntry;
  visible: boolean;
  acknowledged: boolean;
  delivered: boolean;
  delivering: boolean;
  acknowledgeOnDelivery: boolean;
  deliver?: () => Promise<void>;
};
type RecordingBuffer = {
  entries: Map<string, BufferedEntry>;
  evictions: number;
  drain?: Promise<void>;
};

// The same bounded owner retains pending work, delivery progress and replay
// identity. Only acknowledged entries can be evicted or hidden by a clear.
const buffers = new Map<string, RecordingBuffer>();

function getBuffer(sessionId: string): RecordingBuffer {
  let buffer = buffers.get(sessionId);
  if (!buffer) {
    buffer = { entries: new Map(), evictions: 0 };
    buffers.set(sessionId, buffer);
  }
  return buffer;
}

export function assertRecordingAcknowledged(sessionId: string): void {
  const buffer = buffers.get(sessionId);
  if (buffer?.drain || [...buffer?.entries.values() ?? []].some((item) => !item.acknowledged)) {
    throw new Error('Recording delivery is still pending; retain the session and retry or acknowledge committed entries');
  }
}

export function initRecordingBuffer(sessionId: string): void {
  assertRecordingAcknowledged(sessionId);
  buffers.set(sessionId, { entries: new Map(), evictions: 0 });
}

export function bufferTimelineEntry(sessionId: string, entry: TimelineEntry): boolean {
  const buffer = getBuffer(sessionId);
  const existing = buffer.entries.get(entry.id);
  if (existing) {
    if (JSON.stringify(timelineEntryToJson(existing.entry)) !== JSON.stringify(timelineEntryToJson(entry))) {
      throw new Error('Recording observation identity conflicts with its original contents');
    }
    return false;
  }
  if (buffer.entries.size >= MAX_RECORDING_BUFFER_SIZE) {
    for (const [id, item] of buffer.entries) {
      if (item.acknowledged && !item.delivering) {
        buffer.entries.delete(id);
        buffer.evictions++;
        break;
      }
    }
    if (buffer.entries.size >= MAX_RECORDING_BUFFER_SIZE) throw new Error('Recording delivery capacity exhausted; unacknowledged entries are retained');
  }
  buffer.entries.set(entry.id, {
    entry, visible: true, acknowledged: false, delivered: false,
    delivering: false, acknowledgeOnDelivery: true,
  });
  return true;
}

export function getBufferedTimelineEntry(sessionId: string, entryId: string): TimelineEntry | undefined {
  return buffers.get(sessionId)?.entries.get(entryId)?.entry;
}

export async function deliverTimelineEntry(
  sessionId: string, entry: TimelineEntry, deliver: () => Promise<void>, acknowledgeOnDelivery = true,
): Promise<void> {
  const buffer = getBuffer(sessionId);
  const item = buffer.entries.get(entry.id);
  if (!item) throw new Error('Recording entry was not admitted');
  if (item.delivered) return;
  item.deliver ??= deliver;
  item.acknowledgeOnDelivery = acknowledgeOnDelivery;
  await flushRecordingDeliveries(sessionId);
  if (!item.delivered) throw new Error('Recording delivery did not complete');
}

export function flushRecordingDeliveries(sessionId: string): Promise<void> {
  const buffer = getBuffer(sessionId);
  if (buffer.drain) return buffer.drain;
  buffer.drain = (async () => {
    for (const item of buffer.entries.values()) {
      if (item.delivered || !item.deliver) continue;
      item.delivering = true;
      try {
        await item.deliver();
        item.delivered = true;
        if (item.acknowledgeOnDelivery) item.acknowledged = true;
      } finally { item.delivering = false; }
    }
  })().finally(() => { buffer.drain = undefined; });
  return buffer.drain;
}

/** The consumer calls this only after committing the named observations. */
export function acknowledgeTimelineEntries(sessionId: string, ids: string[], clear = false): void {
  const buffer = getBuffer(sessionId);
  for (const id of ids) {
    if (!buffer.entries.has(id)) throw new Error(`Unknown recording entry ${id}`);
  }
  for (const id of ids) {
    const item = buffer.entries.get(id)!;
    item.acknowledged = true;
    item.delivered = true;
    if (clear) item.visible = false;
  }
}

export function getTimelineEntries(sessionId: string): TimelineEntry[] {
  const entries: TimelineEntry[] = [];
  for (const item of buffers.get(sessionId)?.entries.values() ?? []) {
    if (item.visible) entries.push(item.entry);
  }
  return entries;
}

export function getTimelineEntryCount(sessionId: string): number {
  return getTimelineEntries(sessionId).length;
}

export function getBufferStats(sessionId: string): {
  entryCount: number; evictionCount: number; maxSize: number; pendingCount: number;
} {
  const buffer = buffers.get(sessionId);
  return {
    entryCount: getTimelineEntryCount(sessionId), evictionCount: buffer?.evictions ?? 0,
    maxSize: MAX_RECORDING_BUFFER_SIZE,
    pendingCount: [...buffer?.entries.values() ?? []].filter((item) => !item.acknowledged).length,
  };
}

export function removeRecordingBuffer(sessionId: string): void {
  assertRecordingAcknowledged(sessionId);
  buffers.delete(sessionId);
}

export function isEntryBuffered(sessionId: string, entryId: string): boolean {
  return buffers.get(sessionId)?.entries.has(entryId) ?? false;
}

registerSessionCleanup('recording-buffer', removeRecordingBuffer);
