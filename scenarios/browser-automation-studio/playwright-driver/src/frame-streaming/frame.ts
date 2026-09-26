import type { Page } from 'rebrowser-playwright';
import type { SessionState } from '../types/session';

/** Captured before asynchronous work; never relabel bytes with a newer owner. */
export interface FrameSource {
  session_id: string;
  execution_id: string;
  lease_id: string;
  page_id: string;
}

export type FrameSession = Pick<SessionState,
  'id' | 'ownerExecutionId' | 'leaseId' | 'leaseReleasedAt' | 'page' | 'pageToIdMap'>;

export function captureFrameSource(session: FrameSession, page: Page): FrameSource | null {
  const pageId = session.pageToIdMap.get(page);
  if (session.page !== page || page.isClosed() || session.leaseReleasedAt ||
      !session.id || !session.ownerExecutionId || !session.leaseId || !pageId) return null;
  return {session_id:session.id,execution_id:session.ownerExecutionId,lease_id:session.leaseId,page_id:pageId};
}

export function sameFrameSource(left: FrameSource | null, right: FrameSource): boolean {
  return left?.session_id === right.session_id && left.execution_id === right.execution_id &&
    left.lease_id === right.lease_id && left.page_id === right.page_id;
}

/** One source-bearing wire format, independent of optional performance mode. */
export function encodeFrame(
  source: FrameSource,
  jpeg: Buffer,
  capturedAt: number,
  timing?: Record<string, string | number | boolean>,
): Buffer {
  const header = Buffer.from(JSON.stringify({version:1,source,captured_at:new Date(capturedAt).toISOString(),timing}));
  const length = Buffer.alloc(4);
  length.writeUInt32BE(header.length,0);
  return Buffer.concat([length,header,jpeg]);
}
