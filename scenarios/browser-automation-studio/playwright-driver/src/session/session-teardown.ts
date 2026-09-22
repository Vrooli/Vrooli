import { access, mkdir, rename, stat } from 'node:fs/promises';
import { constants } from 'node:fs';
import path from 'node:path';
import type { Video } from 'rebrowser-playwright';
import type { SessionState } from '../types';
import { assertRecordingAcknowledged, removeRecordingBuffer } from '../recording';
import { metrics } from '../utils';
import { stopFrameStreaming } from '../frame-streaming';

/** Retain progress on the session until every required teardown stage succeeds. */
export async function teardownSessionResources(session: SessionState): Promise<string[]> {
  const progress = session.closeProgress ??= { completed: new Set(), videoPaths: new Map() };
  const once = async (operation: string, run: () => Promise<unknown>): Promise<void> => {
    if (progress.completed.has(operation)) return;
    try {
      await run();
      progress.completed.add(operation);
    } catch (cause) {
      metrics.cleanupFailures.inc({ operation: operation.split(':')[0]! });
      throw new Error(`${operation}: ${cause instanceof Error ? cause.message : String(cause)}`, { cause });
    }
  };

  // Flush before disposing the browser. A failure leaves its recovery owner and
  // recording buffer intact; successful stages are not repeated on retry.
  await once('audio_playback_stop', async () => {
    await session.audioPlaybackStop?.();
    session.audioPlaybackStop = undefined;
  });
  await once('recording_stop', async () => {
    if (session.pipelineManager?.isRecording()) await session.pipelineManager.stopRecording();
  });
  assertRecordingAcknowledged(session.id);
  await once('page_callbacks_stop', async () => {
    session.pageLifecycleCleanup?.();
    session.pageLifecycleCleanup = undefined;
  });
  await once('frame_stream_stop', async () => stopFrameStreaming(session.id));
  await once('service_worker_disable', async () => session.serviceWorkerController?.disable());
  await once('accessibility_capture', async () => session.accessibilitySnapshotter?.capture(session.page));
  await once('performance_trace_stop', async () => session.perfTracer?.stop(session.page));
  if (session.tracing && session.tracePath) {
    await once('tracing_stop', async () => session.context.tracing.stop({ path: session.tracePath }));
  }
  if (session.tracePath) await once('trace_file', async () => readableArtifact(session.tracePath!));

  if (session.externalTarget) {
    // Detach BAS's CDP connection. The target owner controls its pages/process.
    await once('cdp_detach', async () => session.browser.close());
  } else {
    for (const [index, page] of session.pages.entries()) {
      await once(`page_close:${index}`, async () => { if (!page.isClosed()) await page.close(); });
    }
    // video.path() yields a destination, not proof that the encoder finished.
    // Context close flushes capture before any file is moved or published.
    await once('context_close', async () => session.context.close());
    for (const [index, page] of session.pages.entries()) {
      await once(`video_file:${index}`, async () => {
        const video = page.video();
        if (video) progress.videoPaths.set(index, await moveVideo(video, session, index));
      });
    }
  }
  if (session.harPath) await once('har_file', async () => readableArtifact(session.harPath!));
  const videoPaths = [...progress.videoPaths.entries()].sort(([a], [b]) => a - b).map(([, file]) => file);
  // A prior attempt may have validated a file before another stage failed.
  // Check every published reference again at the successful close boundary.
  const artifacts = [...videoPaths, session.tracePath, session.harPath].filter((file): file is string => !!file);
  await Promise.all(artifacts.map(readableArtifact));
  removeRecordingBuffer(session.id);
  return videoPaths;
}

async function readableArtifact(file: string): Promise<void> {
  if (!(await stat(file)).isFile()) throw new Error('artifact is not a regular file');
  await access(file, constants.R_OK);
}

export async function moveVideo(video: Video, session: SessionState, index: number): Promise<string> {
  const source = await video.path();
  await readableArtifact(source);
  const target = path.join(session.videoDir || path.dirname(source), `execution-${session.spec.execution_id}-page-${index + 1}${path.extname(source) || '.webm'}`);
  if (target === source) return source;
  try {
    await mkdir(path.dirname(target), { recursive: true });
    await rename(source, target);
  } catch {
    // Relocation is optional; fallback is valid only while the original bytes
    // remain readable. Missing evidence must propagate to the close caller.
    await readableArtifact(source);
    return source;
  }
  await readableArtifact(target);
  return target;
}
