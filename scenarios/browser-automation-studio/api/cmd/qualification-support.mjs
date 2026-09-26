import { createHash } from 'node:crypto';
import { readFile } from 'node:fs/promises';

export function sha256(data) {
  return createHash('sha256').update(data).digest('hex');
}

export async function sha256File(path) {
  return sha256(await readFile(path));
}

export async function buildIdentity(apiBase) {
  const healthURL = new URL(apiBase);
  healthURL.pathname = `${healthURL.pathname.replace(/\/api\/v1\/?$/, '').replace(/\/$/, '')}/health`;
  const response = await fetch(healthURL, { signal: AbortSignal.timeout(5000) });
  if (!response.ok) throw new Error(`BAS health returned ${response.status}`);
  const health = await response.json();
  const identity = String(health.build_identity || '').trim();
  if (!identity) throw new Error('BAS health returned no build_identity');
  return identity;
}
