import { randomUUID } from 'node:crypto';
import { tmpdir } from 'node:os';
import { rm } from 'node:fs/promises';
import * as path from 'node:path';
import { BaseHandler, getDocument, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getDownloadParams } from '../types';
import { DEFAULT_TIMEOUT_MS } from '../constants';
import { normalizeError, validateTimeout } from '../utils';

/** Each new invocation downloads anew; session-run owns transport retry receipts. */
export class DownloadHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['download'];
  }

  async execute(instruction: HandlerInstruction, context: HandlerContext): Promise<HandlerResult> {
    let savePath: string | undefined;
    try {
      const params = this.requireTypedParams(
        instruction.action ? getDownloadParams(instruction.action) : undefined,
        'download',
        instruction.nodeId
      );
      if (!params.selector && !params.url)
        return this.missingParamError('download', 'selector or url');
      const timeout = validateTimeout(params.timeoutMs, DEFAULT_TIMEOUT_MS, 'download');
      const target = getDocument(context);
      // Register before triggering and observe both rejections, including an event
      // timeout after a rejected click. Physical download events belong to Page.
      const [download] = await Promise.all([
        context.page.waitForEvent('download', { timeout }),
        params.selector
          ? target.click(params.selector, { timeout })
          : target.goto(params.url!, { timeout }).catch((error: unknown) => {
              // Chromium aborts navigation when its response becomes a download.
              // The joined event and save below still have to succeed.
              if (!(error instanceof Error) || !error.message.includes('net::ERR_ABORTED'))
                throw error;
            }),
      ]);
      if (!download) throw new Error('Download event not triggered');
      const filename = download.suggestedFilename();
      savePath = path.join(tmpdir(), `download-${randomUUID()}-${path.basename(filename)}`);
      await download.saveAs(savePath);
      return {
        success: true,
        extracted_data: {
          download_path: savePath,
          filename,
          url: download.url(),
        },
      };
    } catch (error) {
      if (savePath) await rm(savePath, { force: true }).catch(() => undefined);
      const normalized = normalizeError(error);
      return {
        success: false,
        error: {
          message: normalized.message,
          code: normalized.code,
          kind: normalized.kind,
          retryable: normalized.retryable,
        },
      };
    }
  }
}
