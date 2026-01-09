import { createReadStream } from 'fs';
import { stat } from 'fs/promises';
import os from 'os';
import path from 'path';
import type { IncomingMessage, ServerResponse } from 'http';
import { sendJson } from '../middleware';
import { logger } from '../utils';

const allowedVideoPrefix = 'videos-';

const resolveContentType = (filePath: string): string => {
  const ext = path.extname(filePath).toLowerCase();
  switch (ext) {
    case '.webm':
      return 'video/webm';
    case '.mp4':
      return 'video/mp4';
    case '.gif':
      return 'image/gif';
    default:
      return 'application/octet-stream';
  }
};

const isAllowedPath = (filePath: string): boolean => {
  const tmpDir = path.resolve(os.tmpdir());
  const normalized = path.resolve(filePath);
  if (!normalized.startsWith(`${tmpDir}${path.sep}`)) {
    return false;
  }
  const relative = path.relative(tmpDir, normalized);
  return relative.startsWith(allowedVideoPrefix);
};

export async function handleArtifactDownload(req: IncomingMessage, res: ServerResponse): Promise<void> {
  const url = new URL(req.url || '/artifacts', 'http://localhost');
  const requestedPath = url.searchParams.get('path');
  if (!requestedPath) {
    sendJson(res, 400, {
      error: { code: 'INVALID_REQUEST', message: 'path query parameter is required' },
    });
    return;
  }

  if (!isAllowedPath(requestedPath)) {
    sendJson(res, 403, {
      error: { code: 'FORBIDDEN', message: 'artifact path not allowed' },
    });
    return;
  }

  let info;
  try {
    info = await stat(requestedPath);
  } catch (error) {
    logger.warn('artifact: stat failed', { path: requestedPath, error });
    sendJson(res, 404, {
      error: { code: 'NOT_FOUND', message: 'artifact not found' },
    });
    return;
  }

  if (!info.isFile()) {
    sendJson(res, 404, {
      error: { code: 'NOT_FOUND', message: 'artifact not found' },
    });
    return;
  }

  res.statusCode = 200;
  res.setHeader('Content-Type', resolveContentType(requestedPath));
  res.setHeader('Content-Length', info.size);

  const stream = createReadStream(requestedPath);
  stream.on('error', (error) => {
    logger.warn('artifact: stream failed', { path: requestedPath, error });
    if (!res.headersSent) {
      sendJson(res, 500, {
        error: { code: 'STREAM_ERROR', message: 'failed to stream artifact' },
      });
      return;
    }
    res.destroy();
  });
  stream.pipe(res);
}
