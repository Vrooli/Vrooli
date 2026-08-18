import { mkdir, rename } from 'node:fs/promises';
import path from 'node:path';
import type { Video } from 'rebrowser-playwright';
import type { SessionState } from '../types';
import { removeRecordingBuffer } from '../recording';
import { logger, metrics, scopedLog, LogContext } from '../utils';

const message = (error: unknown): string => error instanceof Error ? error.message : String(error);

export async function teardownSessionResources(session: SessionState): Promise<string[]> {
  const videoPaths: string[] = [];
  const startedAt = Date.now();
  const warn = (event: string, error: unknown, operation?: string, context: typeof LogContext[keyof typeof LogContext] = LogContext.CLEANUP): void => {
    logger.warn(scopedLog(context, event), { sessionId: session.id, error: message(error) });
    if (operation) metrics.cleanupFailures.inc({ operation });
  };
  try {
    await session.audioPlaybackStop?.().catch((error: unknown) => warn('audio playback stop failed', error, 'audio_playback_stop'));
    session.audioPlaybackStop = undefined;
    if (session.pipelineManager?.isRecording()) await session.pipelineManager.stopRecording().catch((error: unknown) => warn('recording stop failed', error, 'recording_stop'));
    removeRecordingBuffer(session.id);
    await session.serviceWorkerController?.disable().catch((error: unknown) => warn('SW controller disable failed', error));
    await session.accessibilitySnapshotter?.capture(session.page).catch((error: unknown) => warn('accessibility snapshot capture failed', error, undefined, LogContext.TELEMETRY));
    await session.perfTracer?.stop(session.page).catch((error: unknown) => warn('performance tracer stop failed', error, undefined, LogContext.TELEMETRY));
    if (session.tracing && session.tracePath) await session.context.tracing.stop({ path: session.tracePath }).catch((error: unknown) => warn('tracing stop failed', error, 'tracing_stop'));
    if (!session.externalTarget) {
      for (const [index, page] of session.pages.entries()) {
        const video = page.video();
        await page.close().catch((error: unknown) => warn('page close failed', error, 'page_close'));
        if (video) { const result = await moveVideo(video, session, index); if (result) videoPaths.push(result); }
      }
      await session.context.close().catch((error: unknown) => warn('context close failed', error, 'context_close'));
    } else {
      // The desktop target owns the renderer and process. Closing the CDP
      // connection detaches BAS; scenario-to-desktop performs target cleanup.
      await session.browser.close().catch((error: unknown) => warn('CDP detach failed', error, 'cdp_detach'));
    }
    metrics.sessionDuration.observe(Date.now() - startedAt);
  } catch (error) {
    logger.error(scopedLog(LogContext.SESSION, 'close failed'), { sessionId: session.id, error: message(error), hint: 'Session cleanup may be incomplete; browser resources may leak' });
  }
  return videoPaths;
}

export async function moveVideo(video: Video, session: SessionState, index: number): Promise<string | null> {
  let source = '';
  try { source = await video.path(); } catch { return null; }
  if (!source) return null;
  const target = path.join(session.videoDir || path.dirname(source), `execution-${session.spec.execution_id}-page-${index + 1}${path.extname(source) || '.webm'}`);
  if (target === source) return source;
  try { await mkdir(path.dirname(target), { recursive: true }); await rename(source, target); return target; }
  catch { return source; }
}
