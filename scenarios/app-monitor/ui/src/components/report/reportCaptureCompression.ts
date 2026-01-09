/**
 * Handles compression of screenshot and element capture images for report payloads
 */

import { logger } from '@/services/logger';
import { compressBase64Image, formatBytes } from '@/utils/payloadSize';
import type { ReportIssueCapturePayload } from '@/services/api';
import type { ReportElementCapture } from './reportTypes';

const LARGE_CAPTURE_THRESHOLD = 512 * 1024; // 512 KiB

/**
 * Compresses element captures if they exceed the size threshold
 */
export async function compressElementCapturesIfNeeded(
  elementCaptures: ReportElementCapture[],
): Promise<ReportIssueCapturePayload[]> {
  return Promise.all(
    elementCaptures.map(async (capture, index) => {
      const normalizedClasses = (capture.metadata.classes ?? []).filter(Boolean);
      const createdAtIso = Number.isFinite(capture.createdAt)
        ? new Date(capture.createdAt).toISOString()
        : null;

      // Compress if image is large (>512 KiB estimated)
      let imageData = capture.data;
      const estimatedSize = imageData.length * 0.75; // Rough estimate of decoded size
      if (estimatedSize > LARGE_CAPTURE_THRESHOLD) {
        try {
          logger.info(`Compressing large element capture ${index + 1} (estimated ${formatBytes(estimatedSize)})`);
          imageData = await compressBase64Image(imageData, 1600, 1200, 0.8);
        } catch (error) {
          logger.warn(`Failed to compress element capture ${index + 1}, using original`, error);
        }
      }

      return {
        id: capture.id || `element-${index + 1}`,
        type: 'element' as const,
        width: capture.width,
        height: capture.height,
        data: imageData,
        note: capture.note.trim() || null,
        selector: capture.metadata.selector ?? null,
        tagName: capture.metadata.tagName ?? null,
        elementId: capture.metadata.elementId ?? null,
        classes: normalizedClasses.length > 0 ? normalizedClasses : null,
        label: capture.metadata.label ?? null,
        ariaDescription: capture.metadata.ariaDescription ?? null,
        title: capture.metadata.title ?? null,
        role: capture.metadata.role ?? null,
        text: capture.metadata.text ?? null,
        boundingBox: capture.metadata.boundingBox ?? null,
        clip: capture.clip ?? null,
        mode: capture.mode ?? null,
        filename: capture.filename ?? null,
        createdAt: createdAtIso,
      } satisfies ReportIssueCapturePayload;
    }),
  );
}

/**
 * Compresses primary screenshot if it exceeds the size threshold
 */
export async function compressPrimaryScreenshotIfNeeded(
  screenshotData: string,
  dimensions: { width: number; height: number } | null,
  clip: { x: number; y: number; width: number; height: number } | null,
): Promise<ReportIssueCapturePayload> {
  // Compress primary screenshot if large
  let primaryScreenshotData = screenshotData;
  const estimatedSize = screenshotData.length * 0.75;
  if (estimatedSize > LARGE_CAPTURE_THRESHOLD) {
    try {
      logger.info(`Compressing primary screenshot (estimated ${formatBytes(estimatedSize)})`);
      primaryScreenshotData = await compressBase64Image(screenshotData, 1920, 1080, 0.85);
    } catch (error) {
      logger.warn('Failed to compress primary screenshot, using original', error);
    }
  }

  return {
    id: 'page-capture',
    type: 'page' as const,
    width: dimensions?.width ?? 0,
    height: dimensions?.height ?? 0,
    data: primaryScreenshotData,
    note: null,
    selector: null,
    tagName: null,
    elementId: null,
    classes: null,
    label: 'Preview',
    ariaDescription: null,
    title: null,
    role: null,
    text: null,
    boundingBox: null,
    clip: clip ?? null,
    mode: clip ? 'clip' : 'full',
    filename: null,
    createdAt: new Date().toISOString(),
  } satisfies ReportIssueCapturePayload;
}
