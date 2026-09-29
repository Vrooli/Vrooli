import { access, mkdir, rename, stat } from 'node:fs/promises';
import { constants } from 'node:fs';
import path from 'node:path';
import type { Video } from 'rebrowser-playwright';
import type { SessionState } from '../types';
import { assertRecordingAcknowledged, removeRecordingBuffer } from '../recording';
import { metrics } from '../utils';
import { stopFrameStreaming } from '../frame-streaming';
import { settlePageInput } from './live-input';
import { clearFrameCache } from './frame-cache';

/** Retain progress on the session until every required teardown stage succeeds. */
export async function teardownSessionResources(session: SessionState): Promise<string[]> {
  const progress: NonNullable<SessionState['closeProgress']> = session.closeProgress ??= {
    completed: new Set<string>(), videoPaths: new Map<number, string>(),
  };
  const once = async (operation: string, run: () => Promise<unknown>): Promise<void> => {
    if (progress.completed.has(operation)) return;
    try {
      await run();
      progress.completed.add(operation);
    } catch (cause) {
      metrics.cleanupFailures.inc({ operation: operation.split(':')[0] ?? operation });
      throw new Error(`${operation}: ${cause instanceof Error ? cause.message : String(cause)}`, { cause });
    }
  };

  // Snapshot owned pages and video handles before an interrupted page closes.
  // Playwright can release a page's Video handle as part of close; retaining
  // the handle first keeps the artifact flush retryable and lossless.
  const pages = session.externalTarget ? [] : [...session.pages.entries()];
  if (!session.externalTarget && !pages.some(([, page]) => page === session.page)) {
    const nextIndex = pages.reduce((highest, [index]) => Math.max(highest, index), -1) + 1;
    pages.push([nextIndex, session.page]);
  }
  const videos = new Map<number, Video>();
  for (const [index, page] of pages) {
    const video = page.video();
    if (video) videos.set(index, video);
  }

  // An interrupted instruction must lose its browser effect before any
  // secondary cleanup can wait on it. In particular, pipeline readiness and
  // recording flushes may be independent of the active navigation; waiting for
  // them first kept an external effect live during driver shutdown. The normal
  // close path still performs the complete artifact-preserving teardown below.
  if (session.instructionInterrupted && !session.externalTarget) {
    const activePage = pages.find(([, page]) => page === session.page);
    if (activePage) {
      const [index, page] = activePage;
      await once(`page_close:${index}`, async () => {
        if (!page.isClosed()) await page.close();
      });
    }
  }

  // A background AI navigator can still be awaiting a model, callback, or
  // human intervention while close begins. Its page owner must settle before
  // teardown mutates or disposes browser resources.
  await once('ai_navigation_stop', async () => session.aiNavigationCleanup?.());

  // Flush before disposing the browser. A failure leaves its recovery owner and
  // recording buffer intact; successful stages are not repeated on retry.
  await once('audio_playback_stop', async () => {
    await session.audioPlaybackStop?.();
    session.audioPlaybackStop = undefined;
  });
  await once('recording_stop', async () => {
    if (session.pipelineManager?.isRecording()) await session.pipelineManager.stopRecording();
  });
  // Pipeline registration starts during session admission but is intentionally
  // not part of the admission response. Join it after any admitted recording
  // has flushed, before touching browser resources, so verification cannot
  // continue against a closed page/context.
  if (!session.externalTarget) {
    await once('pipeline_ready', async () => {
      await session.pipelineReadyPromise?.catch(() => undefined);
    });
    // Navigation cleanup can admit a page after the initial snapshot. Include
    // it in the same close/artifact inventory without reusing an index.
    for (const [index, page] of session.pages.entries()) {
      if (!pages.some(([, knownPage]) => knownPage === page)) pages.push([index, page]);
    }
    for (const [index, page] of pages) {
      if (videos.has(index)) continue;
      const video = page.video();
      if (video) videos.set(index, video);
    }
  }
  assertRecordingAcknowledged(session.id);
  await once('live_input_settle', async () => {
    await Promise.all([...new Set([...session.pages, session.page])].map((page) => settlePageInput(page)));
  });
  await once('page_callbacks_stop', () => {
    session.pageLifecycleCleanup?.();
    session.pageLifecycleCleanup = undefined;
    return Promise.resolve();
  });
  await once('frame_stream_stop', async () => stopFrameStreaming(session.id));
  await once('service_worker_disable', async () => session.serviceWorkerController?.disable());
  await once('accessibility_capture', async () => session.accessibilitySnapshotter?.capture(session.page));
  await once('performance_trace_stop', async () => session.perfTracer?.stop(session.page));
  if (session.tracing && session.tracePath) {
    await once('tracing_stop', async () => session.context.tracing.stop({ path: session.tracePath }));
  }
  const tracePath = session.tracePath;
  if (tracePath) await once('trace_file', () => readableArtifact(tracePath));

  if (session.externalTarget) {
    // Detach BAS's CDP connection. The target owner controls its pages/process.
    await once('cdp_detach', async () => session.browser.close());
    await session.instructionSettlement;
  } else {
    if (session.instructionInterrupted) {
      pages.sort((left, right) => Number(right[1] === session.page) - Number(left[1] === session.page));
    }
    for (const [index, page] of pages) {
      await once(`page_close:${index}`, async () => {
        if (!page.isClosed()) await page.close();
      });
      if (session.instructionInterrupted && page === session.page) {
        await session.instructionSettlement;
      }
    }
    // video.path() yields a destination, not proof that the encoder finished.
    // Context close flushes capture before any file is moved or published.
    await once('context_close', async () => session.context.close());
    for (const [index, video] of videos) {
      await once(`video_file:${index}`, async () => {
        progress.videoPaths.set(index, await moveVideo(video, session, index));
      });
    }
  }
  const harPath = session.harPath;
  if (harPath) await once('har_file', () => readableArtifact(harPath));
  const videoEntries: Array<[number, string]> = Array.from(progress.videoPaths.entries());
  videoEntries.sort((left, right) => left[0] - right[0]);
  const videoPaths: string[] = videoEntries.map((entry) => entry[1]);
  // A prior attempt may have validated a file before another stage failed.
  // Check every published reference again at the successful close boundary.
  const artifacts: string[] = [...videoPaths];
  if (tracePath) artifacts.push(tracePath);
  if (harPath) artifacts.push(harPath);
  await Promise.all(artifacts.map(readableArtifact));
  removeRecordingBuffer(session.id);
  clearFrameCache(session.id);
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
